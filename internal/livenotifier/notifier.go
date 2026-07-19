// Package livenotifier owns the isolated GitHub issue boundary for live
// conformance. It accepts only a validated producer report or a fixed workflow
// failure envelope derived from trusted Actions metadata.
package livenotifier

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/cpaikr/opendart/internal/liveconformance"
)

const (
	DefaultReportPath = "live-conformance-report.json"

	issueTitle      = "OpenDART live conformance failure"
	issueMarker     = "<!-- opendart-live-conformance -->"
	failedMarker    = "<!-- opendart-live-conformance-state:failed -->"
	recoveredMarker = "<!-- opendart-live-conformance-state:recovered -->"
	githubAPIBase   = "https://api.github.com"
	githubWebBase   = "https://github.com"
	maximumBody     = 64 << 10
	maximumResponse = 1 << 20
	issuesPerPage   = 100
	maximumPages    = 10
)

var repositoryName = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)

// Options contains only trusted workflow metadata, the report path, and the
// job-scoped GitHub token. The OpenDART credential is deliberately absent.
type Options struct {
	ReportPath         string
	Repository         string
	ProducerConclusion string
	ArtifactOutcome    string
	RunID              uint64
	RunAttempt         uint64
	Token              string
}

// Result is the bounded notifier output. It contains no issue body or report
// data so workflow logs cannot become another persistence channel.
type Result struct {
	Action      string `json:"action"`
	IssueNumber int    `json:"issueNumber,omitempty"`
}

type observation struct {
	recovered bool
	source    string
	runURL    string
	report    liveconformance.Report
}

type issue struct {
	Number      int
	Title       string
	Body        string
	PullRequest bool
}

type issueStore interface {
	List(context.Context, string) ([]issue, error)
	Create(context.Context, string, string, string) (issue, error)
	UpdateBody(context.Context, string, int, string) (issue, error)
}

// Notify validates the producer boundary and creates or updates the one
// deduplicated issue. It never changes issue state.
func Notify(ctx context.Context, options Options) (Result, error) {
	if strings.TrimSpace(options.Token) == "" {
		return Result{}, errors.New("GitHub token is unavailable")
	}
	store := &githubStore{
		baseURL: githubAPIBase,
		token:   options.Token,
		client:  newGitHubHTTPClient(),
	}
	return notifyWith(ctx, options, store)
}

func newGitHubHTTPClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	return &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func notifyWith(ctx context.Context, options Options, store issueStore) (Result, error) {
	observed, err := inspect(options)
	if err != nil {
		return Result{}, err
	}
	issues, err := store.List(ctx, options.Repository)
	if err != nil {
		return Result{}, errors.New("list live conformance issues")
	}
	matching := make([]issue, 0, 1)
	for _, candidate := range issues {
		if !candidate.PullRequest && candidate.Title == issueTitle && strings.Contains(candidate.Body, issueMarker) {
			matching = append(matching, candidate)
		}
	}
	if len(matching) > 1 {
		return Result{}, errors.New("multiple live conformance issues exist")
	}

	if observed.recovered {
		if len(matching) == 0 {
			return Result{Action: "unchanged"}, nil
		}
		existing := matching[0]
		failed := strings.Count(existing.Body, failedMarker) == 1
		recovered := strings.Count(existing.Body, recoveredMarker) == 1
		switch {
		case failed == recovered:
			return Result{}, errors.New("live conformance issue state is invalid")
		case recovered:
			return Result{Action: "unchanged", IssueNumber: existing.Number}, nil
		}
		body, err := recoveryBody(observed.runURL)
		if err != nil {
			return Result{}, err
		}
		updated, err := store.UpdateBody(ctx, options.Repository, existing.Number, body)
		if err != nil {
			return Result{}, errors.New("record live conformance recovery")
		}
		return Result{Action: "recovered", IssueNumber: updated.Number}, nil
	}

	body, err := failureBody(observed)
	if err != nil {
		return Result{}, err
	}
	if len(matching) == 0 {
		created, err := store.Create(ctx, options.Repository, issueTitle, body)
		if err != nil {
			return Result{}, errors.New("create live conformance issue")
		}
		return Result{Action: "created", IssueNumber: created.Number}, nil
	}
	updated, err := store.UpdateBody(ctx, options.Repository, matching[0].Number, body)
	if err != nil {
		return Result{}, errors.New("update live conformance issue")
	}
	return Result{Action: "updated", IssueNumber: updated.Number}, nil
}

