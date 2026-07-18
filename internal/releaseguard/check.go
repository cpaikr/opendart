package releaseguard

import (
	"bytes"
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
	releasePleaseAction     = "googleapis/release-please-action@45996ed1f6d02564a971a2fa1b5860e934307cf7"

	draftRecoveryScript = `version="$(jq -r '.["."]' .release-please-manifest.json)"
tag_name="v${version}"
release="$(gh release view "${tag_name}" --json isDraft,targetCommitish 2>/dev/null || true)"
if test -n "${release}" && test "$(jq -r .isDraft <<<"${release}")" = true; then
  echo "recovering=true" >> "${GITHUB_OUTPUT}"
  echo "tag_name=${tag_name}" >> "${GITHUB_OUTPUT}"
  echo "sha=$(jq -r .targetCommitish <<<"${release}")" >> "${GITHUB_OUTPUT}"
else
  echo "recovering=false" >> "${GITHUB_OUTPUT}"
fi`
	prepareReleaseAssetsScript = `mkdir release-assets
cp openapi/generated/openapi.bundle.yaml release-assets/openapi.bundle.yaml
cd release-assets
sha256sum openapi.bundle.yaml > openapi.bundle.yaml.sha256`
	uploadReleaseAssetsScript = `for asset_path in \
  release-assets/openapi.bundle.yaml \
  release-assets/openapi.bundle.yaml.sha256
do
  asset_name="${asset_path##*/}"
  if gh release view "${TAG_NAME}" --json assets --jq '.assets[].name' | grep -Fqx "${asset_name}"; then
    mkdir -p existing-assets
    gh release download "${TAG_NAME}" --pattern "${asset_name}" --dir existing-assets
    if ! cmp -s "${asset_path}" "existing-assets/${asset_name}"; then
      echo "existing release asset differs: ${asset_name}" >&2
      exit 1
    fi
  else
    gh release upload "${TAG_NAME}" "${asset_path}"
  fi
done`
	publishReleaseScript = `gh release edit "${TAG_NAME}" --draft=false --latest`
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
	Name        string                 `yaml:"name"`
	On          map[string]any         `yaml:"on"`
	Permissions map[string]string      `yaml:"permissions"`
	Concurrency workflowConcurrency    `yaml:"concurrency"`
	Defaults    workflowDefaults       `yaml:"defaults"`
	Jobs        map[string]workflowJob `yaml:"jobs"`
}

type workflowConcurrency struct {
	Group            string `yaml:"group"`
	CancelInProgress bool   `yaml:"cancel-in-progress"`
}

type workflowDefaults struct {
	Run workflowRunDefaults `yaml:"run"`
}

type workflowRunDefaults struct {
	Shell            string `yaml:"shell"`
	WorkingDirectory string `yaml:"working-directory"`
}

type workflowJob struct {
	Needs           string            `yaml:"needs"`
	If              string            `yaml:"if"`
	ContinueOnError bool              `yaml:"continue-on-error"`
	Defaults        workflowDefaults  `yaml:"defaults"`
	Permissions     map[string]string `yaml:"permissions"`
	RunsOn          string            `yaml:"runs-on"`
	TimeoutMinutes  int               `yaml:"timeout-minutes"`
	Uses            string            `yaml:"uses"`
	Steps           []workflowStep    `yaml:"steps"`
}

type workflowStep struct {
	Name             string            `yaml:"name"`
	ID               string            `yaml:"id"`
	If               string            `yaml:"if"`
	ContinueOnError  bool              `yaml:"continue-on-error"`
	Shell            string            `yaml:"shell"`
	WorkingDirectory string            `yaml:"working-directory"`
	Uses             string            `yaml:"uses"`
	Run              string            `yaml:"run"`
	With             map[string]any    `yaml:"with"`
	Env              map[string]string `yaml:"env"`
}

