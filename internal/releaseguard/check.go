package releaseguard

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"go.yaml.in/yaml/v4"
)

const (
	releaseWorkflowArtifact = ".github/workflows/release-please.yml"
	verifyWorkflowArtifact  = ".github/workflows/verify.yml"
	configArtifact          = "release-please-config.json"
	manifestArtifact        = ".release-please-manifest.json"
)

var (
	semanticVersion = regexp.MustCompile(`^(?:0|[1-9]\d*)\.(?:0|[1-9]\d*)\.(?:0|[1-9]\d*)$`)
	pinnedAction    = regexp.MustCompile(`^[^@]+@[0-9a-f]{40}$`)
)

// Error identifies the repository artifact and invariant that failed without
// including whole workflow sources or other potentially sensitive content.
type Error struct {
	Artifact  string
	Invariant string
	Detail    string
	Cause     error
}

func (e *Error) Error() string {
	message := fmt.Sprintf("check %s: %s", e.Artifact, e.Invariant)
	if e.Detail != "" {
		message += ": " + e.Detail
	}
	if e.Cause != nil {
		message += ": " + e.Cause.Error()
	}
	return message
}

func (e *Error) Unwrap() error {
	return e.Cause
}

// Check validates the repository's Release Please configuration and the
// permission, ordering, pinning, recovery, and release-asset workflow policy.
func Check(repositoryRoot string) error {
	if strings.TrimSpace(repositoryRoot) == "" {
		return &Error{Artifact: "repository", Invariant: "root is required"}
	}
	absoluteRoot, err := filepath.Abs(repositoryRoot)
	if err != nil {
		return &Error{Artifact: "repository", Invariant: "root resolves", Cause: err}
	}

	configSource, err := readArtifact(absoluteRoot, configArtifact)
	if err != nil {
		return err
	}
	manifestSource, err := readArtifact(absoluteRoot, manifestArtifact)
	if err != nil {
		return err
	}
	if err := checkReleaseConfiguration(configSource, manifestSource); err != nil {
		return err
	}

	releaseSource, err := readArtifact(absoluteRoot, releaseWorkflowArtifact)
	if err != nil {
		return err
	}
	verifySource, err := readArtifact(absoluteRoot, verifyWorkflowArtifact)
	if err != nil {
		return err
	}
	return checkWorkflows(releaseSource, verifySource)
}

type workflow struct {
	On          map[string]any         `yaml:"on"`
	Permissions map[string]string      `yaml:"permissions"`
	Concurrency workflowConcurrency    `yaml:"concurrency"`
	Jobs        map[string]workflowJob `yaml:"jobs"`
}

type workflowConcurrency struct {
	Group            string `yaml:"group"`
	CancelInProgress bool   `yaml:"cancel-in-progress"`
}

type workflowJob struct {
	Needs       string            `yaml:"needs"`
	Permissions map[string]string `yaml:"permissions"`
	Uses        string            `yaml:"uses"`
	Steps       []workflowStep    `yaml:"steps"`
}

type workflowStep struct {
	Name string         `yaml:"name"`
	ID   string         `yaml:"id"`
	If   string         `yaml:"if"`
	Uses string         `yaml:"uses"`
	Run  string         `yaml:"run"`
	With map[string]any `yaml:"with"`
}

func checkReleaseConfiguration(configSource, manifestSource []byte) error {
	var config struct {
		Packages map[string]map[string]any `json:"packages"`
	}
	if err := json.Unmarshal(configSource, &config); err != nil {
		return &Error{Artifact: configArtifact, Invariant: "valid JSON", Cause: err}
	}
	var manifest map[string]string
	if err := json.Unmarshal(manifestSource, &manifest); err != nil {
		return &Error{Artifact: manifestArtifact, Invariant: "valid JSON", Cause: err}
	}

	if err := require(manifestArtifact, "contains only the root component", reflect.DeepEqual(sortedKeys(manifest), []string{"."}), ""); err != nil {
		return err
	}
	if err := require(manifestArtifact, "root version is SemVer", semanticVersion.MatchString(manifest["."]), ""); err != nil {
		return err
	}
	if err := require(configArtifact, "contains only the root package", reflect.DeepEqual(sortedKeys(config.Packages), []string{"."}), ""); err != nil {
		return err
	}
	root := config.Packages["."]
	if _, exists := root["releaseType"]; exists {
		return &Error{Artifact: configArtifact, Invariant: "uses kebab-case release-type"}
	}

	expectedValues := []struct {
		key   string
		value any
	}{
		{key: "release-type", value: "simple"},
		{key: "package-name", value: "opendart-spec"},
		{key: "include-component-in-tag", value: false},
		{key: "include-v-in-tag", value: true},
		{key: "bump-minor-pre-major", value: true},
		{key: "bump-patch-for-minor-pre-major", value: true},
		{key: "draft", value: true},
		{key: "force-tag-creation", value: true},
	}
	for _, expected := range expectedValues {
		if !reflect.DeepEqual(root[expected.key], expected.value) {
			return &Error{Artifact: configArtifact, Invariant: "root package " + expected.key, Detail: fmt.Sprintf("want %v", expected.value)}
		}
	}
	expectedExclusions := []any{
		".agents", ".codex", ".github", "ARCHITECTURE.md", "cmd", "docs",
		"go.mod", "go.sum", "internal", "scripts",
	}
	if err := require(configArtifact, "root package exclude-paths", reflect.DeepEqual(root["exclude-paths"], expectedExclusions), "exact repository-only exclusions are required"); err != nil {
		return err
	}
	return nil
}

