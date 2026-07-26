package releaseguard

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var targetedRacePackages = []string{
	"./internal/driftnotifier",
	"./internal/guide",
	"./internal/livenotifier",
	"./internal/sdkgen",
	"./internal/sdkgen/model",
}

// These packages invoke cancellation APIs. Guide also owns direct concurrency;
// the others use cancellation only to bound sequential work. Introducing
// cancellation in another package requires an explicit ownership review.
var reviewedCancellationPackages = []string{
	"./internal/auditorprobe",
	"./internal/guide",
	"./internal/liveconformance",
	"./internal/multicompanyprobe",
	"./internal/releaseguard",
}

// These packages expose package variables that are treated as immutable
// configuration or test tables. A new package-level state owner must be
// classified before repository verification passes.
var reviewedReadOnlyGlobalPackages = []string{
	"./internal/auditorprobe",
	"./internal/crateverification",
	"./internal/liveconformance",
	"./internal/multicompanyprobe",
	"./internal/openapi",
	"./internal/releaseguard",
	"./internal/sdkgen/rust",
	"./internal/verification",
}

const verificationInstallScript = `install_workspace="${verification_tmp}/install"
CARGO_TARGET_DIR="${install_workspace}/target" cargo +1.97.1 install --locked --offline --path sdk/rust/crates/opendart-cli --root "${install_workspace}/root"
"${install_workspace}/root/bin/opendart" --version
"${install_workspace}/root/bin/opendart" operations list > /dev/null`

var credentialFreeForbiddenPatterns = []struct {
	name    string
	pattern *regexp.Regexp
}{
	{name: "GitHub secrets", pattern: regexp.MustCompile(`(?i)\bsecrets\s*(?:\.|\[)`)},
	{name: "GitHub token", pattern: regexp.MustCompile(`(?i)\bgithub\s*(?:\.\s*token\b|\[\s*['"]token['"]\s*\])`)},
	{name: "OpenDART API key", pattern: regexp.MustCompile(`OPENDART_API_KEY`)},
	{name: "guide synchronization", pattern: regexp.MustCompile(`sync:opendart|opendart-tool\s+sync|scripts/sync-opendart`)},
	{name: "JavaScript or Node package tooling", pattern: regexp.MustCompile(`(?i)(?:actions/setup-node@|\b(?:node|nodejs|npm|npx|corepack|yarn|pnpm|bun|deno)\b)`)},
	{name: "package publication", pattern: regexp.MustCompile(`(?:npm|cargo)\s+publish`)},
	{name: "registry credentials", pattern: regexp.MustCompile(`CARGO_REGISTRY_TOKEN|id-token:\s*write`)},
	{name: "release asset replacement", pattern: regexp.MustCompile(`--clobber`)},
}

func checkCredentialFreeSource(artifact, source string) error {
	for _, forbidden := range credentialFreeForbiddenPatterns {
		if forbidden.pattern.MatchString(source) {
			return &Error{Artifact: artifact, Invariant: "credential-free verification excludes " + forbidden.name}
		}
	}
	return nil
}

func checkVerificationScript(repositoryRoot string, source []byte) error {
	if err := checkCredentialFreeSource(verificationScriptArtifact, string(source)); err != nil {
		return err
	}
	info, err := os.Stat(filepath.Join(repositoryRoot, filepath.FromSlash(verificationScriptArtifact)))
	if err != nil {
		return &Error{Artifact: verificationScriptArtifact, Invariant: "is available", Cause: err}
	}
	if info.Mode().Perm()&0o111 == 0 {
		return &Error{Artifact: verificationScriptArtifact, Invariant: "is executable"}
	}
	if string(source) != expectedVerificationScript() {
		return &Error{
			Artifact:  verificationScriptArtifact,
			Invariant: "matches the reviewed fast, pull-request, and exhaustive tier contract",
		}
	}
	return nil
}

