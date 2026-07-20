package driftnotifier

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cpaikr/opendart/internal/guide"
)

type fakeStore struct {
	issues  []issue
	creates int
	updates int
}

func (store *fakeStore) List(context.Context, string) ([]issue, error) {
	return append([]issue(nil), store.issues...), nil
}

func (store *fakeStore) Create(_ context.Context, _ string, title, body string) (issue, error) {
	store.creates++
	created := issue{Number: 17, Title: title, Body: body}
	store.issues = append(store.issues, created)
	return created, nil
}

func (store *fakeStore) UpdateBody(_ context.Context, _ string, number int, body string) (issue, error) {
	store.updates++
	for index := range store.issues {
		if store.issues[index].Number == number {
			store.issues[index].Body = body
			return store.issues[index], nil
		}
	}
	return issue{}, errors.New("missing issue")
}

func TestNotifyCreatesUpdatesAndRecordsRecoveryOnce(t *testing.T) {
	store := &fakeStore{}
	options := validOptions(writeReport(t, validReport("changed")), "success")

	result, err := notifyWith(context.Background(), options, store)
	if err != nil || result.Action != "created" || result.IssueNumber != 17 || store.creates != 1 || store.updates != 0 {
		t.Fatalf("first drift result = %#v, error = %v, store = %#v", result, err, store)
	}
	if !strings.Contains(store.issues[0].Body, activeMarker) || !strings.Contains(store.issues[0].Body, "Total changes: `1`") {
		t.Fatalf("drift body = %q", store.issues[0].Body)
	}

	result, err = notifyWith(context.Background(), options, store)
	if err != nil || result.Action != "updated" || store.creates != 1 || store.updates != 1 || len(store.issues) != 1 {
		t.Fatalf("repeated drift result = %#v, error = %v, store = %#v", result, err, store)
	}

	options.ReportPath = writeReport(t, validReport("unchanged"))
	result, err = notifyWith(context.Background(), options, store)
	if err != nil || result.Action != "recovered" || store.updates != 2 || !strings.Contains(store.issues[0].Body, recoveredMarker) {
		t.Fatalf("recovery result = %#v, error = %v, store = %#v", result, err, store)
	}
	result, err = notifyWith(context.Background(), options, store)
	if err != nil || result.Action != "unchanged" || store.updates != 2 {
		t.Fatalf("repeated recovery result = %#v, error = %v, store = %#v", result, err, store)
	}
	options.ReportPath = writeReport(t, validReport("changed"))
	result, err = notifyWith(context.Background(), options, store)
	if err != nil || result.Action != "updated" || store.updates != 3 || !strings.Contains(store.issues[0].Body, activeMarker) || strings.Contains(store.issues[0].Body, recoveredMarker) {
		t.Fatalf("new drift result = %#v, error = %v, store = %#v", result, err, store)
	}
}

func TestNotifyChangedThenErrorUsesSameActiveIssue(t *testing.T) {
	store := &fakeStore{}
	changed := validOptions(writeReport(t, validReport("changed")), "success")
	if _, err := notifyWith(context.Background(), changed, store); err != nil {
		t.Fatal(err)
	}
	errorOptions := validOptions(writeReport(t, validReport("error")), "failure")
	result, err := notifyWith(context.Background(), errorOptions, store)
	if err != nil || result.Action != "updated" || result.IssueNumber != 17 || store.creates != 1 || store.updates != 1 || len(store.issues) != 1 {
		t.Fatalf("error result = %#v, error = %v, store = %#v", result, err, store)
	}
	body := store.issues[0].Body
	if !strings.Contains(body, activeMarker) || strings.Contains(body, recoveredMarker) || !strings.Contains(body, "validation-failed") || !strings.Contains(body, "Outcome: `error`") {
		t.Fatalf("error body = %q", body)
	}
}

func TestNotifyUnchangedFirstRunDoesNotCreateIssue(t *testing.T) {
	store := &fakeStore{}
	options := validOptions(writeReport(t, validReport("unchanged")), "success")
	result, err := notifyWith(context.Background(), options, store)
	if err != nil || result.Action != "unchanged" || result.IssueNumber != 0 || store.creates != 0 || store.updates != 0 {
		t.Fatalf("result = %#v, error = %v, store = %#v", result, err, store)
	}
}

func TestNotifyRejectsAmbiguousRecoveryStateWithoutWriting(t *testing.T) {
	for _, body := range []string{
		issueMarker,
		issueMarker + "\n" + activeMarker + "\n" + recoveredMarker,
		issueMarker + "\n" + activeMarker + "\n" + activeMarker,
	} {
		store := &fakeStore{issues: []issue{{Number: 17, Title: issueTitle, Body: body}}}
		for _, outcome := range []string{"unchanged", "changed"} {
			options := validOptions(writeReport(t, validReport(outcome)), "success")
			if _, err := notifyWith(context.Background(), options, store); err == nil || store.creates != 0 || store.updates != 0 {
				t.Fatalf("body = %q, outcome = %q, error = %v, store = %#v", body, outcome, err, store)
			}
		}
	}
}