func checkReleaseConfiguration(configSource, manifestSource []byte) error {
	var configFields map[string]json.RawMessage
	if err := json.Unmarshal(configSource, &configFields); err != nil {
		return &Error{Artifact: configArtifact, Invariant: "valid JSON", Cause: err}
	}
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
	if err := require(
		configArtifact,
		"contains only supported top-level options",
		reflect.DeepEqual(sortedKeys(configFields), []string{"$schema", "bootstrap-sha", "packages"}),
		"exact option allowlist is required",
	); err != nil {
		return err
	}
	root := config.Packages["."]
	if _, exists := root["releaseType"]; exists {
		return &Error{Artifact: configArtifact, Invariant: "uses kebab-case release-type"}
	}
	expectedRootKeys := []string{
		"bump-minor-pre-major",
		"bump-patch-for-minor-pre-major",
		"changelog-path",
		"draft",
		"exclude-paths",
		"force-tag-creation",
		"include-component-in-tag",
		"include-v-in-release-name",
		"include-v-in-tag",
		"package-name",
		"release-type",
	}
	if err := require(
		configArtifact,
		"root package contains only supported options",
		reflect.DeepEqual(sortedKeys(root), expectedRootKeys),
		"exact option allowlist is required",
	); err != nil {
		return err
	}

	expectedValues := []struct {
		key   string
		value any
	}{
		{key: "release-type", value: "simple"},
		{key: "package-name", value: "opendart-spec"},
		{key: "include-component-in-tag", value: false},
		{key: "include-v-in-tag", value: true},
		{key: "include-v-in-release-name", value: true},
		{key: "changelog-path", value: "CHANGELOG.md"},
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
		"go.mod", "go.sum", "internal",
	}
	if err := require(configArtifact, "root package exclude-paths", reflect.DeepEqual(root["exclude-paths"], expectedExclusions), "exact repository-only exclusions are required"); err != nil {
		return err
	}
	return nil
}

func checkWorkflows(releaseSource, verifySource []byte) error {
	release, err := decodeWorkflow(releaseWorkflowArtifact, releaseSource)
	if err != nil {
		return err
	}
	verify, err := decodeWorkflow(verifyWorkflowArtifact, verifySource)
	if err != nil {
		return err
	}

	if err := checkReleaseWorkflow(release, string(releaseSource)); err != nil {
		return err
	}
	return checkVerifyWorkflow(verify, string(verifySource))
}

func decodeWorkflow(artifact string, source []byte) (workflow, error) {
	var result workflow
	decoder := yaml.NewDecoder(bytes.NewReader(source))
	decoder.KnownFields(true)
	if err := decoder.Decode(&result); err != nil {
		return workflow{}, &Error{Artifact: artifact, Invariant: "uses only supported YAML fields", Cause: err}
	}
	return result, nil
}