func checkWorkflows(releaseSource, verifySource []byte) error {
	var release, verify workflow
	if err := yaml.Unmarshal(releaseSource, &release); err != nil {
		return &Error{Artifact: releaseWorkflowArtifact, Invariant: "valid YAML", Cause: err}
	}
	if err := yaml.Unmarshal(verifySource, &verify); err != nil {
		return &Error{Artifact: verifyWorkflowArtifact, Invariant: "valid YAML", Cause: err}
	}

	if err := checkReleaseWorkflow(release, string(releaseSource)); err != nil {
		return err
	}
	return checkVerifyWorkflow(verify, string(verifySource))
}

func checkReleaseWorkflow(release workflow, releaseSource string) error {
	push, ok := release.On["push"].(map[string]any)
	if err := require(releaseWorkflowArtifact, "runs only for pushes to main", ok && reflect.DeepEqual(push["branches"], []any{"main"}), ""); err != nil {
		return err
	}
	if _, exists := release.On["workflow_dispatch"]; exists {
		return &Error{Artifact: releaseWorkflowArtifact, Invariant: "manual dispatch is disabled"}
	}
	if err := require(releaseWorkflowArtifact, "serializes release runs", release.Concurrency.Group == "release-please" && !release.Concurrency.CancelInProgress, ""); err != nil {
		return err
	}
	if err := require(releaseWorkflowArtifact, "root permissions are empty", len(release.Permissions) == 0, ""); err != nil {
		return err
	}

	verifyCall, exists := release.Jobs["verify"]
	if err := require(releaseWorkflowArtifact, "has the reusable verify job", exists && verifyCall.Uses == "./.github/workflows/verify.yml", ""); err != nil {
		return err
	}
	if err := require(releaseWorkflowArtifact, "verify job is read-only", reflect.DeepEqual(verifyCall.Permissions, map[string]string{"contents": "read"}), ""); err != nil {
		return err
	}
	releaseJob, exists := release.Jobs["release-please"]
	if err := require(releaseWorkflowArtifact, "has the release-please job", exists, ""); err != nil {
		return err
	}
	if err := require(releaseWorkflowArtifact, "release waits for verification", releaseJob.Needs == "verify", ""); err != nil {
		return err
	}
	expectedPermissions := map[string]string{"contents": "write", "issues": "write", "pull-requests": "write"}
	if err := require(releaseWorkflowArtifact, "release job has only required write permissions", reflect.DeepEqual(releaseJob.Permissions, expectedPermissions), ""); err != nil {
		return err
	}

	if err := checkActionPins(releaseWorkflowArtifact, release); err != nil {
		return err
	}
	if err := checkCheckoutCredentials(releaseWorkflowArtifact, release); err != nil {
		return err
	}
	if strings.Contains(releaseSource, "npm publish") || strings.Contains(releaseSource, "--clobber") {
		return &Error{Artifact: releaseWorkflowArtifact, Invariant: "does not publish packages or replace assets"}
	}

	draftIndex, draft, err := stepByID(releaseJob.Steps, "draft")
	if err != nil {
		return &Error{Artifact: releaseWorkflowArtifact, Invariant: "has one interrupted-draft detector", Cause: err}
	}
	releaseIndex, releaseStep, err := stepByID(releaseJob.Steps, "release")
	if err != nil {
		return &Error{Artifact: releaseWorkflowArtifact, Invariant: "has one Release Please action", Cause: err}
	}
	if err := require(releaseWorkflowArtifact, "detects drafts before Release Please", draftIndex < releaseIndex, ""); err != nil {
		return err
	}
	for _, fragment := range []string{"gh release view", "isDraft", "targetCommitish", "recovering=true", "recovering=false", "tag_name=", "sha="} {
		if !strings.Contains(draft.Run, fragment) {
			return &Error{Artifact: releaseWorkflowArtifact, Invariant: "draft recovery records " + fragment}
		}
	}
	if !strings.Contains(releaseStep.If, "steps.draft.outputs.recovering != 'true'") {
		return &Error{Artifact: releaseWorkflowArtifact, Invariant: "Release Please is skipped during recovery"}
	}
	if releaseStep.With["token"] != "${{ secrets.GITHUB_TOKEN }}" {
		return &Error{Artifact: releaseWorkflowArtifact, Invariant: "Release Please uses GITHUB_TOKEN"}
	}

	releasedCheckoutIndex, releasedCheckout, err := stepByName(releaseJob.Steps, "Check out released commit")
	if err != nil {
		return &Error{Artifact: releaseWorkflowArtifact, Invariant: "checks out the released commit", Cause: err}
	}
	if releasedCheckoutIndex <= releaseIndex || releasedCheckout.With["ref"] != "${{ steps.release.outputs.sha || steps.draft.outputs.sha }}" {
		return &Error{Artifact: releaseWorkflowArtifact, Invariant: "released checkout uses the created or recovered SHA"}
	}

	prepareIndex, prepare, err := stepByName(releaseJob.Steps, "Prepare release assets")
	if err != nil {
		return &Error{Artifact: releaseWorkflowArtifact, Invariant: "prepares release assets", Cause: err}
	}
	uploadIndex, upload, err := stepByName(releaseJob.Steps, "Upload release assets")
	if err != nil {
		return &Error{Artifact: releaseWorkflowArtifact, Invariant: "uploads release assets", Cause: err}
	}
	publishIndex, publish, err := stepByName(releaseJob.Steps, "Publish immutable release")
	if err != nil {
		return &Error{Artifact: releaseWorkflowArtifact, Invariant: "publishes the immutable release", Cause: err}
	}
	if err := require(
		releaseWorkflowArtifact,
		"orders release, checkout, prepare, upload, then publish",
		releaseIndex < releasedCheckoutIndex && releasedCheckoutIndex < prepareIndex && prepareIndex < uploadIndex && uploadIndex < publishIndex,
		"",
	); err != nil {
		return err
	}
	for _, step := range []workflowStep{releasedCheckout, prepare, upload, publish} {
		if !releaseOrRecovery(step.If) {
			return &Error{Artifact: releaseWorkflowArtifact, Invariant: step.Name + " runs only for a created or recovered release"}
		}
	}
	if !strings.Contains(prepare.Run, "cp openapi/generated/openapi.bundle.yaml release-assets/openapi.bundle.yaml") ||
		!strings.Contains(prepare.Run, "sha256sum openapi.bundle.yaml > openapi.bundle.yaml.sha256") {
		return &Error{Artifact: releaseWorkflowArtifact, Invariant: "prepares the versioned bundle and SHA-256 checksum"}
	}
	if !exactAssetLoop(upload.Run) {
		return &Error{Artifact: releaseWorkflowArtifact, Invariant: "uploads only the bundle and checksum"}
	}
	if strings.Count(upload.Run, "gh release upload") != 1 {
		return &Error{Artifact: releaseWorkflowArtifact, Invariant: "uploads only the bundle and checksum"}
	}
	for _, fragment := range []string{"gh release download", "cmp -s", `gh release upload "${TAG_NAME}" "${asset_path}"`} {
		if !strings.Contains(upload.Run, fragment) {
			return &Error{Artifact: releaseWorkflowArtifact, Invariant: "asset upload preserves immutable recovery semantics", Detail: "missing " + fragment}
		}
	}
	if !regexp.MustCompile(`gh release edit .*--draft=false --latest`).MatchString(publish.Run) {
		return &Error{Artifact: releaseWorkflowArtifact, Invariant: "publishes the draft only after assets are verified"}
	}
	return nil
}