func inspect(options Options) (observation, error) {
	if !repositoryName.MatchString(options.Repository) || strings.Contains(options.Repository, "..") {
		return observation{}, errors.New("repository metadata is invalid")
	}
	if options.RunID == 0 || options.RunAttempt == 0 {
		return observation{}, errors.New("run metadata is invalid")
	}
	if !validProducerConclusion(options.ProducerConclusion) || !oneOf(options.ArtifactOutcome, "success", "failure", "cancelled", "skipped") {
		return observation{}, errors.New("workflow outcome metadata is invalid")
	}
	runURL := githubWebBase + "/" + options.Repository + "/actions/runs/" + strconv.FormatUint(options.RunID, 10) + "/attempts/" + strconv.FormatUint(options.RunAttempt, 10)
	fixed := observation{source: "fixed workflow failure", runURL: runURL}
	if options.ArtifactOutcome != "success" {
		return fixed, nil
	}
	file, err := os.Open(options.ReportPath)
	if err != nil {
		return fixed, nil
	}
	defer file.Close()
	report, err := liveconformance.DecodeReport(file)
	if err != nil {
		return fixed, nil
	}
	if options.ProducerConclusion == "success" && report.Outcome == "passed" {
		return observation{recovered: true, source: "validated report", runURL: runURL, report: report}, nil
	}
	if options.ProducerConclusion == "failure" && report.Outcome == "failed" {
		return observation{source: "validated report", runURL: runURL, report: report}, nil
	}
	return fixed, nil
}

func validProducerConclusion(value string) bool {
	return oneOf(value,
		"action_required",
		"cancelled",
		"failure",
		"neutral",
		"skipped",
		"stale",
		"startup_failure",
		"success",
		"timed_out",
	)
}

func failureBody(observed observation) (string, error) {
	var body strings.Builder
	fmt.Fprintf(&body, "%s\n%s\n\n# OpenDART live conformance failure\n\n- Source: %s\n- Run: %s\n", issueMarker, failedMarker, observed.source, observed.runURL)
	if observed.source == "validated report" {
		failure := observed.report.Failure
		fmt.Fprintf(&body, "- Observed at: %s\n- Failure code: `%s`\n- Failure stage: `%s`\n", observed.report.ObservedAt, failure.Code, failure.Stage)
		if failure.CaseID != "" {
			fmt.Fprintf(&body, "- Case: `%s`\n", failure.CaseID)
		}
		if failure.DiscoveryID != "" {
			fmt.Fprintf(&body, "- Discovery: `%s`\n", failure.DiscoveryID)
		}
		if failure.Operation != "" {
			fmt.Fprintf(&body, "- Operation: `%s`\n", failure.Operation)
		}
		budget := observed.report.RequestBudget
		fmt.Fprintf(&body, "- Request budget: `%d/%d`\n- Discovery budget: `%d/%d`\n", budget.Used, budget.Ceiling, budget.DiscoveryUsed, budget.DiscoveryCeiling)
	} else {
		body.WriteString("- Failure code: `workflow-failure`\n")
	}
	body.WriteString("\nThis issue is updated by trusted automation and is never closed automatically.\n")
	return boundedBody(body.String())
}

func recoveryBody(runURL string) (string, error) {
	body := fmt.Sprintf("%s\n%s\n\n# OpenDART live conformance recovered\n\n- Run: %s\n\nRecovery was recorded once. This issue remains open for maintainer review.\n", issueMarker, recoveredMarker, runURL)
	return boundedBody(body)
}