func checkReleaseWorkflow(release workflow, releaseSource string) error {
	if err := require(releaseWorkflowArtifact, "has the expected workflow name", release.Name == "Release Please", ""); err != nil {
		return err
	}
	push, ok := release.On["push"].(map[string]any)
	if err := require(
		releaseWorkflowArtifact,
		"runs only for pushes to main",
		reflect.DeepEqual(sortedKeys(release.On), []string{"push"}) && ok && reflect.DeepEqual(push["branches"], []any{"main"}),
		"",
	); err != nil {
		return err
	}
	if err := require(releaseWorkflowArtifact, "serializes release runs", release.Concurrency.Group == "release-please" && !release.Concurrency.CancelInProgress, ""); err != nil {
		return err
	}
	if err := require(releaseWorkflowArtifact, "root permissions are empty", len(release.Permissions) == 0, ""); err != nil {
		return err
	}
	if err := require(releaseWorkflowArtifact, "workflow uses default run settings", defaultRunSettings(release.Defaults), ""); err != nil {
		return err
	}
	if err := require(
		releaseWorkflowArtifact,
		"contains only the verify and release-please jobs",
		reflect.DeepEqual(sortedKeys(release.Jobs), []string{"release-please", "verify"}),
		"",
	); err != nil {
		return err
	}

	verifyCall, exists := release.Jobs["verify"]
	if err := require(releaseWorkflowArtifact, "has the reusable verify job", exists && verifyCall.Uses == "./.github/workflows/verify.yml", ""); err != nil {
		return err
	}
	if err := require(releaseWorkflowArtifact, "verify job uses default execution controls", defaultJobExecution(verifyCall), ""); err != nil {
		return err
	}
	if err := require(releaseWorkflowArtifact, "verify job uses default run settings", defaultRunSettings(verifyCall.Defaults), ""); err != nil {
		return err
	}
	if err := require(releaseWorkflowArtifact, "verify job uses reusable-workflow runtime settings", verifyCall.RunsOn == "" && verifyCall.TimeoutMinutes == 0, ""); err != nil {
		return err
	}
	if err := require(releaseWorkflowArtifact, "verify job is read-only", reflect.DeepEqual(verifyCall.Permissions, map[string]string{"contents": "read"}), ""); err != nil {
		return err
	}
	releaseJob, exists := release.Jobs["release-please"]
	if err := require(releaseWorkflowArtifact, "has the release-please job", exists, ""); err != nil {
		return err
	}
	if err := require(releaseWorkflowArtifact, "release job uses default execution controls", defaultJobExecution(releaseJob), ""); err != nil {
		return err
	}
	if err := require(releaseWorkflowArtifact, "release job uses default run settings", defaultRunSettings(releaseJob.Defaults), ""); err != nil {
		return err
	}
	if err := require(releaseWorkflowArtifact, "release job uses the approved runner and timeout", releaseJob.RunsOn == "blacksmith-2vcpu-ubuntu-2404" && releaseJob.TimeoutMinutes == 20, ""); err != nil {
		return err
	}
	expectedReleaseSteps := []string{
		"Check out repository",
		"Detect interrupted draft release",
		"Run Release Please",
		"Check out released commit",
		"Prepare release assets",
		"Upload release assets",
		"Publish immutable release",
	}
	if !reflect.DeepEqual(stepNames(releaseJob.Steps), expectedReleaseSteps) {
		return &Error{Artifact: releaseWorkflowArtifact, Invariant: "uses only the approved release steps in order"}
	}
	for _, step := range releaseJob.Steps {
		if step.ContinueOnError {
			return &Error{Artifact: releaseWorkflowArtifact, Invariant: "release step failures stop the job", Detail: "step " + step.Name}
		}
		if !defaultStepRunSettings(step) {
			return &Error{Artifact: releaseWorkflowArtifact, Invariant: "release steps use default run settings", Detail: "step " + step.Name}
		}
		if !reflect.DeepEqual(step.Env, expectedReleaseStepEnv(step.Name)) {
			return &Error{Artifact: releaseWorkflowArtifact, Invariant: "release steps use only approved environment variables", Detail: "step " + step.Name}
		}
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
	if !defaultStepExecution(draft) {
		return &Error{Artifact: releaseWorkflowArtifact, Invariant: "draft detector uses default execution controls"}
	}
	releaseIndex, releaseStep, err := stepByID(releaseJob.Steps, "release")
	if err != nil {
		return &Error{Artifact: releaseWorkflowArtifact, Invariant: "has one Release Please action", Cause: err}
	}
	if err := require(releaseWorkflowArtifact, "detects drafts before Release Please", draftIndex < releaseIndex, ""); err != nil {
		return err
	}
	if !exactScript(draft.Run, draftRecoveryScript) {
		return &Error{Artifact: releaseWorkflowArtifact, Invariant: "draft recovery uses the canonical script"}
	}
	if !exactWorkflowExpression(releaseStep.If, "steps.draft.outputs.recovering != 'true'") {
		return &Error{Artifact: releaseWorkflowArtifact, Invariant: "Release Please is skipped during recovery"}
	}
	if releaseStep.Uses != releasePleaseAction || releaseStep.Run != "" {
		return &Error{Artifact: releaseWorkflowArtifact, Invariant: "uses the approved pinned Release Please action"}
	}
	if !reflect.DeepEqual(releaseStep.With, map[string]any{"token": "${{ secrets.GITHUB_TOKEN }}"}) {
		return &Error{Artifact: releaseWorkflowArtifact, Invariant: "Release Please uses GITHUB_TOKEN as its only input"}
	}

	releasedCheckoutIndex, releasedCheckout, err := stepByName(releaseJob.Steps, "Check out released commit")
	if err != nil {
		return &Error{Artifact: releaseWorkflowArtifact, Invariant: "checks out the released commit", Cause: err}
	}
	if releasedCheckoutIndex <= releaseIndex ||
		!isCheckoutAction(releasedCheckout.Uses) ||
		releasedCheckout.With["ref"] != "${{ steps.release.outputs.sha || steps.draft.outputs.sha }}" {
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
	if !exactScript(prepare.Run, prepareReleaseAssetsScript) {
		return &Error{Artifact: releaseWorkflowArtifact, Invariant: "prepares the versioned bundle and SHA-256 checksum"}
	}
	if !exactScript(upload.Run, uploadReleaseAssetsScript) {
		return &Error{Artifact: releaseWorkflowArtifact, Invariant: "uploads only the bundle and checksum; asset upload preserves immutable recovery semantics"}
	}
	if !exactScript(publish.Run, publishReleaseScript) {
		return &Error{Artifact: releaseWorkflowArtifact, Invariant: "publishes the draft only after assets are verified"}
	}
	return nil
}

func checkVerifyWorkflow(verify workflow, source string) error {
	if err := require(verifyWorkflowArtifact, "has the expected workflow name", verify.Name == "Verify", ""); err != nil {
		return err
	}
	if err := require(verifyWorkflowArtifact, "permissions are read-only", reflect.DeepEqual(verify.Permissions, map[string]string{"contents": "read"}), ""); err != nil {
		return err
	}
	if err := require(verifyWorkflowArtifact, "workflow uses default run settings", defaultRunSettings(verify.Defaults), ""); err != nil {
		return err
	}
	for _, trigger := range []string{"pull_request", "workflow_call", "workflow_dispatch"} {
		if _, exists := verify.On[trigger]; !exists {
			return &Error{Artifact: verifyWorkflowArtifact, Invariant: "supports " + trigger}
		}
	}
	if err := require(verifyWorkflowArtifact, "contains only the verify job", reflect.DeepEqual(sortedKeys(verify.Jobs), []string{"verify"}), ""); err != nil {
		return err
	}
	job, exists := verify.Jobs["verify"]
	if err := require(verifyWorkflowArtifact, "has the verify job", exists, ""); err != nil {
		return err
	}
	if err := require(verifyWorkflowArtifact, "verify job uses default execution controls", defaultJobExecution(job), ""); err != nil {
		return err
	}
	if err := require(verifyWorkflowArtifact, "verify job uses default run settings", defaultRunSettings(job.Defaults), ""); err != nil {
		return err
	}
	if err := require(verifyWorkflowArtifact, "verify job uses the approved runner and timeout", job.RunsOn == "ubuntu-latest" && job.TimeoutMinutes == 20, ""); err != nil {
		return err
	}
	for _, forbidden := range []struct {
		name    string
		pattern *regexp.Regexp
	}{
		{name: "GitHub secrets", pattern: regexp.MustCompile(`(?i)\bsecrets\s*(?:\.|\[)`)},
		{name: "GitHub token", pattern: regexp.MustCompile(`(?i)\bgithub\s*(?:\.\s*token\b|\[\s*['"]token['"]\s*\])`)},
		{name: "OpenDART API key", pattern: regexp.MustCompile(`OPENDART_API_KEY`)},
		{name: "guide synchronization", pattern: regexp.MustCompile(`sync:opendart|opendart-tool\s+sync|scripts/sync-opendart`)},
		{name: "JavaScript or Node package tooling", pattern: regexp.MustCompile(`(?i)(?:actions/setup-node@|\b(?:node|nodejs|npm|npx|corepack|yarn|pnpm|bun|deno)\b)`)},
		{name: "package publication", pattern: regexp.MustCompile(`npm publish`)},
		{name: "release asset replacement", pattern: regexp.MustCompile(`--clobber`)},
	} {
		if forbidden.pattern.MatchString(source) {
			return &Error{Artifact: verifyWorkflowArtifact, Invariant: "credential-free verification excludes " + forbidden.name}
		}
	}
	requiredCommands := []struct {
		command   string
		invariant string
	}{
		{command: "go vet ./...", invariant: "runs Go vet"},
		{command: "go test -race ./...", invariant: "runs race-enabled Go tests"},
		{command: "go run ./cmd/opendart-tool verify --repository-root .", invariant: "runs the canonical repository verification command"},
	}
	requiredActions := []struct {
		action    string
		invariant string
	}{
		{action: "actions/checkout", invariant: "checks out the repository"},
		{action: "actions/setup-go", invariant: "sets up Go"},
	}
	if len(job.Steps) != len(requiredCommands)+len(requiredActions) {
		return &Error{Artifact: verifyWorkflowArtifact, Invariant: "uses only approved Go verification steps"}
	}
	foundCommands := make(map[string]bool, len(requiredCommands))
	foundActions := make(map[string]bool, len(requiredActions))
	unapprovedStep := ""
	for _, step := range job.Steps {
		approved := false
		for _, required := range requiredCommands {
			if step.Run == required.command {
				approved = step.Uses == ""
				if !defaultStepExecution(step) {
					return &Error{Artifact: verifyWorkflowArtifact, Invariant: required.invariant + " with default execution controls"}
				}
				foundCommands[step.Run] = true
			}
		}
		for _, required := range requiredActions {
			if strings.HasPrefix(step.Uses, required.action+"@") {
				approved = step.Run == ""
				if required.action == "actions/checkout" && !reflect.DeepEqual(step.With, map[string]any{"persist-credentials": false}) {
					return &Error{Artifact: verifyWorkflowArtifact, Invariant: "checkout disables persisted credentials; checkout verifies the triggering revision"}
				}
				foundActions[required.action] = true
			}
		}
		if !approved {
			unapprovedStep = step.Name
		}
		if step.ContinueOnError {
			return &Error{Artifact: verifyWorkflowArtifact, Invariant: "verification step failures stop the job", Detail: "step " + step.Name}
		}
		if !defaultStepRunSettings(step) {
			return &Error{Artifact: verifyWorkflowArtifact, Invariant: "verification steps use default run settings", Detail: "step " + step.Name}
		}
		if len(step.Env) != 0 {
			return &Error{Artifact: verifyWorkflowArtifact, Invariant: "verification steps do not override the environment", Detail: "step " + step.Name}
		}
	}
	for _, required := range requiredCommands {
		if !foundCommands[required.command] {
			return &Error{Artifact: verifyWorkflowArtifact, Invariant: required.invariant}
		}
	}
	for _, required := range requiredActions {
		if !foundActions[required.action] {
			return &Error{Artifact: verifyWorkflowArtifact, Invariant: required.invariant}
		}
	}
	if unapprovedStep != "" {
		return &Error{Artifact: verifyWorkflowArtifact, Invariant: "uses only approved Go verification steps", Detail: "step " + unapprovedStep}
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
			if !isCheckoutAction(step.Uses) {
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
	return exactWorkflowExpression(
		condition,
		"steps.release.outputs.release_created == 'true' || steps.draft.outputs.recovering == 'true'",
	)
}

func exactWorkflowExpression(condition, expected string) bool {
	condition = strings.TrimSpace(condition)
	if strings.HasPrefix(condition, "${{") && strings.HasSuffix(condition, "}}") {
		condition = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(condition, "${{"), "}}"))
	}
	return strings.Join(strings.Fields(condition), " ") == expected
}

func defaultJobExecution(job workflowJob) bool {
	return strings.TrimSpace(job.If) == "" && !job.ContinueOnError
}

func defaultStepExecution(step workflowStep) bool {
	return strings.TrimSpace(step.If) == "" && !step.ContinueOnError
}

func defaultRunSettings(defaults workflowDefaults) bool {
	return strings.TrimSpace(defaults.Run.Shell) == "" && strings.TrimSpace(defaults.Run.WorkingDirectory) == ""
}

func defaultStepRunSettings(step workflowStep) bool {
	return strings.TrimSpace(step.Shell) == "" && strings.TrimSpace(step.WorkingDirectory) == ""
}

func stepNames(steps []workflowStep) []string {
	names := make([]string, len(steps))
	for index, step := range steps {
		names[index] = step.Name
	}
	return names
}

func expectedReleaseStepEnv(name string) map[string]string {
	switch name {
	case "Detect interrupted draft release":
		return map[string]string{"GH_TOKEN": "${{ secrets.GITHUB_TOKEN }}"}
	case "Upload release assets", "Publish immutable release":
		return map[string]string{
			"GH_TOKEN": "${{ secrets.GITHUB_TOKEN }}",
			"TAG_NAME": "${{ steps.release.outputs.tag_name || steps.draft.outputs.tag_name }}",
		}
	default:
		return nil
	}
}

func exactScript(actual, expected string) bool {
	normalize := func(script string) string {
		return strings.TrimSpace(strings.ReplaceAll(script, "\r\n", "\n"))
	}
	return normalize(actual) == normalize(expected)
}

func isCheckoutAction(action string) bool {
	return strings.HasPrefix(action, "actions/checkout@")
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
