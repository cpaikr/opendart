package releaseguard

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"go.yaml.in/yaml/v4"
)

const (
	releaseWorkflowArtifact    = ".github/workflows/release-please.yml"
	verifyWorkflowArtifact     = ".github/workflows/verify.yml"
	liveWorkflowArtifact       = ".github/workflows/live-conformance.yml"
	notifyWorkflowArtifact     = ".github/workflows/live-conformance-notify.yml"
	configArtifact             = "release-please-config.json"
	manifestArtifact           = ".release-please-manifest.json"
	specificationPackagePath   = "openapi/generated"
	rustPackagePath            = "sdk/rust/crates/opendart"
	rustCLIPackagePath         = "sdk/rust/crates/opendart-cli"
	rustWorkspaceArtifact      = "sdk/rust/Cargo.toml"
	rustCargoArtifact          = "sdk/rust/crates/opendart/Cargo.toml"
	rustCLICargoArtifact       = "sdk/rust/crates/opendart-cli/Cargo.toml"
	rustLockArtifact           = "sdk/rust/Cargo.lock"
	rustProvenanceArtifact     = "sdk/rust/crates/opendart/src/provenance.rs"
	rustPackageListArtifact    = "sdk/rust/package-files.txt"
	rustCLIPackageListArtifact = "sdk/rust/opendart-cli-package-files.txt"
	canonicalBundleArtifact    = "openapi/generated/openapi.bundle.yaml"
	releasePleaseAction        = "googleapis/release-please-action@45996ed1f6d02564a971a2fa1b5860e934307cf7"
	checkoutAction             = "actions/checkout@9c091bb21b7c1c1d1991bb908d89e4e9dddfe3e0"
	setupGoAction              = "actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e"
	uploadArtifactAction       = "actions/upload-artifact@043fb46d1a93c77aae656e7c1c64a875d1fc6a0a"
	downloadArtifactAction     = "actions/download-artifact@3e5f45b2cfb9172054b4087a40e8e0b5a5461e7c"

	liveBuildScript = `mkdir -p .live-bin
go build -o .live-bin/opendart-tool ./cmd/opendart-tool
live_smoke_executable="$(
  cargo +1.97.1 test --locked --offline --manifest-path sdk/rust/Cargo.toml -p opendart-cli --test live_smoke --no-run --message-format=json |
    jq -r 'select(.reason == "compiler-artifact" and .target.name == "live_smoke" and .executable != null) | .executable' |
    tail -n 1
)"
test -n "${live_smoke_executable}"
test -x "${live_smoke_executable}"
cp "${live_smoke_executable}" .live-bin/live-smoke`
	liveRunScript = `.live-bin/opendart-tool live-conformance --repository-root . > live-conformance-report.json
.live-bin/live-smoke --exact structured_and_binary_live_paths_are_read_only_and_sanitized`
	notifyRunScript = `go run ./cmd/opendart-tool live-conformance-notify \
  --report live-conformance-report.json \
  --repository "${NOTIFY_REPOSITORY}" \
  --producer-conclusion "${NOTIFY_PRODUCER_CONCLUSION}" \
  --artifact-outcome "${NOTIFY_ARTIFACT_OUTCOME}" \
  --run-id "${NOTIFY_RUN_ID}" \
  --run-attempt "${NOTIFY_RUN_ATTEMPT}"`

	draftRecoveryScript = `version="$(jq -r '.["openapi/generated"]' .release-please-manifest.json)"
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
	publishReleaseScript        = `gh release edit "${TAG_NAME}" --draft=false --latest`
	installRustToolchainsScript = `rustup toolchain install 1.97.1 --profile minimal --component clippy --component rustfmt
rustup toolchain install 1.85.0 --profile minimal`
	fetchRustDependenciesScript = `cargo +1.97.1 fetch --locked --manifest-path sdk/rust/Cargo.toml
cargo +1.97.1 fetch --locked --manifest-path sdk/rust/compat/reqwest-feature-unification/Cargo.toml`
	stableRustVerificationScript = `cargo +1.97.1 fmt --manifest-path sdk/rust/Cargo.toml --all -- --check
cargo +1.97.1 clippy --locked --offline --manifest-path sdk/rust/Cargo.toml --workspace --all-targets --all-features -- -D warnings
cargo +1.97.1 clippy --locked --offline --manifest-path sdk/rust/Cargo.toml -p opendart --all-targets --no-default-features -- -D warnings
cargo +1.97.1 test --locked --offline --manifest-path sdk/rust/Cargo.toml --workspace --all-features
RUSTFLAGS="--cfg opendart_compat" cargo +1.97.1 test --locked --offline --manifest-path sdk/rust/Cargo.toml -p opendart-cli --test structured_loopback
RUSTFLAGS="--cfg opendart_compat" cargo +1.97.1 test --locked --offline --manifest-path sdk/rust/Cargo.toml -p opendart-cli --test binary_loopback
cargo +1.97.1 test --locked --offline --manifest-path sdk/rust/Cargo.toml -p opendart --no-default-features
cargo +1.97.1 test --locked --offline --manifest-path sdk/rust/Cargo.toml -p opendart-cli --no-default-features
RUSTDOCFLAGS="-D warnings" cargo +1.97.1 doc --locked --offline --manifest-path sdk/rust/Cargo.toml --workspace --all-features --no-deps`
	nativeArtifactFetchScript       = `cargo +1.97.1 fetch --locked --manifest-path sdk/rust/Cargo.toml`
	nativeArtifactTestScript        = `cargo +1.97.1 test --locked --offline --manifest-path sdk/rust/Cargo.toml -p opendart-cli --test binary_loopback`
	compatibilityVerificationScript = `RUSTFLAGS="--cfg opendart_compat" cargo +1.97.1 test --locked --offline --manifest-path sdk/rust/compat/reqwest-feature-unification/Cargo.toml`
	transportIndependentGraphScript = `no_default_tree="$(mktemp)"
trap 'rm -f "${no_default_tree}"' EXIT
cargo +1.97.1 tree --locked --offline --manifest-path sdk/rust/Cargo.toml -p opendart --no-default-features -e normal --prefix none > "${no_default_tree}"
if grep -Eq '^(bytes|futures-(core|io|sink|task|util)|h2|hickory-[^ ]+|http-body(-[^ ]+)?|hyper(-[^ ]+)?|native-tls|openssl(-[^ ]+)?|reqwest|ring|rustls(-[^ ]+)?|tokio(-[^ ]+)?|tower(-[^ ]+)?|trust-dns-[^ ]+|webpki(-[^ ]+)?)[[:space:]]v' "${no_default_tree}"; then
  grep -E '^(bytes|futures-(core|io|sink|task|util)|h2|hickory-[^ ]+|http-body(-[^ ]+)?|hyper(-[^ ]+)?|native-tls|openssl(-[^ ]+)?|reqwest|ring|rustls(-[^ ]+)?|tokio(-[^ ]+)?|tower(-[^ ]+)?|trust-dns-[^ ]+|webpki(-[^ ]+)?)[[:space:]]v' "${no_default_tree}"
  exit 1
fi`
	msrvVerificationScript = `cargo +1.85.0 check --locked --offline --manifest-path sdk/rust/Cargo.toml --workspace --all-targets --all-features
cargo +1.85.0 check --locked --offline --manifest-path sdk/rust/Cargo.toml -p opendart --no-default-features
cargo +1.85.0 check --locked --offline --manifest-path sdk/rust/Cargo.toml -p opendart-cli --all-targets --no-default-features
cargo +1.85.0 metadata --locked --offline --manifest-path sdk/rust/Cargo.toml --no-deps > /dev/null`
	packageVerificationScript = `sdk_package_files="$(mktemp)"
cli_package_files="$(mktemp)"
trap 'rm -f "${sdk_package_files}" "${cli_package_files}"' EXIT
cargo +1.97.1 package --locked --offline --manifest-path sdk/rust/crates/opendart/Cargo.toml --list > "${sdk_package_files}"
diff -u sdk/rust/package-files.txt "${sdk_package_files}"
cargo +1.97.1 package --locked --offline --manifest-path sdk/rust/crates/opendart-cli/Cargo.toml --list > "${cli_package_files}"
diff -u sdk/rust/opendart-cli-package-files.txt "${cli_package_files}"
cargo +1.97.1 package --workspace --locked --offline --manifest-path sdk/rust/Cargo.toml`
	sourceInstallScript = `install_workspace="$(mktemp -d)"
CARGO_TARGET_DIR="${install_workspace}/target" cargo +1.97.1 install --locked --offline --path sdk/rust/crates/opendart-cli --root "${install_workspace}/root"
"${install_workspace}/root/bin/opendart" --version
"${install_workspace}/root/bin/opendart" operations list > /dev/null`
	windowsSourceInstallScript = `$installWorkspace = Join-Path $env:RUNNER_TEMP ([guid]::NewGuid().ToString())
$installRoot = Join-Path $installWorkspace "root"
$env:CARGO_TARGET_DIR = Join-Path $installWorkspace "target"
cargo +1.97.1 install --locked --offline --path sdk/rust/crates/opendart-cli --root $installRoot
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
$binary = Join-Path $installRoot "bin/opendart.exe"
& $binary --version
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
& $binary operations list | Out-Null
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }`
)

var canonicalSpecificationSources = []string{
	"openapi/openapi.yaml",
	"openapi/components",
	"openapi/paths",
	"openapi/schemas",
}

var (
	semanticVersion = regexp.MustCompile(`^(?:0|[1-9]\d*)\.(?:0|[1-9]\d*)\.(?:0|[1-9]\d*)(?:-(?:0|[1-9]\d*|\d*[A-Za-z-][0-9A-Za-z-]*)(?:\.(?:0|[1-9]\d*|\d*[A-Za-z-][0-9A-Za-z-]*))*)?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$`)
	pinnedAction    = regexp.MustCompile(`^[^@]+@[0-9a-f]{40}$`)
	rustBundleSHA   = regexp.MustCompile(`(?m)^const CANONICAL_BUNDLE_SHA256: &str =\s*"([0-9a-f]{64})";$`)
	rustSourceTag   = regexp.MustCompile(`(?m)^const SPECIFICATION_SOURCE_RELEASE: Option<&str> = Some\("(v[0-9]+\.[0-9]+\.[0-9]+)"\);$`)
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

// Check validates the repository's release, credential-free verification, and
// protected live-automation workflow policies.
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
	cargoSource, err := readArtifact(absoluteRoot, rustCargoArtifact)
	if err != nil {
		return err
	}
	lockSource, err := readArtifact(absoluteRoot, rustLockArtifact)
	if err != nil {
		return err
	}
	cliCargoSource, err := readArtifact(absoluteRoot, rustCLICargoArtifact)
	if err != nil {
		return err
	}
	workspaceSource, err := readArtifact(absoluteRoot, rustWorkspaceArtifact)
	if err != nil {
		return err
	}
	if err := checkReleaseConfiguration(configSource, manifestSource, cargoSource, cliCargoSource, lockSource); err != nil {
		return err
	}
	provenanceSource, err := readArtifact(absoluteRoot, rustProvenanceArtifact)
	if err != nil {
		return err
	}
	packageListSource, err := readArtifact(absoluteRoot, rustPackageListArtifact)
	if err != nil {
		return err
	}
	bundleSource, err := readArtifact(absoluteRoot, canonicalBundleArtifact)
	if err != nil {
		return err
	}
	if err := checkSpecificationSourceRelease(absoluteRoot, provenanceSource); err != nil {
		return err
	}
	if err := checkRustPackage(cargoSource, provenanceSource, packageListSource, bundleSource); err != nil {
		return err
	}
	cliPackageListSource, err := readArtifact(absoluteRoot, rustCLIPackageListArtifact)
	if err != nil {
		return err
	}
	if err := checkRustCLIPackage(cliCargoSource, workspaceSource, lockSource, cliPackageListSource); err != nil {
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
	liveSource, err := readArtifact(absoluteRoot, liveWorkflowArtifact)
	if err != nil {
		return err
	}
	notifySource, err := readArtifact(absoluteRoot, notifyWorkflowArtifact)
	if err != nil {
		return err
	}
	return checkWorkflows(releaseSource, verifySource, liveSource, notifySource)
}

func checkSpecificationSourceRelease(repositoryRoot string, provenanceSource []byte) error {
	matches := rustSourceTag.FindAllSubmatch(provenanceSource, -1)
	if len(matches) != 1 {
		return &Error{
			Artifact:  rustProvenanceArtifact,
			Invariant: "names one semantic specification source release",
			Detail:    "one active vX.Y.Z source-release constant is required",
		}
	}
	tag := string(matches[0][1])
	reference := "refs/tags/" + tag
	if err := exec.Command("git", "-C", repositoryRoot, "rev-parse", "--verify", "--quiet", reference+"^{commit}").Run(); err != nil {
		return &Error{
			Artifact:  rustProvenanceArtifact,
			Invariant: "references an available specification source-release tag",
			Detail:    tag,
			Cause:     err,
		}
	}
	for _, source := range canonicalSpecificationSources {
		if err := exec.Command("git", "-C", repositoryRoot, "cat-file", "-e", reference+":"+source).Run(); err != nil {
			return &Error{
				Artifact:  rustProvenanceArtifact,
				Invariant: "semantic source release contains the canonical specification inputs",
				Detail:    tag + ":" + source,
				Cause:     err,
			}
		}
	}
	return nil
}

func checkRustPackage(cargoSource, provenanceSource, packageListSource, bundleSource []byte) error {
	publish, err := cargoPackageStringArray(cargoSource, "publish")
	if err != nil || !reflect.DeepEqual(publish, []string{"crates-io"}) {
		return &Error{Artifact: rustCargoArtifact, Invariant: "authorizes only the crates.io registry", Cause: err}
	}
	include, err := cargoPackageStringArray(cargoSource, "include")
	if err != nil {
		return &Error{Artifact: rustCargoArtifact, Invariant: "packages release documentation and provenance", Cause: err}
	}
	includeSet := make(map[string]bool, len(include))
	for _, path := range include {
		includeSet[path] = true
	}
	if err := require(rustCargoArtifact, "packages release documentation and provenance", includeSet["CHANGELOG.md"] && includeSet["src/**"], ""); err != nil {
		return err
	}
	bundleChecksum := fmt.Sprintf("%x", sha256.Sum256(bundleSource))
	checksumMatches := rustBundleSHA.FindAllSubmatch(provenanceSource, -1)
	provenanceMatches := len(checksumMatches) == 1 && string(checksumMatches[0][1]) == bundleChecksum
	if err := require(rustProvenanceArtifact, "matches the canonical bundle SHA-256", provenanceMatches, "one active checksum constant is required"); err != nil {
		return err
	}

	lines := strings.Fields(string(packageListSource))
	if !sort.StringsAreSorted(lines) {
		return &Error{Artifact: rustPackageListArtifact, Invariant: "is sorted for deterministic comparison"}
	}
	for _, name := range []string{
		".cargo_vcs_info.json",
		"CHANGELOG.md",
		"Cargo.toml",
		"LICENSE",
		"README.md",
		"src/generated/.opendart-sdk-generated",
		"src/generated/operations/mod.rs",
		"src/lib.rs",
		"src/provenance.rs",
	} {
		if !sortedContains(lines, name) {
			return &Error{Artifact: rustPackageListArtifact, Invariant: "contains required package evidence", Detail: name}
		}
	}
	return checkPackageInventoryPrivateInputs(rustPackageListArtifact, lines)
}

func checkRustCLIPackage(cargoSource, workspaceSource, lockSource, packageListSource []byte) error {
	publish, err := cargoPackageStringArray(cargoSource, "publish")
	if err != nil || !reflect.DeepEqual(publish, []string{"crates-io"}) {
		return &Error{Artifact: rustCLICargoArtifact, Invariant: "authorizes only the crates.io registry", Cause: err}
	}
	include, err := cargoPackageStringArray(cargoSource, "include")
	if err != nil {
		return &Error{Artifact: rustCLICargoArtifact, Invariant: "packages the reviewed source distribution", Cause: err}
	}
	approvedIncludes := map[string]bool{
		"src/**": true, "tests/**": true, "Cargo.toml": true, "Cargo.lock": true,
		"README.md": true, "CHANGELOG.md": true, "LICENSE": true,
	}
	if len(include) != len(approvedIncludes) {
		return &Error{Artifact: rustCLICargoArtifact, Invariant: "packages the reviewed source distribution", Detail: "exact include allowlist is required"}
	}
	seenIncludes := make(map[string]bool, len(include))
	for _, name := range include {
		if !approvedIncludes[name] || seenIncludes[name] {
			return &Error{Artifact: rustCLICargoArtifact, Invariant: "packages the reviewed source distribution", Detail: name}
		}
		seenIncludes[name] = true
	}
	sdkVersion, err := cargoLockPackageVersion(lockSource, "opendart")
	if err != nil {
		return &Error{Artifact: rustLockArtifact, Invariant: "contains one opendart package version", Cause: err}
	}
	pin, err := cargoInlineDependencyVersion(cargoSource, "opendart")
	if err != nil || pin != "="+sdkVersion {
		return &Error{Artifact: rustCLICargoArtifact, Invariant: "exact-pins the workspace SDK version", Cause: err}
	}
	if !bytes.Contains(cargoSource, []byte(`opendart = { path = "../opendart", version = "=`+sdkVersion+`", default-features = false, features = ["client-reqwest", "serde-json"] } # x-release-please-version`)) {
		return &Error{Artifact: rustCLICargoArtifact, Invariant: "marks the exact SDK pin for SDK-owned release updates"}
	}
	if !bytes.Contains(cargoSource, []byte("serde_json.workspace = true")) {
		return &Error{Artifact: rustCLICargoArtifact, Invariant: "inherits the reviewed JSON encoder behavior"}
	}
	if !bytes.Contains(workspaceSource, []byte(`serde_json = { version = "=1.0.150", features = ["arbitrary_precision", "preserve_order"] }`)) {
		return &Error{Artifact: rustWorkspaceArtifact, Invariant: "exact-pins the reviewed JSON encoder behavior"}
	}

	lines := strings.Fields(string(packageListSource))
	if !sort.StringsAreSorted(lines) {
		return &Error{Artifact: rustCLIPackageListArtifact, Invariant: "is sorted for deterministic comparison"}
	}
	for _, name := range []string{
		".cargo_vcs_info.json",
		"CHANGELOG.md",
		"Cargo.lock",
		"Cargo.toml",
		"LICENSE",
		"README.md",
		"src/generated/.opendart-cli-generated",
		"src/generated/catalog.rs",
		"src/generated/command.rs",
		"src/generated/dispatch.rs",
		"src/main.rs",
		"tests/binary_loopback.rs",
		"tests/common/mod.rs",
		"tests/discovery.rs",
		"tests/fixtures/invalid-invocation.json",
		"tests/fixtures/missing-api-key.json",
		"tests/live_smoke.rs",
		"tests/structured_loopback.rs",
	} {
		if !sortedContains(lines, name) {
			return &Error{Artifact: rustCLIPackageListArtifact, Invariant: "contains required package evidence", Detail: name}
		}
	}
	return checkPackageInventoryPrivateInputs(rustCLIPackageListArtifact, lines)
}