func checkVerifyWorkflow(verify workflow, source string) error {
	if err := require(verifyWorkflowArtifact, "permissions are read-only", reflect.DeepEqual(verify.Permissions, map[string]string{"contents": "read"}), ""); err != nil {
		return err
	}
	for _, trigger := range []string{"pull_request", "workflow_call", "workflow_dispatch"} {
		if _, exists := verify.On[trigger]; !exists {
			return &Error{Artifact: verifyWorkflowArtifact, Invariant: "supports " + trigger}
		}
	}
	job, exists := verify.Jobs["verify"]
	if err := require(verifyWorkflowArtifact, "has the verify job", exists, ""); err != nil {
		return err
	}
	for _, forbidden := range []struct {
		name    string
		pattern *regexp.Regexp
	}{
		{name: "GitHub secrets", pattern: regexp.MustCompile(`(?i)secrets\.`)},
		{name: "OpenDART API key", pattern: regexp.MustCompile(`OPENDART_API_KEY`)},
		{name: "guide synchronization", pattern: regexp.MustCompile(`sync:opendart|opendart-tool\s+sync|scripts/sync-opendart`)},
		{name: "package publication", pattern: regexp.MustCompile(`npm publish`)},
		{name: "release asset replacement", pattern: regexp.MustCompile(`--clobber`)},
	} {
		if forbidden.pattern.MatchString(source) {
			return &Error{Artifact: verifyWorkflowArtifact, Invariant: "credential-free verification excludes " + forbidden.name}
		}
	}
	foundCommand := false
	for _, step := range job.Steps {
		if step.Run == "npm run verify:opendart" {
			foundCommand = true
		}
	}
	if !foundCommand {
		return &Error{Artifact: verifyWorkflowArtifact, Invariant: "runs the canonical repository verification command"}
	}
	if err := checkActionPins(verifyWorkflowArtifact, verify); err != nil {
		return err
	}
	if err := checkCheckoutCredentials(verifyWorkflowArtifact, verify); err != nil {
		return err
	}
	return nil
}