func checkFullRaceWorkflow(fullRace workflow, source string) error {
	if err := checkCredentialFreeSource(fullRaceWorkflowArtifact, source); err != nil {
		return err
	}
	if err := require(fullRaceWorkflowArtifact, "has the expected workflow name", fullRace.Name == "Full race verification", ""); err != nil {
		return err
	}
	schedule, scheduleOK := fullRace.On["schedule"].([]any)
	expectedSchedule := []any{map[string]any{"cron": "17 3 * * 1"}}
	if err := require(
		fullRaceWorkflowArtifact,
		"runs on the weekly schedule and manual dispatch only",
		reflect.DeepEqual(sortedKeys(fullRace.On), []string{"schedule", "workflow_dispatch"}) &&
			scheduleOK && reflect.DeepEqual(schedule, expectedSchedule),
		"",
	); err != nil {
		return err
	}
	if err := require(fullRaceWorkflowArtifact, "permissions are read-only", reflect.DeepEqual(fullRace.Permissions, map[string]string{"contents": "read"}), ""); err != nil {
		return err
	}
	if err := require(fullRaceWorkflowArtifact, "workflow uses default run settings", defaultRunSettings(fullRace.Defaults), ""); err != nil {
		return err
	}
	if err := require(
		fullRaceWorkflowArtifact,
		"keeps per-ref full-race runs non-cancelling",
		fullRace.Concurrency.Group == "full-race-${{ github.ref }}" &&
			!fullRace.Concurrency.CancelInProgress &&
			strings.Contains(source, "  cancel-in-progress: false"),
		"",
	); err != nil {
		return err
	}
	if err := require(fullRaceWorkflowArtifact, "contains only the full-race job", reflect.DeepEqual(sortedKeys(fullRace.Jobs), []string{"full-race"}), ""); err != nil {
		return err
	}
	job := fullRace.Jobs["full-race"]
	if !workflowNeedsExactly(job.Needs) || !defaultJobExecution(job) || job.Environment != "" || job.Uses != "" || len(job.Permissions) != 0 {
		return &Error{Artifact: fullRaceWorkflowArtifact, Invariant: "full-race job uses default execution controls"}
	}
	if !defaultRunSettings(job.Defaults) {
		return &Error{Artifact: fullRaceWorkflowArtifact, Invariant: "full-race job uses default run settings"}
	}
	if job.RunsOn != "ubuntu-latest" || job.TimeoutMinutes != 30 {
		return &Error{Artifact: fullRaceWorkflowArtifact, Invariant: "full-race job uses the approved runner and timeout"}
	}
	expectedSteps := []workflowStepExpectation{
		{
			name: "Check out repository",
			uses: "actions/checkout",
			with: map[string]any{"fetch-depth": 0, "persist-credentials": false},
		},
		{
			name: "Set up Go",
			uses: "actions/setup-go",
			with: map[string]any{"go-version-file": "go.mod", "cache": true},
		},
		{name: "Run full Go race sweep", run: "./scripts/verify full-race"},
	}
	if len(job.Steps) != len(expectedSteps) {
		return &Error{Artifact: fullRaceWorkflowArtifact, Invariant: "uses only the approved full-race steps"}
	}
	for index, want := range expectedSteps {
		step := job.Steps[index]
		if step.Name != want.name || !exactScript(step.Run, want.run) {
			return &Error{Artifact: fullRaceWorkflowArtifact, Invariant: "uses only the approved full-race steps", Detail: step.Name}
		}
		if want.uses == "" {
			if step.Uses != "" || len(step.With) != 0 {
				return &Error{Artifact: fullRaceWorkflowArtifact, Invariant: "uses only the approved full-race steps", Detail: step.Name}
			}
		} else if !strings.HasPrefix(step.Uses, want.uses+"@") || !reflect.DeepEqual(step.With, want.with) {
			return &Error{Artifact: fullRaceWorkflowArtifact, Invariant: "uses only the approved full-race actions", Detail: step.Name}
		}
		if !defaultStepExecution(step) || !defaultStepRunSettings(step) || len(step.Env) != 0 {
			return &Error{Artifact: fullRaceWorkflowArtifact, Invariant: "full-race steps use default execution controls", Detail: step.Name}
		}
	}
	if err := checkActionPins(fullRaceWorkflowArtifact, fullRace); err != nil {
		return err
	}
	return checkCheckoutCredentials(fullRaceWorkflowArtifact, fullRace)
}

