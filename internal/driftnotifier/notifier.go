// Package driftnotifier owns the isolated GitHub issue boundary for public
// guide drift. It accepts only a validated drift report or a fixed workflow
// failure envelope derived from trusted Actions metadata.
package driftnotifier

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

	"github.com/cpaikr/opendart/internal/guide"
)

const (
	DefaultReportPath = "guide-drift-report.json"

	issueTitle      = "OpenDART public guide semantic drift"
	issueMarker     = "<!-- opendart-guide-drift -->"
	activeMarker    = "<!-- opendart-guide-drift-state:active -->"
	recoveredMarker = "<!-- opendart-guide-drift-state:recovered -->"
	githubAPIBase   = "https://api.github.com"
	githubWebBase   = "https://github.com"
	maximumBody     = 64 << 10
	maximumResponse = 16 << 20
	issuesPerPage   = 100
	maximumPages    = 10
)

var repositoryName = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)

// Options contains only trusted workflow metadata, the report path, and the
// job-scoped GitHub token. A guide or OpenDART credential is deliberately
// absent.
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
	report    guide.DriftReport
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
	transport := &http.Transport{}
	if defaultTransport, ok := http.DefaultTransport.(*http.Transport); ok {
		transport = defaultTransport.Clone()
	}
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
		return Result{}, errors.New("list guide drift issues")
	}
	matching := make([]issue, 0, 1)
	for _, candidate := range issues {
		if !candidate.PullRequest && candidate.Title == issueTitle && strings.Contains(candidate.Body, issueMarker) {
			matching = append(matching, candidate)
		}
	}
	if len(matching) > 1 {
		return Result{}, errors.New("multiple guide drift issues exist")
	}
	var existingActive, existingRecovered bool
	if len(matching) == 1 {
		existingActive = strings.Count(matching[0].Body, activeMarker) == 1
		existingRecovered = strings.Count(matching[0].Body, recoveredMarker) == 1
		if existingActive == existingRecovered {
			return Result{}, errors.New("guide drift issue state is invalid")
		}
	}

	if observed.recovered {
		if len(matching) == 0 {
			return Result{Action: "unchanged"}, nil
		}
		existing := matching[0]
		if existingRecovered {
			return Result{Action: "unchanged", IssueNumber: existing.Number}, nil
		}
		body, err := recoveryBody(observed.runURL)
		if err != nil {
			return Result{}, err
		}
		updated, err := store.UpdateBody(ctx, options.Repository, existing.Number, body)
		if err != nil {
			return Result{}, errors.New("record guide drift recovery")
		}
		return Result{Action: "recovered", IssueNumber: updated.Number}, nil
	}

	body, err := activeBody(observed)
	if err != nil {
		return Result{}, err
	}
	if len(matching) == 0 {
		created, err := store.Create(ctx, options.Repository, issueTitle, body)
		if err != nil {
			return Result{}, errors.New("create guide drift issue")
		}
		return Result{Action: "created", IssueNumber: created.Number}, nil
	}
	updated, err := store.UpdateBody(ctx, options.Repository, matching[0].Number, body)
	if err != nil {
		return Result{}, errors.New("update guide drift issue")
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
	defer func() { _ = file.Close() }()
	report, err := guide.DecodeDriftReport(file)
	if err != nil {
		return fixed, nil
	}
	if options.ProducerConclusion == "success" {
		switch report.Outcome {
		case "unchanged":
			return observation{recovered: true, source: "validated report", runURL: runURL, report: report}, nil
		case "changed":
			return observation{source: "validated report", runURL: runURL, report: report}, nil
		}
	}
	if options.ProducerConclusion == "failure" && report.Outcome == "error" {
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

func activeBody(observed observation) (string, error) {
	var body strings.Builder
	fmt.Fprintf(&body, "%s\n%s\n\n# OpenDART public guide semantic drift\n\n- Source: %s\n- Run: %s\n", issueMarker, activeMarker, observed.source, observed.runURL)
	if observed.source != "validated report" {
		body.WriteString("- Failure code: `workflow-failure`\n")
		body.WriteString("\nThis issue is updated by trusted automation and is never closed automatically.\n")
		return boundedBody(body.String())
	}

	report := observed.report
	fmt.Fprintf(&body, "- Observed at: %s\n- Outcome: `%s`\n- Request budget: `%d/%d`\n", report.ObservedAt, report.Outcome, report.RequestBudget.Used, report.RequestBudget.Ceiling)
	switch report.Outcome {
	case "changed":
		comparison := report.Comparison
		fmt.Fprintf(&body, "- Total changes: `%d`\n- Breaking changes: `%d`\n- Findings retained: `%d`\n- Findings truncated: `%t`\n", comparison.TotalChanges, comparison.BreakingChanges, len(comparison.Findings), comparison.Truncated)
		if len(comparison.Findings) > 0 {
			body.WriteString("\n## Findings\n\n")
			for _, finding := range comparison.Findings {
				fmt.Fprintf(&body, "- `%s` at `%s`", finding.Change, finding.Location)
				if finding.Path != "" {
					fmt.Fprintf(&body, ": `%s %s` (`%s`, `%s`)", finding.Method, finding.Path, finding.LogicalOperationID, finding.OperationID)
				}
				body.WriteByte('\n')
			}
		}
	case "error":
		fmt.Fprintf(&body, "- Failure code: `%s`\n- Failure stage: `%s`\n", report.Failure.Code, report.Failure.Stage)
	default:
		return "", errors.New("guide drift observation is invalid")
	}
	body.WriteString("\nThis issue is updated by trusted automation and is never closed automatically.\n")
	return boundedBody(body.String())
}

func recoveryBody(runURL string) (string, error) {
	body := fmt.Sprintf("%s\n%s\n\n# OpenDART public guide drift recovered\n\n- Run: %s\n\nRecovery was recorded once. This issue remains open for maintainer review.\n", issueMarker, recoveredMarker, runURL)
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

type githubIssueSearch struct {
	TotalCount        int           `json:"total_count"`
	IncompleteResults bool          `json:"incomplete_results"`
	Items             []githubIssue `json:"items"`
}

func (store *githubStore) List(ctx context.Context, repository string) ([]issue, error) {
	result := make([]issue, 0)
	seen := make(map[int]struct{})
	for page := 1; page <= maximumPages; page++ {
		query := url.Values{
			"q":        []string{fmt.Sprintf("repo:%s is:issue in:title %q", repository, issueTitle)},
			"per_page": []string{strconv.Itoa(issuesPerPage)},
			"page":     []string{strconv.Itoa(page)},
		}
		var response githubIssueSearch
		if err := store.do(ctx, http.MethodGet, "/search/issues?"+query.Encode(), nil, &response); err != nil {
			return nil, err
		}
		if response.IncompleteResults || response.TotalCount < 0 || response.TotalCount > issuesPerPage*maximumPages {
			return nil, errors.New("GitHub issue search response is incomplete")
		}
		if err := appendGitHubIssues(&result, seen, response.Items); err != nil {
			return nil, err
		}
		if len(response.Items) < issuesPerPage || page*issuesPerPage >= response.TotalCount {
			break
		}
		if page == maximumPages {
			return nil, errors.New("GitHub issue search pagination limit reached")
		}
	}

	query := url.Values{
		"state":     []string{"all"},
		"sort":      []string{"created"},
		"direction": []string{"desc"},
		"per_page":  []string{strconv.Itoa(issuesPerPage)},
		"page":      []string{"1"},
	}
	var recent []githubIssue
	if err := store.do(ctx, http.MethodGet, "/repos/"+repository+"/issues?"+query.Encode(), nil, &recent); err != nil {
		return nil, err
	}
	if err := appendGitHubIssues(&result, seen, recent); err != nil {
		return nil, err
	}
	return result, nil
}

func appendGitHubIssues(result *[]issue, seen map[int]struct{}, candidates []githubIssue) error {
	for _, candidate := range candidates {
		if candidate.Number <= 0 || len(candidate.Title) > maximumBody {
			return errors.New("GitHub issue response is invalid")
		}
		if len(candidate.PullRequest) != 0 || candidate.Title != issueTitle {
			continue
		}
		if len(candidate.Body) > maximumBody {
			return errors.New("GitHub issue response is invalid")
		}
		if _, exists := seen[candidate.Number]; exists {
			continue
		}
		seen[candidate.Number] = struct{}{}
		*result = append(*result, issue{Number: candidate.Number, Title: candidate.Title, Body: candidate.Body})
	}
	return nil
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
	request.Header.Set("User-Agent", "opendart-guide-drift-notifier")
	if input != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := store.client.Do(request)
	if err != nil {
		return errors.New("perform GitHub issue request")
	}
	defer func() { _ = response.Body.Close() }()
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