func boundedBody(body string) (string, error) {
	if len(body) == 0 || len(body) > maximumBody {
		return "", errors.New("notification body is invalid")
	}
	return body, nil
}

func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

type githubStore struct {
	baseURL string
	token   string
	client  *http.Client
}

type githubIssue struct {
	Number      int             `json:"number"`
	Title       string          `json:"title"`
	Body        string          `json:"body"`
	PullRequest json.RawMessage `json:"pull_request"`
}

func (store *githubStore) List(ctx context.Context, repository string) ([]issue, error) {
	result := make([]issue, 0)
	for page := 1; page <= maximumPages; page++ {
		query := url.Values{
			"state":    []string{"all"},
			"per_page": []string{strconv.Itoa(issuesPerPage)},
			"page":     []string{strconv.Itoa(page)},
		}
		var response []githubIssue
		if err := store.do(ctx, http.MethodGet, "/repos/"+repository+"/issues?"+query.Encode(), nil, &response); err != nil {
			return nil, err
		}
		for _, candidate := range response {
			if candidate.Number <= 0 || len(candidate.Title) > maximumBody || len(candidate.Body) > maximumBody {
				return nil, errors.New("GitHub issue response is invalid")
			}
			result = append(result, issue{Number: candidate.Number, Title: candidate.Title, Body: candidate.Body, PullRequest: len(candidate.PullRequest) != 0})
		}
		if len(response) < issuesPerPage {
			return result, nil
		}
	}
	return nil, errors.New("GitHub issue pagination limit reached")
}

func (store *githubStore) Create(ctx context.Context, repository, title, body string) (issue, error) {
	return store.write(ctx, http.MethodPost, "/repos/"+repository+"/issues", map[string]string{"title": title, "body": body})
}

func (store *githubStore) UpdateBody(ctx context.Context, repository string, number int, body string) (issue, error) {
	if number <= 0 {
		return issue{}, errors.New("GitHub issue number is invalid")
	}
	return store.write(ctx, http.MethodPatch, "/repos/"+repository+"/issues/"+strconv.Itoa(number), map[string]string{"body": body})
}

func (store *githubStore) write(ctx context.Context, method, path string, input map[string]string) (issue, error) {
	var response githubIssue
	if err := store.do(ctx, method, path, input, &response); err != nil {
		return issue{}, err
	}
	if response.Number <= 0 {
		return issue{}, errors.New("GitHub issue response is invalid")
	}
	return issue{Number: response.Number, Title: response.Title, Body: response.Body}, nil
}

func (store *githubStore) do(ctx context.Context, method, path string, input any, output any) error {
	var body io.Reader
	if input != nil {
		encoded, err := json.Marshal(input)
		if err != nil || len(encoded) > maximumBody {
			return errors.New("encode GitHub issue request")
		}
		body = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, store.baseURL+path, body)
	if err != nil {
		return errors.New("construct GitHub issue request")
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("Authorization", "Bearer "+store.token)
	request.Header.Set("X-GitHub-Api-Version", "2026-03-10")
	request.Header.Set("User-Agent", "opendart-live-conformance-notifier")
	if input != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := store.client.Do(request)
	if err != nil {
		return errors.New("perform GitHub issue request")
	}
	defer response.Body.Close()
	limited := io.LimitReader(response.Body, maximumResponse+1)
	content, err := io.ReadAll(limited)
	if err != nil || len(content) > maximumResponse {
		return errors.New("read GitHub issue response")
	}
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return errors.New("GitHub issue request was unsuccessful")
	}
	decoder := json.NewDecoder(bytes.NewReader(content))
	if err := decoder.Decode(output); err != nil {
		return errors.New("decode GitHub issue response")
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("GitHub issue response has trailing content")
	}
	return nil
}