func expectedVerificationScript() string {
	var source strings.Builder
	source.WriteString(`#!/bin/sh

set +x
set -eu

repository_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$repository_root"

verification_tmp=

`)
	functions := []struct {
		name string
		body string
	}{
		{name: "phase", body: `  printf 'phase: %s\n' "$1" >&2`},
		{name: "usage", body: `  printf '%s\n' \
    'description: Run repository-owned verification tiers' \
    'usage: ./scripts/verify <mode>' \
    'modes[7]{name,scope}:' \
    '  "fast go <go-test-args...>",Focused Go tests' \
    '  "fast rust <cargo-test-args...>",Focused Rust tests' \
    '  go,Required Linux Go pull-request contract' \
    '  rust,Required Linux Rust pull-request contract' \
    '  pre-push,Required Linux Go and Rust contracts' \
    '  full-race,Full Go race-detector sweep' \
    '  exhaustive,Pre-push contract plus full Go race sweep'`},
		{name: "usage_error", body: `  printf 'error: %s\n' "$1" >&2
  usage >&2
  exit 2`},
		{name: "require_no_arguments", body: `  mode=$1
  shift
  if [ "$#" -ne 0 ]; then
    usage_error "${mode} does not accept arguments"
  fi`},
		{name: "fast_go", body: `  if [ "$#" -eq 0 ]; then
    usage_error 'fast go requires at least one package or go test argument'
  fi
  phase "focused Go tests"
  go test "$@"`},
		{name: "fast_rust", body: `  if [ "$#" -eq 0 ]; then
    usage_error 'fast rust requires at least one cargo test argument'
  fi
  phase "focused Rust tests"
  cargo +1.97.1 test --locked --manifest-path sdk/rust/Cargo.toml "$@"`},
		{name: "verify_targeted_race", body: expectedTargetedRaceBody()},
		{name: "verify_go", body: `  phase "Go vet"
  go vet ./...
  phase "normal Go tests"
  go test -vet=off ./...
  phase "targeted Go race tests"
  verify_targeted_race
  phase "repository policy and artifact verification"
  go run ./cmd/opendart-tool verify --repository-root .`},
		{name: "install_rust_toolchains", body: indentScript(installRustToolchainsScript)},
		{name: "fetch_rust_dependencies", body: indentScript(fetchRustDependenciesScript)},
		{name: "verify_rust_stable", body: indentScript(stableRustVerificationScript)},
		{name: "verify_transport_independent_graph", body: indentScript(transportIndependentGraphScript)},
		{name: "verify_wasm", body: indentScript(wasmVerificationScript)},
		{name: "verify_reqwest_compatibility", body: indentScript(compatibilityVerificationScript)},
		{name: "verify_rust_msrv", body: indentScript(msrvVerificationScript)},
		{name: "verify_rust_packages", body: indentScript(packageVerificationScript)},
		{name: "verify_rust_install", body: indentScript(verificationInstallScript)},
		{name: "cleanup_rust_workspace", body: `  if [ -n "${verification_tmp}" ]; then
    rm -rf -- "${verification_tmp}"
  fi`},
		{name: "verify_rust", body: `  verification_tmp=$(mktemp -d)
  trap cleanup_rust_workspace EXIT
  trap 'exit 1' HUP INT TERM
  phase "pinned Rust toolchains"
  install_rust_toolchains
  phase "locked Rust dependency fetch"
  fetch_rust_dependencies
  phase "stable Rust contracts offline"
  verify_rust_stable
  phase "transport-independent dependency graph offline"
  verify_transport_independent_graph
  phase "WebAssembly contracts offline"
  verify_wasm
  phase "reqwest compatibility offline"
  verify_reqwest_compatibility
  phase "Rust MSRV offline"
  verify_rust_msrv
  phase "Rust package contents offline"
  verify_rust_packages
  phase "clean Rust CLI install offline"
  verify_rust_install`},
		{name: "verify_pre_push", body: `  verify_go
  verify_rust`},
		{name: "verify_full_race", body: `  phase "full Go race sweep"
  go test -race ./...`},
		{name: "verify_exhaustive", body: `  verify_pre_push
  verify_full_race`},
	}
	for _, function := range functions {
		source.WriteString(function.name)
		source.WriteString("() {\n")
		source.WriteString(function.body)
		source.WriteString("\n}\n\n")
	}
	source.WriteString(`if [ "$#" -eq 0 ]; then
  usage_error 'a mode is required'
fi

case "$1" in
  -h|--help)
    require_no_arguments "$@"
    usage
    ;;
  fast)
    shift
    if [ "$#" -eq 0 ]; then
      usage_error 'fast requires go or rust'
    fi
    language=$1
    shift
    case "$language" in
      go) fast_go "$@" ;;
      rust) fast_rust "$@" ;;
      *) usage_error "unknown fast language: ${language}" ;;
    esac
    ;;
  go)
    require_no_arguments "$@"
    verify_go
    ;;
  rust)
    require_no_arguments "$@"
    verify_rust
    ;;
  pre-push)
    require_no_arguments "$@"
    verify_pre_push
    ;;
  full-race)
    require_no_arguments "$@"
    verify_full_race
    ;;
  exhaustive)
    require_no_arguments "$@"
    verify_exhaustive
    ;;
  *)
    usage_error "unknown mode: $1"
    ;;
esac
`)
	return source.String()
}