func cargoInlineDependencyVersion(source []byte, dependency string) (string, error) {
	inDependencies := false
	prefix := dependency + " = "
	versionPattern := regexp.MustCompile(`\bversion\s*=\s*"([^"]+)"`)
	for _, line := range strings.Split(string(source), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") {
			inDependencies = trimmed == "[dependencies]"
			continue
		}
		if inDependencies && strings.HasPrefix(trimmed, prefix) {
			matches := versionPattern.FindStringSubmatch(trimmed)
			if len(matches) == 2 {
				return matches[1], nil
			}
			return "", errors.New("dependency version is missing")
		}
	}
	return "", fmt.Errorf("dependency %q is missing", dependency)
}

func checkPackageInventoryPrivateInputs(artifact string, lines []string) error {
	for _, name := range lines {
		for _, prefix := range []string{".github/", "compat/", "internal/", "openapi/", "target/"} {
			if strings.HasPrefix(name, prefix) {
				return &Error{Artifact: artifact, Invariant: "excludes repository-private inputs", Detail: name}
			}
		}
	}
	return nil
}

func sortedContains(values []string, target string) bool {
	index := sort.SearchStrings(values, target)
	return index < len(values) && values[index] == target
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
	Environment     string            `yaml:"environment"`
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

func checkReleaseConfiguration(configSource, manifestSource, cargoSource, cliCargoSource, lockSource []byte) error {
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

	manifestKeys := sortedKeys(manifest)
	bootstrapManifest := reflect.DeepEqual(manifestKeys, []string{specificationPackagePath})
	if err := require(manifestArtifact, "contains only the published specification component", bootstrapManifest, "unpublished Rust components must remain absent"); err != nil {
		return err
	}
	if err := require(manifestArtifact, "specification version is SemVer", semanticVersion.MatchString(manifest[specificationPackagePath]), ""); err != nil {
		return err
	}
	if err := require(configArtifact, "contains only the specification, SDK, and CLI packages", reflect.DeepEqual(sortedKeys(config.Packages), []string{specificationPackagePath, rustPackagePath, rustCLIPackagePath}), ""); err != nil {
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
	root := config.Packages[specificationPackagePath]
	if _, exists := root["releaseType"]; exists {
		return &Error{Artifact: configArtifact, Invariant: "uses kebab-case release-type"}
	}
	expectedRootKeys := []string{
		"bump-minor-pre-major",
		"bump-patch-for-minor-pre-major",
		"changelog-path",
		"draft",
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
		{key: "changelog-path", value: "/CHANGELOG.md"},
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
	rustPackage := config.Packages[rustPackagePath]
	expectedRustKeys := []string{
		"bump-minor-pre-major",
		"bump-patch-for-minor-pre-major",
		"changelog-path",
		"component",
		"draft",
		"extra-files",
		"force-tag-creation",
		"include-component-in-tag",
		"include-v-in-release-name",
		"include-v-in-tag",
		"release-type",
	}
	if err := require(configArtifact, "Rust package contains only supported options", reflect.DeepEqual(sortedKeys(rustPackage), expectedRustKeys), "exact option allowlist is required"); err != nil {
		return err
	}
	expectedRustValues := map[string]any{
		"release-type":                   "rust",
		"component":                      "opendart",
		"include-component-in-tag":       true,
		"include-v-in-tag":               true,
		"include-v-in-release-name":      true,
		"changelog-path":                 "CHANGELOG.md",
		"bump-minor-pre-major":           true,
		"bump-patch-for-minor-pre-major": true,
		"draft":                          true,
		"force-tag-creation":             false,
	}
	for key, value := range expectedRustValues {
		if !reflect.DeepEqual(rustPackage[key], value) {
			return &Error{Artifact: configArtifact, Invariant: "Rust package " + key, Detail: fmt.Sprintf("want %v", value)}
		}
	}
	expectedExtraFiles := []any{
		map[string]any{
			"type":     "toml",
			"path":     "/sdk/rust/Cargo.lock",
			"jsonpath": `$.package[?(@.name == "opendart")].version`,
		},
		map[string]any{
			"type": "generic",
			"path": "/sdk/rust/crates/opendart-cli/Cargo.toml",
		},
	}
	if err := require(configArtifact, "Rust package updates the workspace lock and CLI SDK pin", reflect.DeepEqual(rustPackage["extra-files"], expectedExtraFiles), "exact root-relative updaters are required"); err != nil {
		return err
	}

	cliPackage := config.Packages[rustCLIPackagePath]
	if err := require(configArtifact, "CLI package contains only supported options", reflect.DeepEqual(sortedKeys(cliPackage), expectedRustKeys), "exact option allowlist is required"); err != nil {
		return err
	}
	expectedCLIValues := make(map[string]any, len(expectedRustValues))
	for key, value := range expectedRustValues {
		expectedCLIValues[key] = value
	}
	expectedCLIValues["component"] = "opendart-cli"
	for key, value := range expectedCLIValues {
		if !reflect.DeepEqual(cliPackage[key], value) {
			return &Error{Artifact: configArtifact, Invariant: "CLI package " + key, Detail: fmt.Sprintf("want %v", value)}
		}
	}
	expectedCLIExtraFiles := []any{map[string]any{
		"type":     "toml",
		"path":     "/sdk/rust/Cargo.lock",
		"jsonpath": `$.package[?(@.name == "opendart-cli")].version`,
	}}
	if err := require(configArtifact, "CLI package updates its workspace lock version", reflect.DeepEqual(cliPackage["extra-files"], expectedCLIExtraFiles), "exact root-relative TOML updater is required"); err != nil {
		return err
	}

	cargoVersion, err := cargoPackageVersion(cargoSource)
	if err != nil {
		return &Error{Artifact: rustCargoArtifact, Invariant: "declares one package version", Cause: err}
	}
	lockVersion, err := cargoLockPackageVersion(lockSource, "opendart")
	if err != nil {
		return &Error{Artifact: rustLockArtifact, Invariant: "contains one opendart package version", Cause: err}
	}
	if err := require(rustLockArtifact, "matches the crate package version", cargoVersion == lockVersion, ""); err != nil {
		return err
	}
	cliCargoVersion, err := cargoPackageVersion(cliCargoSource)
	if err != nil {
		return &Error{Artifact: rustCLICargoArtifact, Invariant: "declares one package version", Cause: err}
	}
	cliLockVersion, err := cargoLockPackageVersion(lockSource, "opendart-cli")
	if err != nil {
		return &Error{Artifact: rustLockArtifact, Invariant: "contains one opendart-cli package version", Cause: err}
	}
	if err := require(rustLockArtifact, "matches the CLI crate package version", cliCargoVersion == cliLockVersion, ""); err != nil {
		return err
	}
	return nil
}

func cargoPackageVersion(source []byte) (string, error) {
	inPackage := false
	for _, line := range strings.Split(string(source), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") {
			inPackage = trimmed == "[package]"
			continue
		}
		if inPackage && strings.HasPrefix(trimmed, "version = ") {
			return quotedTOMLValue(trimmed)
		}
	}
	return "", errors.New("package version is missing")
}

func cargoPackageStringArray(source []byte, key string) ([]string, error) {
	inPackage := false
	prefix := key + " = "
	for _, line := range strings.Split(string(source), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") {
			inPackage = trimmed == "[package]"
			continue
		}
		if inPackage && strings.HasPrefix(trimmed, prefix) {
			var values []string
			if err := json.Unmarshal([]byte(strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))), &values); err != nil {
				return nil, err
			}
			return values, nil
		}
	}
	return nil, fmt.Errorf("package %s array is missing", key)
}