func checkActionPins(artifact string, workflow workflow) error {
	for jobName, job := range workflow.Jobs {
		if job.Uses != "" && !strings.HasPrefix(job.Uses, "./") && !pinnedAction.MatchString(job.Uses) {
			return &Error{Artifact: artifact, Invariant: "third-party job action is pinned to a full commit SHA", Detail: "job " + jobName}
		}
		for _, step := range job.Steps {
			if step.Uses != "" && !strings.HasPrefix(step.Uses, "./") && !pinnedAction.MatchString(step.Uses) {
				return &Error{Artifact: artifact, Invariant: "third-party step action is pinned to a full commit SHA", Detail: "step " + step.Name}
			}
		}
	}
	return nil
}

func checkCheckoutCredentials(artifact string, workflow workflow) error {
	for _, job := range workflow.Jobs {
		for _, step := range job.Steps {
			if !strings.HasPrefix(step.Uses, "actions/checkout@") {
				continue
			}
			persist, exists := step.With["persist-credentials"]
			if !exists || persist != false {
				return &Error{Artifact: artifact, Invariant: "checkout disables persisted credentials", Detail: "step " + step.Name}
			}
		}
	}
	return nil
}

func releaseOrRecovery(condition string) bool {
	return strings.Contains(condition, "steps.release.outputs.release_created == 'true'") &&
		strings.Contains(condition, "steps.draft.outputs.recovering == 'true'")
}

func exactAssetLoop(command string) bool {
	fields := strings.Fields(command)
	expected := []string{
		"for", "asset_path", "in", "\\",
		"release-assets/openapi.bundle.yaml", "\\",
		"release-assets/openapi.bundle.yaml.sha256", "do",
	}
	for index := 0; index+len(expected) <= len(fields); index++ {
		if reflect.DeepEqual(fields[index:index+len(expected)], expected) {
			return true
		}
	}
	return false
}

func stepByID(steps []workflowStep, id string) (int, workflowStep, error) {
	return findStep(steps, func(step workflowStep) bool { return step.ID == id })
}

func stepByName(steps []workflowStep, name string) (int, workflowStep, error) {
	return findStep(steps, func(step workflowStep) bool { return step.Name == name })
}

func findStep(steps []workflowStep, match func(workflowStep) bool) (int, workflowStep, error) {
	index := -1
	var found workflowStep
	for candidateIndex, step := range steps {
		if !match(step) {
			continue
		}
		if index >= 0 {
			return -1, workflowStep{}, errors.New("multiple matching steps")
		}
		index, found = candidateIndex, step
	}
	if index < 0 {
		return -1, workflowStep{}, errors.New("matching step is missing")
	}
	return index, found, nil
}

func readArtifact(root, artifact string) ([]byte, error) {
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(artifact)))
	if err != nil {
		return nil, &Error{Artifact: artifact, Invariant: "can be read", Cause: err}
	}
	return content, nil
}

func require(artifact, invariant string, condition bool, detail string) error {
	if condition {
		return nil
	}
	return &Error{Artifact: artifact, Invariant: invariant, Detail: detail}
}

func sortedKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