func expectedTargetedRaceBody() string {
	var body strings.Builder
	body.WriteString(`  # Each package owns concurrency or a shared test fixture. Releaseguard's AST
  # audit keeps this explicit set aligned with concurrency-bearing Go packages.
  go test -race -vet=off \`)
	for index, packagePath := range targetedRacePackages {
		body.WriteString("\n    ")
		body.WriteString(packagePath)
		if index != len(targetedRacePackages)-1 {
			body.WriteString(` \`)
		}
	}
	return body.String()
}

func indentScript(script string) string {
	return "  " + strings.ReplaceAll(script, "\n", "\n  ")
}

func checkRaceOwnership(repositoryRoot string) error {
	directOwners := make(map[string]bool)
	cancellationOwners := make(map[string]bool)
	globalOwners := make(map[string]bool)
	err := filepath.WalkDir(repositoryRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "target", "vendor":
				if path != repositoryRoot {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		ownership, err := inspectGoOwnership(path)
		if err != nil {
			return err
		}
		if !ownership.direct && !ownership.cancellation && !ownership.global {
			return nil
		}
		directory, err := filepath.Rel(repositoryRoot, filepath.Dir(path))
		if err != nil {
			return err
		}
		packagePath := "./" + filepath.ToSlash(directory)
		if ownership.direct {
			directOwners[packagePath] = true
		}
		if ownership.cancellation {
			cancellationOwners[packagePath] = true
		}
		if ownership.global {
			globalOwners[packagePath] = true
		}
		return nil
	})
	if err != nil {
		return &Error{Artifact: verificationScriptArtifact, Invariant: "audits parseable Go concurrency ownership", Cause: err}
	}
	if actual := sortedKeys(directOwners); !reflect.DeepEqual(actual, targetedRacePackages) {
		return &Error{
			Artifact:  verificationScriptArtifact,
			Invariant: "targeted race packages match AST-discovered direct concurrency ownership",
			Detail:    "discovered: " + strings.Join(actual, ", "),
		}
	}
	expectedCancellation := append([]string(nil), reviewedCancellationPackages...)
	sort.Strings(expectedCancellation)
	if actual := sortedKeys(cancellationOwners); !reflect.DeepEqual(actual, expectedCancellation) {
		return &Error{
			Artifact:  verificationScriptArtifact,
			Invariant: "cancellation ownership matches the reviewed package classification",
			Detail:    "discovered: " + strings.Join(actual, ", "),
		}
	}
	expectedGlobals := append(append([]string(nil), targetedRacePackages...), reviewedReadOnlyGlobalPackages...)
	sort.Strings(expectedGlobals)
	if actual := sortedKeys(globalOwners); !reflect.DeepEqual(actual, expectedGlobals) {
		return &Error{
			Artifact:  verificationScriptArtifact,
			Invariant: "package-level state ownership matches the reviewed package classification",
			Detail: "discovered: " + strings.Join(actual, ", ") +
				"; after review, classify read-only owners in reviewedReadOnlyGlobalPackages",
		}
	}
	return nil
}

type goOwnership struct {
	direct       bool
	cancellation bool
	global       bool
}

func inspectGoOwnership(sourcePath string) (goOwnership, error) {
	file, err := parser.ParseFile(token.NewFileSet(), sourcePath, nil, 0)
	if err != nil {
		return goOwnership{}, err
	}
	ownership := goOwnership{}
	imports := make(map[string]string)
	for _, spec := range file.Imports {
		importPath, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			return goOwnership{}, err
		}
		name := path.Base(importPath)
		if spec.Name != nil {
			name = spec.Name.Name
		}
		imports[name] = importPath
		if importPath == "sync" || importPath == "sync/atomic" {
			ownership.direct = true
		}
	}
	for _, declaration := range file.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if ok && general.Tok == token.VAR {
			ownership.global = true
			break
		}
	}
	ast.Inspect(file, func(node ast.Node) bool {
		switch current := node.(type) {
		case *ast.GoStmt, *ast.ChanType:
			ownership.direct = true
		case *ast.CallExpr:
			selector, ok := current.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if selector.Sel.Name == "Parallel" && strings.HasSuffix(sourcePath, "_test.go") {
				ownership.direct = true
			}
			identifier, ok := selector.X.(*ast.Ident)
			if !ok {
				return true
			}
			switch imports[identifier.Name] {
			case "context":
				switch selector.Sel.Name {
				case "AfterFunc", "WithCancel", "WithCancelCause", "WithDeadline", "WithDeadlineCause", "WithTimeout", "WithTimeoutCause":
					ownership.cancellation = true
				}
			case "net/http/httptest":
				switch selector.Sel.Name {
				case "NewServer", "NewTLSServer", "NewUnstartedServer":
					ownership.direct = true
				}
			}
		}
		return true
	})
	return ownership, nil
}