func TestNotifyRejectsMultipleManagedIssuesWithoutWriting(t *testing.T) {
	store := &fakeStore{issues: []issue{
		{Number: 17, Title: issueTitle, Body: issueMarker + "\n" + activeMarker},
		{Number: 18, Title: issueTitle, Body: issueMarker + "\n" + activeMarker},
	}}
	options := validOptions(writeReport(t, validReport("changed")), "success")
	if _, err := notifyWith(context.Background(), options, store); err == nil || store.creates != 0 || store.updates != 0 {
		t.Fatalf("error = %v, store = %#v", err, store)
	}
}

func TestNotifyUsesOnlyFixedFallbackForUntrustedProducerInputs(t *testing.T) {
	oversized := strings.Repeat("x", guide.MaximumDriftReportBytes+1)
	for _, test := range []struct {
		name       string
		content    string
		conclusion string
		artifact   string
	}{
		{name: "missing artifact", conclusion: "failure", artifact: "failure"},
		{name: "missing report", conclusion: "failure", artifact: "success"},
		{name: "oversized report", content: oversized, conclusion: "failure", artifact: "success"},
		{name: "invalid report", content: `{"rawBody":"secret authenticated URL"}`, conclusion: "failure", artifact: "success"},
		{name: "unchanged failure mismatch", content: string(mustJSON(t, validReport("unchanged"))), conclusion: "failure", artifact: "success"},
		{name: "changed failure mismatch", content: string(mustJSON(t, validReport("changed"))), conclusion: "failure", artifact: "success"},
		{name: "error success mismatch", content: string(mustJSON(t, validReport("error"))), conclusion: "success", artifact: "success"},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "missing.json")
			if test.content != "" {
				if err := os.WriteFile(path, []byte(test.content), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			store := &fakeStore{}
			options := validOptions(path, test.conclusion)
			options.ArtifactOutcome = test.artifact
			if _, err := notifyWith(context.Background(), options, store); err != nil {
				t.Fatal(err)
			}
			body := store.issues[0].Body
			if !strings.Contains(body, "fixed workflow failure") || !strings.Contains(body, "workflow-failure") || strings.Contains(body, "secret") || strings.Contains(body, "rawBody") || strings.Contains(body, oversized[:100]) {
				t.Fatalf("body = %q", body)
			}
		})
	}
}

func TestNotifyUsesFixedFallbackForEveryNonResultConclusion(t *testing.T) {
	for _, conclusion := range []string{
		"action_required",
		"cancelled",
		"neutral",
		"skipped",
		"stale",
		"startup_failure",
		"timed_out",
	} {
		t.Run(conclusion, func(t *testing.T) {
			store := &fakeStore{}
			options := validOptions(filepath.Join(t.TempDir(), "missing.json"), conclusion)
			options.ArtifactOutcome = "failure"
			if _, err := notifyWith(context.Background(), options, store); err != nil {
				t.Fatal(err)
			}
			if store.creates != 1 || !strings.Contains(store.issues[0].Body, "fixed workflow failure") {
				t.Fatalf("store = %#v", store)
			}
		})
	}
}

func TestFindingRenderingIsSafeAndBounded(t *testing.T) {
	report := validReport("changed")
	report.Comparison.TotalChanges = guide.MaximumDriftFindings
	report.Comparison.Findings = make([]guide.DriftFinding, guide.MaximumDriftFindings)
	for index := range report.Comparison.Findings {
		report.Comparison.Findings[index] = guide.DriftFinding{
			Change:   "modified",
			Location: "#/components/schemas/Safe" + strings.Repeat("x", 400),
		}
	}
	store := &fakeStore{}
	options := validOptions(writeReport(t, report), "success")
	if _, err := notifyWith(context.Background(), options, store); err != nil {
		t.Fatal(err)
	}
	body := store.issues[0].Body
	if len(body) > maximumBody || strings.Count(body, "- `modified` at") != guide.MaximumDriftFindings || strings.Contains(body, "raw") {
		t.Fatalf("body size = %d, finding count = %d", len(body), strings.Count(body, "- `modified` at"))
	}
}