func cargoLockPackageVersion(source []byte, packageName string) (string, error) {
	var name, version string
	var matches []string
	flush := func() {
		if name == packageName {
			matches = append(matches, version)
		}
	}
	for _, line := range strings.Split(string(source), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "[[package]]" {
			flush()
			name, version = "", ""
			continue
		}
		if strings.HasPrefix(trimmed, "name = ") {
			name, _ = quotedTOMLValue(trimmed)
		}
		if strings.HasPrefix(trimmed, "version = ") {
			version, _ = quotedTOMLValue(trimmed)
		}
	}
	flush()
	if len(matches) != 1 || matches[0] == "" {
		return "", fmt.Errorf("package %q must appear exactly once with a version", packageName)
	}
	return matches[0], nil
}

func quotedTOMLValue(line string) (string, error) {
	_, value, ok := strings.Cut(line, "=")
	if !ok {
		return "", errors.New("value assignment is malformed")
	}
	value = strings.TrimSpace(value)
	if len(value) < 2 || value[0] != '"' || value[len(value)-1] != '"' {
		return "", errors.New("value is not a quoted string")
	}
	return value[1 : len(value)-1], nil
}

func checkWorkflows(releaseSource, verifySource, liveSource, notifySource []byte) error {
	release, err := decodeWorkflow(releaseWorkflowArtifact, releaseSource)
	if err != nil {
		return err
	}
	verify, err := decodeWorkflow(verifyWorkflowArtifact, verifySource)
	if err != nil {
		return err
	}
	live, err := decodeWorkflow(liveWorkflowArtifact, liveSource)
	if err != nil {
		return err
	}
	notify, err := decodeWorkflow(notifyWorkflowArtifact, notifySource)
	if err != nil {
		return err
	}

	if err := checkReleaseWorkflow(release, string(releaseSource)); err != nil {
		return err
	}
	if err := checkVerifyWorkflow(verify, string(verifySource)); err != nil {
		return err
	}
	if err := checkLiveWorkflow(live, string(liveSource)); err != nil {
		return err
	}
	return checkNotifyWorkflow(notify, string(notifySource))
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
	if strings.Contains(releaseSource, "npm publish") || strings.Contains(releaseSource, "cargo publish") ||
		strings.Contains(releaseSource, "CARGO_REGISTRY_TOKEN") || strings.Contains(releaseSource, "id-token: write") ||
		strings.Contains(releaseSource, "--clobber") {
		return &Error{Artifact: releaseWorkflowArtifact, Invariant: "does not publish packages, grant registry authority, or replace assets"}
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
		releasedCheckout.With["ref"] != "${{ steps.release.outputs['openapi/generated--sha'] || steps.draft.outputs.sha }}" {
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
	if err := require(verifyWorkflowArtifact, "contains only approved verification jobs", reflect.DeepEqual(sortedKeys(verify.Jobs), []string{"artifact-macos", "artifact-windows", "verify"}), ""); err != nil {
		return err
	}
	for _, native := range []struct {
		name   string
		runner string
	}{
		{name: "artifact-macos", runner: "macos-latest"},
		{name: "artifact-windows", runner: "windows-latest"},
	} {
		if err := checkNativeArtifactJob(native.name, native.runner, verify.Jobs[native.name]); err != nil {
			return err
		}
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
	if err := require(verifyWorkflowArtifact, "verify job uses the approved runner and timeout", job.RunsOn == "ubuntu-latest" && job.TimeoutMinutes == 30, ""); err != nil {
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
		{name: "package publication", pattern: regexp.MustCompile(`(?:npm|cargo)\s+publish`)},
		{name: "registry credentials", pattern: regexp.MustCompile(`CARGO_REGISTRY_TOKEN|id-token:\s*write`)},
		{name: "release asset replacement", pattern: regexp.MustCompile(`--clobber`)},
	} {
		if forbidden.pattern.MatchString(source) {
			return &Error{Artifact: verifyWorkflowArtifact, Invariant: "credential-free verification excludes " + forbidden.name}
		}
	}
	expectedSteps := []struct {
		name string
		run  string
		uses string
		with map[string]any
	}{
		{name: "Check out repository", uses: "actions/checkout", with: map[string]any{"fetch-depth": 0, "persist-credentials": false}},
		{name: "Set up Go", uses: "actions/setup-go", with: map[string]any{"go-version-file": "go.mod", "cache": true}},
		{name: "Install pinned Rust toolchains", run: installRustToolchainsScript},
		{name: "Fetch locked Rust dependencies", run: fetchRustDependenciesScript},
		{name: "Vet Go", run: "go vet ./..."},
		{name: "Test Go", run: "go test -race ./..."},
		{name: "Verify repository", run: "go run ./cmd/opendart-tool verify --repository-root ."},
		{name: "Verify Rust stable contracts offline", run: stableRustVerificationScript},
		{name: "Verify transport-independent dependency graph offline", run: transportIndependentGraphScript},
		{name: "Verify reqwest feature compatibility offline", run: compatibilityVerificationScript},
		{name: "Verify Rust MSRV offline", run: msrvVerificationScript},
		{name: "Verify workspace package contents offline", run: packageVerificationScript},
		{name: "Install CLI from reviewed source offline", run: sourceInstallScript},
	}
	if len(job.Steps) != len(expectedSteps) {
		return &Error{Artifact: verifyWorkflowArtifact, Invariant: "uses only the approved verification steps"}
	}
	for index, expected := range expectedSteps {
		step := job.Steps[index]
		if step.Name != expected.name || !exactScript(step.Run, expected.run) {
			return &Error{Artifact: verifyWorkflowArtifact, Invariant: "uses only the approved verification steps", Detail: "step " + step.Name}
		}
		if expected.uses == "" {
			if step.Uses != "" || len(step.With) != 0 {
				return &Error{Artifact: verifyWorkflowArtifact, Invariant: "uses only the approved verification steps", Detail: "step " + step.Name}
			}
		} else if !strings.HasPrefix(step.Uses, expected.uses+"@") || !reflect.DeepEqual(step.With, expected.with) {
			return &Error{Artifact: verifyWorkflowArtifact, Invariant: "uses only the approved verification actions", Detail: "step " + step.Name}
		}
		if !defaultStepExecution(step) {
			return &Error{Artifact: verifyWorkflowArtifact, Invariant: "verification steps use default execution controls", Detail: "step " + step.Name}
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
	if err := checkActionPins(verifyWorkflowArtifact, verify); err != nil {
		return err
	}
	if err := checkCheckoutCredentials(verifyWorkflowArtifact, verify); err != nil {
		return err
	}
	return nil
}

func checkNativeArtifactJob(name, runner string, job workflowJob) error {
	if !defaultJobExecution(job) || job.Needs != "" || job.Uses != "" || len(job.Permissions) != 0 || !defaultRunSettings(job.Defaults) {
		return &Error{Artifact: verifyWorkflowArtifact, Invariant: "native artifact jobs use default execution controls", Detail: name}
	}
	if job.RunsOn != runner || job.TimeoutMinutes != 20 {
		return &Error{Artifact: verifyWorkflowArtifact, Invariant: "native artifact jobs use approved runners and timeouts", Detail: name}
	}
	installScript := sourceInstallScript
	if runner == "windows-latest" {
		installScript = windowsSourceInstallScript
	}
	expected := []struct {
		name string
		run  string
		uses string
		with map[string]any
		env  map[string]string
	}{
		{name: "Check out repository", uses: "actions/checkout", with: map[string]any{"fetch-depth": 0, "persist-credentials": false}},
		{name: "Install pinned Rust toolchain", run: "rustup toolchain install 1.97.1 --profile minimal"},
		{name: "Fetch locked Rust dependencies", run: nativeArtifactFetchScript},
		{name: "Verify native binary artifact behavior", run: nativeArtifactTestScript, env: map[string]string{"RUSTFLAGS": "--cfg opendart_compat"}},
		{name: "Install CLI from reviewed source offline", run: installScript},
	}
	if len(job.Steps) != len(expected) {
		return &Error{Artifact: verifyWorkflowArtifact, Invariant: "native artifact jobs use only approved steps", Detail: name}
	}
	for index, want := range expected {
		step := job.Steps[index]
		if step.Name != want.name || !exactScript(step.Run, want.run) || !reflect.DeepEqual(step.Env, want.env) || !defaultStepExecution(step) || !defaultStepRunSettings(step) {
			return &Error{Artifact: verifyWorkflowArtifact, Invariant: "native artifact jobs use only approved steps", Detail: name + ": " + step.Name}
		}
		if want.uses == "" {
			if step.Uses != "" || len(step.With) != 0 {
				return &Error{Artifact: verifyWorkflowArtifact, Invariant: "native artifact jobs use only approved steps", Detail: name + ": " + step.Name}
			}
		} else if !strings.HasPrefix(step.Uses, want.uses+"@") || !reflect.DeepEqual(step.With, want.with) {
			return &Error{Artifact: verifyWorkflowArtifact, Invariant: "native artifact jobs use only approved actions", Detail: name + ": " + step.Name}
		}
	}
	return nil
}

func checkLiveWorkflow(live workflow, source string) error {
	if err := require(liveWorkflowArtifact, "has the expected workflow name", live.Name == "Live Conformance", ""); err != nil {
		return err
	}
	if err := require(liveWorkflowArtifact, "is manual only", reflect.DeepEqual(sortedKeys(live.On), []string{"workflow_dispatch"}), ""); err != nil {
		return err
	}
	if err := require(liveWorkflowArtifact, "root permissions are empty", len(live.Permissions) == 0, ""); err != nil {
		return err
	}
	if err := require(liveWorkflowArtifact, "serializes live runs", live.Concurrency.Group == "live-conformance" && !live.Concurrency.CancelInProgress, ""); err != nil {
		return err
	}
	if err := require(liveWorkflowArtifact, "workflow uses default run settings", defaultRunSettings(live.Defaults), ""); err != nil {
		return err
	}
	if err := require(liveWorkflowArtifact, "contains only the conformance job", reflect.DeepEqual(sortedKeys(live.Jobs), []string{"conformance"}), ""); err != nil {
		return err
	}
	job := live.Jobs["conformance"]
	if !exactWorkflowExpression(job.If, "github.repository == 'cpaikr/opendart' && github.ref == 'refs/heads/main'") {
		return &Error{Artifact: liveWorkflowArtifact, Invariant: "runs only trusted main code in the canonical repository"}
	}
	if err := require(liveWorkflowArtifact, "uses only the protected live environment", job.Environment == "opendart-live-conformance", ""); err != nil {
		return err
	}
	if err := require(liveWorkflowArtifact, "producer has read-only repository permission", reflect.DeepEqual(job.Permissions, map[string]string{"contents": "read"}), ""); err != nil {
		return err
	}
	if job.Needs != "" || job.ContinueOnError || job.Uses != "" || !defaultRunSettings(job.Defaults) || job.RunsOn != "ubuntu-latest" || job.TimeoutMinutes != 30 {
		return &Error{Artifact: liveWorkflowArtifact, Invariant: "producer uses only approved execution controls"}
	}
	expectedNames := []string{"Check out trusted revision", "Set up Go", "Install pinned Rust toolchain", "Fetch locked Rust dependencies", "Build live runners without credentials", "Recheck offline gates", "Run live conformance and Rust CLI smoke", "Upload sanitized report"}
	if !reflect.DeepEqual(stepNames(job.Steps), expectedNames) {
		return &Error{Artifact: liveWorkflowArtifact, Invariant: "uses only approved producer steps in order"}
	}
	checkout := job.Steps[0]
	setup := job.Steps[1]
	installRust := job.Steps[2]
	fetchRust := job.Steps[3]
	build := job.Steps[4]
	preflight := job.Steps[5]
	run := job.Steps[6]
	upload := job.Steps[7]
	if checkout.Uses != checkoutAction || checkout.Run != "" || !reflect.DeepEqual(checkout.With, map[string]any{"persist-credentials": false}) || !defaultStepExecution(checkout) {
		return &Error{Artifact: liveWorkflowArtifact, Invariant: "checks out the trusted dispatched revision without credentials"}
	}
	if setup.Uses != setupGoAction || setup.Run != "" || !reflect.DeepEqual(setup.With, map[string]any{"cache": true, "go-version-file": "go.mod"}) || !defaultStepExecution(setup) {
		return &Error{Artifact: liveWorkflowArtifact, Invariant: "uses the approved Go setup"}
	}
	if installRust.Run != "rustup toolchain install 1.97.1 --profile minimal" || installRust.Uses != "" || !defaultStepExecution(installRust) {
		return &Error{Artifact: liveWorkflowArtifact, Invariant: "installs the approved Rust toolchain before credential exposure"}
	}
	if fetchRust.Run != nativeArtifactFetchScript || fetchRust.Uses != "" || !defaultStepExecution(fetchRust) {
		return &Error{Artifact: liveWorkflowArtifact, Invariant: "fetches locked Rust dependencies before credential exposure"}
	}
	if build.Run != liveBuildScript || build.Uses != "" || !defaultStepExecution(build) {
		return &Error{Artifact: liveWorkflowArtifact, Invariant: "builds only the approved live runners before credential exposure"}
	}
	if preflight.Run != ".live-bin/opendart-tool live-conformance --preflight-only --repository-root ." || preflight.Uses != "" || !defaultStepExecution(preflight) {
		return &Error{Artifact: liveWorkflowArtifact, Invariant: "rechecks credential-free live gates before secret exposure"}
	}
	expectedCredential := map[string]string{
		"OPENDART_API_KEY":    "${{ secrets.OPENDART_API_KEY }}",
		"OPENDART_LIVE_TESTS": "1",
	}
	if run.Run != liveRunScript {
		return &Error{Artifact: liveWorkflowArtifact, Invariant: "runs only the approved live conformance commands"}
	}
	if run.Uses != "" || !defaultStepExecution(run) {
		return &Error{Artifact: liveWorkflowArtifact, Invariant: "exposes credentials only through a direct fail-closed request step"}
	}
	if !reflect.DeepEqual(run.Env, expectedCredential) {
		return &Error{Artifact: liveWorkflowArtifact, Invariant: "sets only the approved live request gates"}
	}
	expectedUpload := map[string]any{
		"name": "live-conformance-report-${{ github.run_attempt }}", "path": "live-conformance-report.json",
		"if-no-files-found": "error", "retention-days": 7, "compression-level": 0,
		"overwrite": false, "include-hidden-files": false,
	}
	if upload.Uses != uploadArtifactAction || upload.Run != "" || !exactWorkflowExpression(upload.If, "always()") || upload.ContinueOnError || !reflect.DeepEqual(upload.With, expectedUpload) {
		return &Error{Artifact: liveWorkflowArtifact, Invariant: "uploads only the bounded sanitized report"}
	}
	for _, step := range job.Steps {
		if !defaultStepRunSettings(step) {
			return &Error{Artifact: liveWorkflowArtifact, Invariant: "producer steps use default run settings", Detail: "step " + step.Name}
		}
		if step.Name != "Run live conformance and Rust CLI smoke" && len(step.Env) != 0 {
			return &Error{Artifact: liveWorkflowArtifact, Invariant: "API key is absent outside the request boundary", Detail: "step " + step.Name}
		}
	}
	if strings.Count(source, "${{ secrets.OPENDART_API_KEY }}") != 1 || strings.Count(source, "OPENDART_API_KEY") != 2 || strings.Count(source, "OPENDART_LIVE_TESTS") != 1 || strings.Contains(source, "issues: write") || strings.Contains(source, "github.token") || strings.Contains(source, "GITHUB_TOKEN") {
		return &Error{Artifact: liveWorkflowArtifact, Invariant: "contains only the single protected credential reference and no issue authority"}
	}
	if err := checkActionPins(liveWorkflowArtifact, live); err != nil {
		return err
	}
	return checkCheckoutCredentials(liveWorkflowArtifact, live)
}

func checkNotifyWorkflow(notify workflow, source string) error {
	if err := require(notifyWorkflowArtifact, "has the expected workflow name", notify.Name == "Live Conformance Notifier", ""); err != nil {
		return err
	}
	workflowRun, ok := notify.On["workflow_run"].(map[string]any)
	expectedTrigger := map[string]any{"types": []any{"completed"}, "workflows": []any{"Live Conformance"}}
	if err := require(notifyWorkflowArtifact, "runs only after the live producer completes", reflect.DeepEqual(sortedKeys(notify.On), []string{"workflow_run"}) && ok && reflect.DeepEqual(workflowRun, expectedTrigger), ""); err != nil {
		return err
	}
	if err := require(notifyWorkflowArtifact, "root permissions are empty", len(notify.Permissions) == 0, ""); err != nil {
		return err
	}
	if err := require(notifyWorkflowArtifact, "serializes notifier runs", notify.Concurrency.Group == "live-conformance-notifier" && !notify.Concurrency.CancelInProgress, ""); err != nil {
		return err
	}
	if err := require(notifyWorkflowArtifact, "workflow uses default run settings", defaultRunSettings(notify.Defaults), ""); err != nil {
		return err
	}
	if err := require(notifyWorkflowArtifact, "contains only the notifier job", reflect.DeepEqual(sortedKeys(notify.Jobs), []string{"notify"}), ""); err != nil {
		return err
	}
	job := notify.Jobs["notify"]
	expectedCondition := "github.event.workflow_run.event == 'workflow_dispatch' && github.event.workflow_run.head_repository.full_name == github.repository && github.event.workflow_run.head_branch == github.event.repository.default_branch"
	if !exactWorkflowExpression(job.If, expectedCondition) {
		return &Error{Artifact: notifyWorkflowArtifact, Invariant: "accepts only manual trusted default-branch producer runs"}
	}
	expectedPermissions := map[string]string{"actions": "read", "contents": "read", "issues": "write"}
	if !reflect.DeepEqual(job.Permissions, expectedPermissions) || job.Environment != "" {
		return &Error{Artifact: notifyWorkflowArtifact, Invariant: "isolates minimal issue authority without the protected environment"}
	}
	if job.Needs != "" || job.ContinueOnError || job.Uses != "" || !defaultRunSettings(job.Defaults) || job.RunsOn != "ubuntu-latest" || job.TimeoutMinutes != 20 {
		return &Error{Artifact: notifyWorkflowArtifact, Invariant: "notifier uses only approved execution controls"}
	}
	expectedNames := []string{"Check out trusted producer revision", "Set up Go", "Download sanitized report", "Update live conformance issue"}
	if !reflect.DeepEqual(stepNames(job.Steps), expectedNames) {
		return &Error{Artifact: notifyWorkflowArtifact, Invariant: "uses only approved notifier steps in order"}
	}
	checkout := job.Steps[0]
	setup := job.Steps[1]
	download := job.Steps[2]
	run := job.Steps[3]
	if checkout.Uses != checkoutAction || checkout.Run != "" || !defaultStepExecution(checkout) || !reflect.DeepEqual(checkout.With, map[string]any{"persist-credentials": false, "ref": "${{ github.event.workflow_run.head_sha }}"}) {
		return &Error{Artifact: notifyWorkflowArtifact, Invariant: "checks out the exact trusted producer revision without credentials"}
	}
	if setup.Uses != setupGoAction || setup.Run != "" || !defaultStepExecution(setup) || !reflect.DeepEqual(setup.With, map[string]any{"cache": true, "go-version-file": "go.mod"}) {
		return &Error{Artifact: notifyWorkflowArtifact, Invariant: "uses the approved Go setup"}
	}
	expectedDownload := map[string]any{
		"name": "live-conformance-report-${{ github.event.workflow_run.run_attempt }}", "path": ".", "github-token": "${{ github.token }}",
		"run-id": "${{ github.event.workflow_run.id }}",
	}
	if download.ID != "report" || download.Uses != downloadArtifactAction || download.Run != "" || !download.ContinueOnError || strings.TrimSpace(download.If) != "" || !reflect.DeepEqual(download.With, expectedDownload) {
		return &Error{Artifact: notifyWorkflowArtifact, Invariant: "downloads only the producer report with fixed-failure fallback"}
	}
	expectedEnv := map[string]string{
		"GITHUB_TOKEN":               "${{ github.token }}",
		"NOTIFY_REPOSITORY":          "${{ github.repository }}",
		"NOTIFY_PRODUCER_CONCLUSION": "${{ github.event.workflow_run.conclusion }}",
		"NOTIFY_ARTIFACT_OUTCOME":    "${{ steps.report.outcome }}",
		"NOTIFY_RUN_ID":              "${{ github.event.workflow_run.id }}",
		"NOTIFY_RUN_ATTEMPT":         "${{ github.event.workflow_run.run_attempt }}",
	}
	if !exactScript(run.Run, notifyRunScript) || run.Uses != "" || !exactWorkflowExpression(run.If, "always()") || run.ContinueOnError || !reflect.DeepEqual(run.Env, expectedEnv) {
		return &Error{Artifact: notifyWorkflowArtifact, Invariant: "invokes only the isolated notifier with trusted metadata"}
	}
	for _, step := range job.Steps {
		if !defaultStepRunSettings(step) {
			return &Error{Artifact: notifyWorkflowArtifact, Invariant: "notifier steps use default run settings", Detail: "step " + step.Name}
		}
		if step.Name != "Update live conformance issue" && len(step.Env) != 0 {
			return &Error{Artifact: notifyWorkflowArtifact, Invariant: "job token environment is confined to the notifier boundary", Detail: "step " + step.Name}
		}
	}
	if strings.Contains(source, "OPENDART_API_KEY") || strings.Contains(source, "secrets.") || strings.Contains(source, "secrets[") || strings.Contains(source, "environment:") {
		return &Error{Artifact: notifyWorkflowArtifact, Invariant: "notifier cannot access the OpenDART credential or protected environment"}
	}
	if err := checkActionPins(notifyWorkflowArtifact, notify); err != nil {
		return err
	}
	return checkCheckoutCredentials(notifyWorkflowArtifact, notify)
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
		"steps.release.outputs['openapi/generated--release_created'] == 'true' || steps.draft.outputs.recovering == 'true'",
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
			"TAG_NAME": "${{ steps.release.outputs['openapi/generated--tag_name'] || steps.draft.outputs.tag_name }}",
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
