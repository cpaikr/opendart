package livenotifier

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

	"github.com/cpaikr/opendart/internal/liveconformance"
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

func TestNotifyDeduplicatesFailuresAndRecordsRecoveryOnce(t *testing.T) {
	failurePath := writeReport(t, validReport("failed"))
	store := &fakeStore{}
	options := validOptions(failurePath, "failure")

	result, err := notifyWith(context.Background(), options, store)
	if err != nil || result.Action != "created" || result.IssueNumber != 17 || store.creates != 1 || store.updates != 0 {
		t.Fatalf("first failure result = %#v, error = %v, store = %#v", result, err, store)
	}
	if !strings.Contains(store.issues[0].Body, failedMarker) || !strings.Contains(store.issues[0].Body, "credential-unavailable") {
		t.Fatalf("failure body = %q", store.issues[0].Body)
	}

	result, err = notifyWith(context.Background(), options, store)
	if err != nil || result.Action != "updated" || store.creates != 1 || store.updates != 1 || len(store.issues) != 1 {
		t.Fatalf("repeated failure result = %#v, error = %v, store = %#v", result, err, store)
	}

	options.ReportPath = writeReport(t, validReport("passed"))
	options.ProducerConclusion = "success"
	result, err = notifyWith(context.Background(), options, store)
	if err != nil || result.Action != "recovered" || store.updates != 2 || !strings.Contains(store.issues[0].Body, recoveredMarker) {
		t.Fatalf("recovery result = %#v, error = %v, store = %#v", result, err, store)
	}
	result, err = notifyWith(context.Background(), options, store)
	if err != nil || result.Action != "unchanged" || store.updates != 2 {
		t.Fatalf("repeated recovery result = %#v, error = %v, store = %#v", result, err, store)
	}
}

func TestNotifySuccessfulFirstRunDoesNotCreateIssue(t *testing.T) {
	store := &fakeStore{}
	options := validOptions(writeReport(t, validReport("passed")), "success")
	result, err := notifyWith(context.Background(), options, store)
	if err != nil || result.Action != "unchanged" || result.IssueNumber != 0 || store.creates != 0 || store.updates != 0 {
		t.Fatalf("result = %#v, error = %v, store = %#v", result, err, store)
	}
}

func TestNotifyRejectsAmbiguousRecoveryState(t *testing.T) {
	store := &fakeStore{issues: []issue{{
		Number: 17, Title: issueTitle,
		Body: issueMarker + "\n" + failedMarker + "\n" + recoveredMarker,
	}}}
	options := validOptions(writeReport(t, validReport("passed")), "success")
	if _, err := notifyWith(context.Background(), options, store); err == nil || store.updates != 0 {
		t.Fatalf("error = %v, store = %#v", err, store)
	}
}

func TestNotifyUsesOnlyFixedFallbackForUntrustedProducerInputs(t *testing.T) {
	for _, test := range []struct {
		name       string
		content    string
		conclusion string
		artifact   string
	}{
		{name: "missing artifact", conclusion: "failure", artifact: "failure"},
		{name: "invalid report", content: `{"rawBody":"secret authenticated URL"}`, conclusion: "failure", artifact: "success"},
		{name: "outcome mismatch", content: string(mustJSON(t, validReport("passed"))), conclusion: "failure", artifact: "success"},
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
			if !strings.Contains(body, "fixed workflow failure") || !strings.Contains(body, "workflow-failure") || strings.Contains(body, "secret") || strings.Contains(body, "rawBody") {
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
		_, _ = response.Write([]byte(`{"number":17,"title":"OpenDART live conformance failure","body":"updated"}`))
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

func validReport(outcome string) liveconformance.Report {
	report := liveconformance.Report{
		SchemaVersion:       liveconformance.ReportSchemaVersion,
		Kind:                liveconformance.ReportKind,
		Outcome:             outcome,
		ObservedAt:          "2026-07-19T01:02:03.000Z",
		CredentialSource:    liveconformance.CredentialSource,
		CredentialPersisted: false,
		RequestBudget:       liveconformance.RequestBudget{},
		Cases:               []liveconformance.CaseResult{},
	}
	if outcome == "failed" {
		report.Failure = &liveconformance.Failure{Code: "credential-unavailable", Stage: "credential"}
	}
	return report
}

func writeReport(t *testing.T, report liveconformance.Report) string {
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

func TestBoundedReportFallbackDoesNotReadOversizedContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "report.json")
	if err := os.WriteFile(path, bytes.Repeat([]byte("x"), liveconformance.MaximumReportBytes+1), 0o600); err != nil {
		t.Fatal(err)
	}
	store := &fakeStore{}
	if _, err := notifyWith(context.Background(), validOptions(path, "failure"), store); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(store.issues[0].Body, "fixed workflow failure") {
		t.Fatalf("body = %q", store.issues[0].Body)
	}
}