func TestNotifyNeverChangesIssueState(t *testing.T) {
	var received map[string]json.RawMessage
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer job-token" {
			t.Error("missing job token")
		}
		content, err := io.ReadAll(request.Body)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(content, &received); err != nil {
			t.Fatal(err)
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"number":17,"title":"OpenDART public guide semantic drift","body":"updated"}`))
	}))
	defer server.Close()
	store := &githubStore{baseURL: server.URL, token: "job-token", client: server.Client()}
	if _, err := store.UpdateBody(context.Background(), "owner/repository", 17, "safe body"); err != nil {
		t.Fatal(err)
	}
	if _, exists := received["state"]; exists || len(received) != 1 || string(received["body"]) != `"safe body"` {
		t.Fatalf("request = %s", mustJSON(t, received))
	}
}

func TestGitHubErrorsDiscardResponseBodies(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusForbidden)
		_, _ = response.Write([]byte("secret authenticated URL"))
	}))
	defer server.Close()
	store := &githubStore{baseURL: server.URL, token: "job-token", client: server.Client()}
	_, err := store.List(context.Background(), "owner/repository")
	if err == nil || strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), "authenticated") {
		t.Fatalf("error = %v", err)
	}
}

func TestGitHubListCombinesSearchWithRecentIssues(t *testing.T) {
	requests := make(map[string]int)
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests[request.URL.Path]++
		response.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/search/issues":
			query := request.URL.Query()
			if !strings.Contains(query.Get("q"), "repo:owner/repository") || !strings.Contains(query.Get("q"), "is:issue") || !strings.Contains(query.Get("q"), issueTitle) {
				t.Errorf("search query = %q", query.Get("q"))
			}
			_, _ = response.Write([]byte(`{"total_count":1,"incomplete_results":false,"items":[{"number":17,"title":"OpenDART public guide semantic drift","body":"<!-- opendart-guide-drift -->"}]}`))
		case "/repos/owner/repository/issues":
			query := request.URL.Query()
			if query.Get("state") != "all" || query.Get("sort") != "created" || query.Get("direction") != "desc" || query.Get("page") != "1" {
				t.Errorf("recent query = %q", query.Encode())
			}
			recent := []map[string]any{{"number": 17, "title": issueTitle, "body": issueMarker}}
			for number := 18; number < 38; number++ {
				recent = append(recent, map[string]any{"number": number, "title": "unrelated", "body": strings.Repeat("x", maximumBody)})
			}
			if err := json.NewEncoder(response).Encode(recent); err != nil {
				t.Fatal(err)
			}
		default:
			http.NotFound(response, request)
		}
	}))
	defer server.Close()
	store := &githubStore{baseURL: server.URL, token: "job-token", client: server.Client()}
	issues, err := store.List(context.Background(), "owner/repository")
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 1 || issues[0].Number != 17 || requests["/search/issues"] != 1 || requests["/repos/owner/repository/issues"] != 1 {
		t.Fatalf("issues = %#v, requests = %#v", issues, requests)
	}
}

func TestGitHubListFindsRecentIssueBeforeSearchIndexesIt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/search/issues":
			_, _ = response.Write([]byte(`{"total_count":0,"incomplete_results":false,"items":[]}`))
		case "/repos/owner/repository/issues":
			_, _ = response.Write([]byte(`[{"number":17,"title":"OpenDART public guide semantic drift","body":"<!-- opendart-guide-drift -->"}]`))
		default:
			http.NotFound(response, request)
		}
	}))
	defer server.Close()
	store := &githubStore{baseURL: server.URL, token: "job-token", client: server.Client()}
	issues, err := store.List(context.Background(), "owner/repository")
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 1 || issues[0].Number != 17 {
		t.Fatalf("issues = %#v", issues)
	}
}

func TestGitHubClientDoesNotFollowRedirectsWithJobToken(t *testing.T) {
	redirected := false
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		redirected = true
	}))
	defer target.Close()
	source := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		http.Redirect(response, &http.Request{}, target.URL, http.StatusFound)
	}))
	defer source.Close()
	store := &githubStore{baseURL: source.URL, token: "job-token", client: newGitHubHTTPClient()}
	if _, err := store.List(context.Background(), "owner/repository"); err == nil {
		t.Fatal("List() accepted a redirect")
	}
	if redirected {
		t.Fatal("GitHub client followed a redirect")
	}
}

func validOptions(reportPath, conclusion string) Options {
	return Options{
		ReportPath: reportPath, Repository: "cpaikr/opendart",
		ProducerConclusion: conclusion, ArtifactOutcome: "success",
		RunID: 123, RunAttempt: 1,
	}
}

func validReport(outcome string) guide.DriftReport {
	report := guide.DriftReport{
		SchemaVersion: guide.DriftReportSchemaVersion,
		Kind:          guide.DriftReportKind,
		Outcome:       outcome,
		ObservedAt:    "2026-07-19T01:02:03.000Z",
		RequestBudget: guide.RequestBudget{Used: len(guide.Groups), Ceiling: len(guide.Groups)},
	}
	switch outcome {
	case "unchanged":
		report.Comparison = &guide.DriftComparison{}
	case "changed":
		report.Comparison = &guide.DriftComparison{
			TotalChanges: 1,
			Findings: []guide.DriftFinding{{
				Change: "modified", Location: "#/paths/company.json/get/responses",
			}},
		}
	case "error":
		report.RequestBudget = guide.RequestBudget{}
		report.Failure = &guide.DriftFailure{Code: "validation-failed", Stage: "validation"}
	}
	return report
}

func writeReport(t *testing.T, report guide.DriftReport) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "report.json")
	if err := os.WriteFile(path, mustJSON(t, report), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	content, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return content
}

func TestGitHubResponseLimitIsEnforced(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		_, _ = io.Copy(response, bytes.NewReader(bytes.Repeat([]byte("x"), maximumResponse+1)))
	}))
	defer server.Close()
	store := &githubStore{baseURL: server.URL, token: "job-token", client: server.Client()}
	if _, err := store.List(context.Background(), "owner/repository"); err == nil {
		t.Fatal("List() accepted an oversized response")
	}
}
