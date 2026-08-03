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
	releaseWorkflowArtifact        = ".github/workflows/release-please.yml"
	rustCrateWorkflowArtifact      = ".github/workflows/rust-crate-release.yml"
	verifyWorkflowArtifact         = ".github/workflows/verify.yml"
	fullRaceWorkflowArtifact       = ".github/workflows/full-race.yml"
	verificationScriptArtifact     = "scripts/verify"
	liveWorkflowArtifact           = ".github/workflows/live-conformance.yml"
	notifyWorkflowArtifact         = ".github/workflows/live-conformance-notify.yml"
	driftWorkflowArtifact          = ".github/workflows/guide-drift.yml"
	driftNotifyArtifact            = ".github/workflows/guide-drift-notify.yml"
	configArtifact                 = "release-please-config.json"
	manifestArtifact               = ".release-please-manifest.json"
	specificationPackagePath       = "openapi/generated"
	rustPackagePath                = "sdk/rust/crates/opendart"
	rustCLIPackagePath             = "sdk/rust/crates/opendart-cli"
	rustWorkspaceArtifact          = "sdk/rust/Cargo.toml"
	rustCargoArtifact              = "sdk/rust/crates/opendart/Cargo.toml"
	rustCLICargoArtifact           = "sdk/rust/crates/opendart-cli/Cargo.toml"
	rustLockArtifact               = "sdk/rust/Cargo.lock"
	rustCompatibilityLockArtifact  = "sdk/rust/compat/reqwest-feature-unification/Cargo.lock"
	rustProvenanceArtifact         = "sdk/rust/crates/opendart/src/provenance.rs"
	rustSDKClientArtifact          = "sdk/rust/crates/opendart/src/client.rs"
	rustCLIExecutionArtifact       = "sdk/rust/crates/opendart-cli/src/execution.rs"
	rustPackageListArtifact        = "sdk/rust/package-files.txt"
	rustCLIPackageListArtifact     = "sdk/rust/opendart-cli-package-files.txt"
	canonicalBundleArtifact        = "openapi/generated/openapi.bundle.yaml"
	releasePleaseAction            = "googleapis/release-please-action@45996ed1f6d02564a971a2fa1b5860e934307cf7"
	checkoutAction                 = "actions/checkout@9c091bb21b7c1c1d1991bb908d89e4e9dddfe3e0"
	setupGoAction                  = "actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e"
	uploadArtifactAction           = "actions/upload-artifact@043fb46d1a93c77aae656e7c1c64a875d1fc6a0a"
	downloadArtifactAction         = "actions/download-artifact@3e5f45b2cfb9172054b4087a40e8e0b5a5461e7c"
	verifyConcurrencyGroup         = "${{ github.workflow }}-${{ github.event.pull_request.number || github.run_id }}"
	recoveryScriptDigest           = "e89afe63da8e3ea1e7358ca85dff550e8337fc027fc204e9fcd1d4dad5e615ba"
	proposalScriptDigest           = "996a55e656f9409c6f4a6ee882d9c23d73f070cead0d20d27000afea41723d9d"
	componentScriptDigest          = "695c00a82d604738ac93fa794396acaf8f2d3e95d4f1b434721f727f10177fb8"
	dispatchScriptDigest           = "8e243fcb82c1d33d2fa3414d504f9f33e1a553c0ed488695b3fb8e18ee7fd0e3"
	reportProposalScriptDigest     = "d4d4c4abf3621cc737a28de686982e419fc3028199da05f02c9dbd4a70755129"
	candidateInputScriptDigest     = "58e8d863f7b44be6444100c828d15b5437ee3bce440c8883788cb902b45516d6"
	candidateAttestScriptDigest    = "4af1a683a5744824aab2210dc052ca88bcb49d76d6be5febbf216a21c21d4f99"
	candidateToolchainScriptDigest = "f8f6aaff0f83fe81760c269bbea9a39bad1ff01de95d86816a485dd3b5c818a7"
	candidateVerifyScriptDigest    = "a8443131373c7f10358fc257225b31994ed570059f55c64efa8519c5f32f1bbc"
	candidatePackageScriptDigest   = "641a0815dfbf5a967ee18a386d68ae352bd0764241d5c01fdd2bb8fee59e3813"
	candidateEvidenceScriptDigest  = "741c2079c9a6ba8469e9b66ff6baaa6b6199fe42a9985716273900a23909cf03"
	publishInputScriptDigest       = "145c6249f03ffee066267c70f7fec7e1c173350b4c0c23f312af88b6fb2ea108"
	publishEvidenceScriptDigest    = "7b13de5c812d460a59c5609365f19d2ae6d1643ba66c2355ea5460f2754673c8"
	minimalToolchainScriptDigest   = "c77326a24c08ec5f7d261209b5acc436c1a8a7f550d711ff1507ae2578c60ed5"
	publishScriptDigest            = "ccbc7055e7dc4e0b708ca469943021c693eeede20489a70242c2ce44b660f0e4"
	reconcileCrateScriptDigest     = "e1ec7e345fe461414f38fb08dce44f776ea0aa94f2604c1908133deaa629a88d"
	reconcileConsumerScriptDigest  = "aadfff67de44b6c0542080a5a5855d36357ecd1f3cdf6ba7a067041454b3d506"
	reconcileReadinessScriptDigest = "0d25fae53a860c390e350414fb2bcc853444159b338e40f22d6cff3af8762ff9"
	finalizeScriptDigest           = "679196ebf280c34d7829aa6d39d96a54b4405ecbfdd88fb5958eb00e066ecc10"

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
	driftRunScript       = "go run ./cmd/opendart-tool guide-drift --repository-root . > guide-drift-report.json"
	driftNotifyRunScript = `go run ./cmd/opendart-tool guide-drift-notify \
  --report guide-drift-report.json \
  --repository "${NOTIFY_REPOSITORY}" \
  --producer-conclusion "${NOTIFY_PRODUCER_CONCLUSION}" \
  --artifact-outcome "${NOTIFY_ARTIFACT_OUTCOME}" \
  --run-id "${NOTIFY_RUN_ID}" \
  --run-attempt "${NOTIFY_RUN_ATTEMPT}"`

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
	publishReleaseScript        = `gh release edit "${TAG_NAME}" --draft=false --prerelease=false --latest`
	installRustToolchainsScript = `rustup toolchain install 1.97.1 --profile minimal --component clippy --component rustfmt
rustup target add --toolchain 1.97.1 wasm32-unknown-unknown
rustup toolchain install 1.85.0 --profile minimal`
	fetchRustDependenciesScript = `cargo +1.97.1 fetch --locked --manifest-path sdk/rust/Cargo.toml
cargo +1.97.1 fetch --locked --manifest-path sdk/rust/compat/reqwest-feature-unification/Cargo.toml`
	stableRustVerificationScript = `cargo +1.97.1 fmt --manifest-path sdk/rust/Cargo.toml --all -- --check
cargo +1.97.1 clippy --locked --offline --manifest-path sdk/rust/Cargo.toml --workspace --all-targets --all-features -- -D warnings
cargo +1.97.1 clippy --locked --offline --manifest-path sdk/rust/Cargo.toml -p opendart --all-targets --no-default-features -- -D warnings
cargo +1.97.1 test --locked --offline --manifest-path sdk/rust/Cargo.toml --workspace --all-features
RUSTFLAGS="--cfg opendart_compat" cargo +1.97.1 test --locked --offline --manifest-path sdk/rust/Cargo.toml -p opendart --all-features --lib conformance
RUSTFLAGS="--cfg opendart_compat" cargo +1.97.1 test --locked --offline --manifest-path sdk/rust/Cargo.toml -p opendart --test public_contract repository_contract_corpus_crosses_the_public_interpreter
RUSTFLAGS="--cfg opendart_compat" cargo +1.97.1 test --locked --offline --manifest-path sdk/rust/Cargo.toml -p opendart-cli --test structured_loopback
RUSTFLAGS="--cfg opendart_compat" cargo +1.97.1 test --locked --offline --manifest-path sdk/rust/Cargo.toml -p opendart-cli --test binary_loopback
cargo +1.97.1 test --locked --offline --manifest-path sdk/rust/Cargo.toml -p opendart --no-default-features
cargo +1.97.1 test --locked --offline --manifest-path sdk/rust/Cargo.toml -p opendart-cli --no-default-features
RUSTDOCFLAGS="-D warnings" cargo +1.97.1 doc --locked --offline --manifest-path sdk/rust/Cargo.toml --workspace --all-features --no-deps`
	nativeArtifactFetchScript = `cargo +1.97.1 fetch --locked --manifest-path sdk/rust/Cargo.toml`
	nativeArtifactTestScript  = `cargo +1.97.1 test --locked --offline --manifest-path sdk/rust/Cargo.toml -p opendart-cli --bin opendart
cargo +1.97.1 test --locked --offline --manifest-path sdk/rust/Cargo.toml -p opendart-cli --test binary_loopback`
	compatibilityVerificationScript = `RUSTFLAGS="--cfg opendart_compat" cargo +1.97.1 test --locked --offline --manifest-path sdk/rust/compat/reqwest-feature-unification/Cargo.toml`
	transportIndependentGraphScript = `no_default_tree="${verification_tmp}/no-default-tree.txt"
cargo +1.97.1 tree --locked --offline --manifest-path sdk/rust/Cargo.toml -p opendart --no-default-features -e normal --prefix none > "${no_default_tree}"
if grep -Eq '^(bytes|futures(-[^ ]+)?|h2|hickory-[^ ]+|http-body(-[^ ]+)?|hyper(-[^ ]+)?|native-tls|openssl(-[^ ]+)?|reqwest|ring|rustls(-[^ ]+)?|tokio(-[^ ]+)?|tower(-[^ ]+)?|trust-dns-[^ ]+|webpki(-[^ ]+)?)[[:space:]]v' "${no_default_tree}"; then
  grep -E '^(bytes|futures(-[^ ]+)?|h2|hickory-[^ ]+|http-body(-[^ ]+)?|hyper(-[^ ]+)?|native-tls|openssl(-[^ ]+)?|reqwest|ring|rustls(-[^ ]+)?|tokio(-[^ ]+)?|tower(-[^ ]+)?|trust-dns-[^ ]+|webpki(-[^ ]+)?)[[:space:]]v' "${no_default_tree}"
  exit 1
fi`
	wasmVerificationScript = `cargo +1.97.1 clippy --locked --offline --manifest-path sdk/rust/Cargo.toml -p opendart --lib --target wasm32-unknown-unknown -- -D warnings
cargo +1.97.1 clippy --locked --offline --manifest-path sdk/rust/Cargo.toml -p opendart --lib --target wasm32-unknown-unknown --no-default-features -- -D warnings
wasm_tree="${verification_tmp}/wasm-tree.txt"
cargo +1.97.1 tree --locked --offline --manifest-path sdk/rust/Cargo.toml -p opendart --target wasm32-unknown-unknown -e normal --prefix none > "${wasm_tree}"
if grep -Eq '^(bytes|futures(-[^ ]+)?|h2|hickory-[^ ]+|http-body(-[^ ]+)?|hyper(-[^ ]+)?|native-tls|openssl(-[^ ]+)?|reqwest|ring|rustls(-[^ ]+)?|tokio(-[^ ]+)?|tower(-[^ ]+)?|trust-dns-[^ ]+|webpki(-[^ ]+)?)[[:space:]]v' "${wasm_tree}"; then
  grep -E '^(bytes|futures(-[^ ]+)?|h2|hickory-[^ ]+|http-body(-[^ ]+)?|hyper(-[^ ]+)?|native-tls|openssl(-[^ ]+)?|reqwest|ring|rustls(-[^ ]+)?|tokio(-[^ ]+)?|tower(-[^ ]+)?|trust-dns-[^ ]+|webpki(-[^ ]+)?)[[:space:]]v' "${wasm_tree}"
  exit 1
fi`
	msrvVerificationScript = `cargo +1.85.0 check --locked --offline --manifest-path sdk/rust/Cargo.toml --workspace --all-targets --all-features
cargo +1.85.0 check --locked --offline --manifest-path sdk/rust/Cargo.toml -p opendart --no-default-features
cargo +1.85.0 check --locked --offline --manifest-path sdk/rust/Cargo.toml -p opendart-cli --all-targets --no-default-features
cargo +1.85.0 metadata --locked --offline --manifest-path sdk/rust/Cargo.toml --no-deps > /dev/null`
	packageVerificationScript = `sdk_package_files="${verification_tmp}/sdk-package-files.txt"
cli_package_files="${verification_tmp}/cli-package-files.txt"
cargo +1.97.1 package --locked --offline --manifest-path sdk/rust/crates/opendart/Cargo.toml --list > "${sdk_package_files}"
diff -u sdk/rust/package-files.txt "${sdk_package_files}"
cargo +1.97.1 package --locked --offline --manifest-path sdk/rust/crates/opendart-cli/Cargo.toml --list > "${cli_package_files}"
diff -u sdk/rust/opendart-cli-package-files.txt "${cli_package_files}"
CARGO_TARGET_DIR="${verification_tmp}/package-target" cargo +1.97.1 package --workspace --locked --offline --manifest-path sdk/rust/Cargo.toml`
	sourceInstallScript = `install_workspace="$(mktemp -d)"
CARGO_TARGET_DIR="${install_workspace}/target" cargo +1.97.1 install --locked --offline --path sdk/rust/crates/opendart-cli --root "${install_workspace}/root"
"${install_workspace}/root/bin/opendart" --version
"${install_workspace}/root/bin/opendart" operations list > /dev/null`
	verifyAggregateScript = `failed=0
for result in \
  "go=${GO_RESULT}" \
  "rust=${RUST_RESULT}" \
  "artifact-macos=${MACOS_RESULT}" \
  "artifact-windows=${WINDOWS_RESULT}"
do
  case "${result}" in
    *=success) ;;
    *)
      echo "${result}" >&2
      failed=1
      ;;
  esac
done
exit "${failed}"`
	windowsSourceInstallScript = `$installWorkspace = Join-Path $env:RUNNER_TEMP ([guid]::NewGuid().ToString())
$installRoot = Join-Path $installWorkspace "root"
$env:CARGO_TARGET_DIR = Join-Path $installWorkspace "target"
cargo +1.97.1 install --locked --offline --path sdk/rust/crates/opendart-cli --root $installRoot
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
$binary = Join-Path $installRoot "bin/opendart.exe"
& $binary --version
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
& $binary operations list | Out-Null
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
Remove-Item Env:HOME -ErrorAction SilentlyContinue
$env:USERPROFILE = $installRoot
$homeDocument = & $binary | ConvertFrom-Json
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
if (-not $homeDocument.executable.display.StartsWith("~")) { exit 1 }`
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
	rustSDKTimeout  = regexp.MustCompile(`(?m)^const DEFAULT_TOTAL_TIMEOUT: Duration = Duration::from_secs\(([0-9]+)\);$`)
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
// automated observation workflow policies.
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
	compatibilityLockSource, err := readArtifact(absoluteRoot, rustCompatibilityLockArtifact)
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
	if err := checkReleaseConfiguration(configSource, manifestSource, cargoSource, cliCargoSource, lockSource, compatibilityLockSource); err != nil {
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
	sdkClientSource, err := readArtifact(absoluteRoot, rustSDKClientArtifact)
	if err != nil {
		return err
	}
	cliExecutionSource, err := readArtifact(absoluteRoot, rustCLIExecutionArtifact)
	if err != nil {
		return err
	}
	if err := checkRustCLITimeoutMirror(sdkClientSource, cliExecutionSource); err != nil {
		return err
	}

	releaseSource, err := readArtifact(absoluteRoot, releaseWorkflowArtifact)
	if err != nil {
		return err
	}
	rustCrateSource, err := readArtifact(absoluteRoot, rustCrateWorkflowArtifact)
	if err != nil {
		return err
	}
	verifySource, err := readArtifact(absoluteRoot, verifyWorkflowArtifact)
	if err != nil {
		return err
	}
	fullRaceSource, err := readArtifact(absoluteRoot, fullRaceWorkflowArtifact)
	if err != nil {
		return err
	}
	verificationScriptSource, err := readArtifact(absoluteRoot, verificationScriptArtifact)
	if err != nil {
		return err
	}
	if err := checkVerificationScript(absoluteRoot, verificationScriptSource); err != nil {
		return err
	}
	if err := checkRaceOwnership(absoluteRoot); err != nil {
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
	driftSource, err := readArtifact(absoluteRoot, driftWorkflowArtifact)
	if err != nil {
		return err
	}
	driftNotifySource, err := readArtifact(absoluteRoot, driftNotifyArtifact)
	if err != nil {
		return err
	}
	return checkWorkflows(releaseSource, rustCrateSource, verifySource, fullRaceSource, liveSource, notifySource, driftSource, driftNotifySource)
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
	if !packagePathsAreSorted(lines) {
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
		if !contains(lines, name) {
			return &Error{Artifact: rustPackageListArtifact, Invariant: "contains required package evidence", Detail: name}
		}
	}
	return checkPackageInventoryPrivateInputs(rustPackageListArtifact, lines)
}

func checkRustCLIPackage(cliCargoSource, workspaceSource, lockSource, packageListSource []byte) error {
	publish, err := cargoPackageStringArray(cliCargoSource, "publish")
	if err != nil || !reflect.DeepEqual(publish, []string{"crates-io"}) {
		return &Error{Artifact: rustCLICargoArtifact, Invariant: "authorizes only the crates.io registry", Cause: err}
	}
	include, err := cargoPackageStringArray(cliCargoSource, "include")
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
	pin, err := cargoInlineDependencyVersion(cliCargoSource, "opendart")
	if err != nil || pin != "="+sdkVersion {
		return &Error{Artifact: rustCLICargoArtifact, Invariant: "exact-pins the workspace SDK version", Cause: err}
	}
	if !bytes.Contains(cliCargoSource, []byte(`opendart = { path = "../opendart", version = "=`+sdkVersion+`", default-features = false, features = ["client-reqwest", "serde-json"] } # x-release-please-version`)) {
		return &Error{Artifact: rustCLICargoArtifact, Invariant: "marks the exact SDK pin for SDK-owned release updates"}
	}
	if !bytes.Contains(cliCargoSource, []byte("serde_json.workspace = true")) {
		return &Error{Artifact: rustCLICargoArtifact, Invariant: "inherits the reviewed JSON encoder behavior"}
	}
	if !bytes.Contains(workspaceSource, []byte(`serde_json = { version = "=1.0.150", features = ["arbitrary_precision", "preserve_order"] }`)) {
		return &Error{Artifact: rustWorkspaceArtifact, Invariant: "exact-pins the reviewed JSON encoder behavior"}
	}

	lines := strings.Fields(string(packageListSource))
	if !packagePathsAreSorted(lines) {
		return &Error{Artifact: rustCLIPackageListArtifact, Invariant: "is sorted for deterministic comparison"}
	}
	for _, name := range []string{
		".cargo_vcs_info.json",
		"CHANGELOG.md",
		"Cargo.lock",
		"Cargo.toml",
		"LICENSE",
		"README.md",
		"src/generated/dispatch/.opendart-cli-dispatch-generated",
		"src/generated/dispatch/adapter.rs",
		"src/generated/dispatch/dispatch_cases.json",
		"src/generated/dispatch/mod.rs",
		"src/generated/interface/.opendart-cli-interface-generated",
		"src/generated/interface/catalog.rs",
		"src/generated/interface/command.rs",
		"src/generated/interface/mod.rs",
		"src/main.rs",
		"tests/binary_loopback.rs",
		"tests/common/mod.rs",
		"tests/discovery.rs",
		"tests/fixtures/invalid-invocation.json",
		"tests/fixtures/missing-api-key.json",
		"tests/live_smoke.rs",
		"tests/structured_loopback.rs",
	} {
		if !contains(lines, name) {
			return &Error{Artifact: rustCLIPackageListArtifact, Invariant: "contains required package evidence", Detail: name}
		}
	}
	return checkPackageInventoryPrivateInputs(rustCLIPackageListArtifact, lines)
}

func checkRustCLITimeoutMirror(sdkClientSource, cliExecutionSource []byte) error {
	sdkMatches := rustSDKTimeout.FindAllSubmatch(sdkClientSource, -1)
	if len(sdkMatches) != 1 {
		return &Error{
			Artifact:  rustSDKClientArtifact,
			Invariant: "defines one SDK total timeout default",
		}
	}
	if bytes.Contains(cliExecutionSource, []byte("SDK_DEFAULT_TOTAL_TIMEOUT")) ||
		bytes.Count(cliExecutionSource, []byte("let total_timeout = client.total_timeout();")) != 1 {
		return &Error{
			Artifact:  rustCLIExecutionArtifact,
			Invariant: "reads total timeout policy from the constructed SDK client",
		}
	}
	return nil
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

func packagePathsAreSorted(values []string) bool {
	return sort.SliceIsSorted(values, func(left, right int) bool {
		return packagePathLess(values[left], values[right])
	})
}

func packagePathLess(left, right string) bool {
	leftParts := strings.Split(left, "/")
	rightParts := strings.Split(right, "/")
	shared := min(len(leftParts), len(rightParts))
	for index := range shared {
		if leftParts[index] != rightParts[index] {
			return leftParts[index] < rightParts[index]
		}
	}
	return len(leftParts) < len(rightParts)
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
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
	Needs           workflowNeeds     `yaml:"needs"`
	If              string            `yaml:"if"`
	ContinueOnError bool              `yaml:"continue-on-error"`
	Defaults        workflowDefaults  `yaml:"defaults"`
	Permissions     map[string]string `yaml:"permissions"`
	Strategy        workflowStrategy  `yaml:"strategy"`
	RunsOn          string            `yaml:"runs-on"`
	TimeoutMinutes  int               `yaml:"timeout-minutes"`
	Environment     string            `yaml:"environment"`
	Uses            string            `yaml:"uses"`
	With            map[string]any    `yaml:"with"`
	Outputs         map[string]string `yaml:"outputs"`
	Steps           []workflowStep    `yaml:"steps"`
}

type workflowStrategy struct {
	FailFast *bool             `yaml:"fail-fast"`
	Matrix   map[string]string `yaml:"matrix"`
}

type workflowNeeds []string

func (needs *workflowNeeds) UnmarshalYAML(node *yaml.Node) error {
	var values []string
	switch node.Kind {
	case yaml.ScalarNode:
		var value string
		if err := node.Decode(&value); err != nil {
			return err
		}
		values = []string{value}
	case yaml.SequenceNode:
		if err := node.Decode(&values); err != nil {
			return err
		}
	default:
		return fmt.Errorf("needs must be a job name or sequence of job names")
	}
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("needs contains an empty job name")
		}
	}
	*needs = values
	return nil
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

type workflowStepExpectation struct {
	name  string
	run   string
	uses  string
	with  map[string]any
	env   map[string]string
	shell string
}

type releaseStepExpectation struct {
	name      string
	id        string
	runDigest string
	uses      string
	with      map[string]any
	env       map[string]string
}

func checkReleaseConfiguration(configSource, manifestSource, cargoSource, cliCargoSource, lockSource, compatibilityLockSource []byte) error {
	var configFields map[string]json.RawMessage
	if err := json.Unmarshal(configSource, &configFields); err != nil {
		return &Error{Artifact: configArtifact, Invariant: "valid JSON", Cause: err}
	}
	var config struct {
		SeparatePullRequests bool                      `json:"separate-pull-requests"`
		Packages             map[string]map[string]any `json:"packages"`
	}
	if err := json.Unmarshal(configSource, &config); err != nil {
		return &Error{Artifact: configArtifact, Invariant: "valid JSON", Cause: err}
	}
	var manifest map[string]string
	if err := json.Unmarshal(manifestSource, &manifest); err != nil {
		return &Error{Artifact: manifestArtifact, Invariant: "valid JSON", Cause: err}
	}

	manifestKeys := sortedKeys(manifest)
	manifestScope := reflect.DeepEqual(manifestKeys, []string{specificationPackagePath}) ||
		reflect.DeepEqual(manifestKeys, []string{specificationPackagePath, rustPackagePath})
	if err := require(manifestArtifact, "contains only the specification and optionally published SDK component", manifestScope, "the unpublished CLI must remain absent"); err != nil {
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
		reflect.DeepEqual(sortedKeys(configFields), []string{"$schema", "bootstrap-sha", "packages", "separate-pull-requests"}),
		"exact option allowlist is required",
	); err != nil {
		return err
	}
	if err := require(configArtifact, "isolates component release proposals", config.SeparatePullRequests, "separate-pull-requests must be true"); err != nil {
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
		"prerelease",
		"prerelease-type",
		"release-type",
		"versioning",
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
		"bump-minor-pre-major":           false,
		"bump-patch-for-minor-pre-major": false,
		"versioning":                     "prerelease",
		"prerelease":                     true,
		"prerelease-type":                "beta",
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
			"jsonpath": `$.package[?(@.name.value == "opendart")].version`,
		},
		map[string]any{
			"type":     "toml",
			"path":     "/sdk/rust/compat/reqwest-feature-unification/Cargo.lock",
			"jsonpath": `$.package[?(@.name.value == "opendart")].version`,
		},
		map[string]any{
			"type": "generic",
			"path": "/sdk/rust/crates/opendart-cli/Cargo.toml",
		},
	}
	if err := require(configArtifact, "Rust package updates both SDK locks and CLI SDK pin", reflect.DeepEqual(rustPackage["extra-files"], expectedExtraFiles), "exact root-relative updaters are required"); err != nil {
		return err
	}

	cliPackage := config.Packages[rustCLIPackagePath]
	expectedCLIKeys := []string{
		"bump-minor-pre-major",
		"bump-patch-for-minor-pre-major",
		"changelog-path",
		"component",
		"draft",
		"exclude-paths",
		"extra-files",
		"force-tag-creation",
		"include-component-in-tag",
		"include-v-in-release-name",
		"include-v-in-tag",
		"release-type",
	}
	if err := require(configArtifact, "CLI package contains only supported options", reflect.DeepEqual(sortedKeys(cliPackage), expectedCLIKeys), "exact option allowlist is required"); err != nil {
		return err
	}
	expectedCLIValues := make(map[string]any, len(expectedRustValues))
	for key, value := range expectedRustValues {
		expectedCLIValues[key] = value
	}
	expectedCLIValues["component"] = "opendart-cli"
	expectedCLIValues["bump-minor-pre-major"] = false
	expectedCLIValues["bump-patch-for-minor-pre-major"] = false
	delete(expectedCLIValues, "versioning")
	delete(expectedCLIValues, "prerelease")
	delete(expectedCLIValues, "prerelease-type")
	delete(expectedCLIValues, "release-as")
	for key, value := range expectedCLIValues {
		if !reflect.DeepEqual(cliPackage[key], value) {
			return &Error{Artifact: configArtifact, Invariant: "CLI package " + key, Detail: fmt.Sprintf("want %v", value)}
		}
	}
	expectedCLIExtraFiles := []any{map[string]any{
		"type":     "toml",
		"path":     "/sdk/rust/Cargo.lock",
		"jsonpath": `$.package[?(@.name.value == "opendart-cli")].version`,
	}}
	if err := require(configArtifact, "CLI package updates its workspace lock version", reflect.DeepEqual(cliPackage["extra-files"], expectedCLIExtraFiles), "exact root-relative TOML updater is required"); err != nil {
		return err
	}
	expectedCLIExclusions := []any{rustCLIPackagePath}
	if err := require(configArtifact, "CLI package remains excluded until publication is authorized", reflect.DeepEqual(cliPackage["exclude-paths"], expectedCLIExclusions), "exact CLI path exclusion is required"); err != nil {
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
	compatibilityLockVersion, err := cargoLockPackageVersion(compatibilityLockSource, "opendart")
	if err != nil {
		return &Error{Artifact: rustCompatibilityLockArtifact, Invariant: "contains one opendart package version", Cause: err}
	}
	if err := require(rustCompatibilityLockArtifact, "matches the crate package version", cargoVersion == compatibilityLockVersion, ""); err != nil {
		return err
	}
	if manifestVersion, exists := manifest[rustPackagePath]; exists {
		betaVersion := regexp.MustCompile(`^0\.1\.0-beta\.[1-9][0-9]*$`).MatchString(manifestVersion)
		if err := require(manifestArtifact, "published SDK beta version matches the crate and both locks", semanticVersion.MatchString(manifestVersion) && betaVersion && manifestVersion == cargoVersion, ""); err != nil {
			return err
		}
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

func checkWorkflows(releaseSource, rustCrateSource, verifySource, fullRaceSource, liveSource, notifySource, driftSource, driftNotifySource []byte) error {
	release, err := decodeWorkflow(releaseWorkflowArtifact, releaseSource)
	if err != nil {
		return err
	}
	rustCrate, err := decodeWorkflow(rustCrateWorkflowArtifact, rustCrateSource)
	if err != nil {
		return err
	}
	verify, err := decodeWorkflow(verifyWorkflowArtifact, verifySource)
	if err != nil {
		return err
	}
	fullRace, err := decodeWorkflow(fullRaceWorkflowArtifact, fullRaceSource)
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
	drift, err := decodeWorkflow(driftWorkflowArtifact, driftSource)
	if err != nil {
		return err
	}
	driftNotify, err := decodeWorkflow(driftNotifyArtifact, driftNotifySource)
	if err != nil {
		return err
	}

	if err := checkReleasePipelineWorkflow(release, string(releaseSource)); err != nil {
		return err
	}
	if err := checkRustCrateWorkflow(rustCrate, string(rustCrateSource)); err != nil {
		return err
	}
	if err := checkVerifyWorkflow(verify, string(verifySource)); err != nil {
		return err
	}
	if err := checkFullRaceWorkflow(fullRace, string(fullRaceSource)); err != nil {
		return err
	}
	if err := checkLiveWorkflow(live, string(liveSource)); err != nil {
		return err
	}
	if err := checkNotifyWorkflow(notify, string(notifySource)); err != nil {
		return err
	}
	if err := checkDriftWorkflow(drift, string(driftSource)); err != nil {
		return err
	}
	return checkDriftNotifyWorkflow(driftNotify, string(driftNotifySource))
}

func checkReleasePipelineWorkflow(release workflow, source string) error {
	if err := require(releaseWorkflowArtifact, "has the expected workflow name", release.Name == "Release Please", ""); err != nil {
		return err
	}
	push, ok := release.On["push"].(map[string]any)
	if err := require(releaseWorkflowArtifact, "runs only for pushes to main", reflect.DeepEqual(sortedKeys(release.On), []string{"push"}) && ok && reflect.DeepEqual(push["branches"], []any{"main"}), ""); err != nil {
		return err
	}
	if err := require(releaseWorkflowArtifact, "serializes release runs", release.Concurrency.Group == "release-please" && !release.Concurrency.CancelInProgress, ""); err != nil {
		return err
	}
	if err := require(releaseWorkflowArtifact, "root permissions are empty", len(release.Permissions) == 0, ""); err != nil {
		return err
	}
	if !defaultRunSettings(release.Defaults) {
		return &Error{Artifact: releaseWorkflowArtifact, Invariant: "workflow uses default run settings"}
	}
	if strings.Contains(source, "cargo publish") || strings.Contains(source, "CARGO_REGISTRY_TOKEN") || strings.Contains(source, "id-token: write") || strings.Contains(source, "secrets: inherit") {
		return &Error{Artifact: releaseWorkflowArtifact, Invariant: "keeps registry credentials out and actions-write authority isolated"}
	}
	if err := require(releaseWorkflowArtifact, "contains only approved release jobs", reflect.DeepEqual(sortedKeys(release.Jobs), []string{"dispatch-release-pr-checks", "release-please", "report-release-pr-status", "sdk-release", "verify"}), ""); err != nil {
		return err
	}

	verifyCall := release.Jobs["verify"]
	expectedVerifyWith := map[string]any{"expected_sha": "${{ github.sha }}"}
	if verifyCall.Uses != "./.github/workflows/verify.yml" || !reflect.DeepEqual(verifyCall.Permissions, map[string]string{"contents": "read"}) || !reflect.DeepEqual(verifyCall.With, expectedVerifyWith) || !workflowNeedsExactly(verifyCall.Needs) || !defaultJobExecution(verifyCall) || !defaultRunSettings(verifyCall.Defaults) {
		return &Error{Artifact: releaseWorkflowArtifact, Invariant: "verifies the exact pushed revision with read-only authority"}
	}

	releaseJob := release.Jobs["release-please"]
	expectedReleasePermissions := map[string]string{"contents": "write", "issues": "write", "pull-requests": "write"}
	expectedReleaseOutputs := map[string]string{
		"prs":                 "${{ steps.proposals.outputs.numbers }}",
		"sdk_prerelease":      "${{ steps.component.outputs.sdk_prerelease }}",
		"sdk_release_created": "${{ steps.component.outputs.sdk_release_created }}",
		"sdk_sha":             "${{ steps.component.outputs.sdk_sha }}",
		"sdk_tag_name":        "${{ steps.component.outputs.sdk_tag_name }}",
		"sdk_version":         "${{ steps.component.outputs.sdk_version }}",
	}
	expectedReleaseSteps := []string{
		"Check out repository",
		"Detect interrupted component release",
		"Run Release Please",
		"Normalize release proposal numbers",
		"Normalize component release outputs",
		"Check out specification release commit",
		"Prepare specification release assets",
		"Upload specification release assets",
		"Publish immutable specification release",
	}
	if !workflowNeedsExactly(releaseJob.Needs, "verify") || !reflect.DeepEqual(releaseJob.Permissions, expectedReleasePermissions) || !reflect.DeepEqual(releaseJob.Outputs, expectedReleaseOutputs) || releaseJob.RunsOn != "blacksmith-2vcpu-ubuntu-2404" || releaseJob.TimeoutMinutes != 20 || !defaultJobExecution(releaseJob) || !defaultRunSettings(releaseJob.Defaults) || !reflect.DeepEqual(stepNames(releaseJob.Steps), expectedReleaseSteps) {
		return &Error{Artifact: releaseWorkflowArtifact, Invariant: "release proposal and specification publication use the approved boundary"}
	}
	for _, step := range releaseJob.Steps {
		if step.ContinueOnError || !defaultStepRunSettings(step) {
			return &Error{Artifact: releaseWorkflowArtifact, Invariant: "release steps fail closed with default run settings", Detail: step.Name}
		}
	}
	releaseStepIndex, releaseStep, err := stepByID(releaseJob.Steps, "release")
	if err != nil || releaseStep.Uses != releasePleaseAction || releaseStep.Run != "" || !exactWorkflowExpression(releaseStep.If, "steps.recovery.outputs.components == '[]'") || !reflect.DeepEqual(releaseStep.With, map[string]any{"token": "${{ secrets.GITHUB_TOKEN }}"}) {
		return &Error{Artifact: releaseWorkflowArtifact, Invariant: "uses the approved pinned Release Please action only outside recovery", Cause: err}
	}
	recoveryIndex, recovery, err := stepByID(releaseJob.Steps, "recovery")
	recoveryDigestDetail := scriptDigestMismatchDetail(recovery.Run, recoveryScriptDigest)
	if err != nil || recoveryIndex >= releaseStepIndex || !defaultStepExecution(recovery) || !hasScriptDigest(recovery.Run, recoveryScriptDigest) || !reflect.DeepEqual(recovery.Env, map[string]string{"GH_TOKEN": "${{ secrets.GITHUB_TOKEN }}"}) || !containsAll(recovery.Run,
		"gh api --paginate --slurp", "tag exists without a matching GitHub release", "ambiguous component recovery state",
		"git merge-base --is-ancestor", "probe_tag_ref()", "gh api --include", "if test \"${http_status}\" = 404",
		"tag_ref_status=\"$(probe_tag_ref \"${tag_name}\")\"", "inspect_component specification openapi/generated v",
		"inspect_component sdk sdk/rust/crates/opendart opendart-v",
		"if test \"${component}\" = sdk;", "test \"${version}\" = 0.1.0-beta.1",
		"157d78aa62bace4b00df6677bc3372baf88b9281", "superseded SDK candidate state mismatch") {
		return &Error{Artifact: releaseWorkflowArtifact, Invariant: "component recovery fails closed on ambiguous, tag-only, or non-ancestor state", Detail: recoveryDigestDetail, Cause: err}
	}
	_, proposals, err := stepByID(releaseJob.Steps, "proposals")
	proposalDigestDetail := scriptDigestMismatchDetail(proposals.Run, proposalScriptDigest)
	if err != nil || !defaultStepExecution(proposals) || !hasScriptDigest(proposals.Run, proposalScriptDigest) || !reflect.DeepEqual(proposals.Env, map[string]string{"RELEASE_PRS": "${{ steps.release.outputs.prs }}"}) || !containsAll(proposals.Run, "[.[].number]", "type == \"number\" and . > 0", "numbers=[]") {
		return &Error{Artifact: releaseWorkflowArtifact, Invariant: "exposes only validated release proposal numbers", Detail: proposalDigestDetail, Cause: err}
	}
	_, component, err := stepByID(releaseJob.Steps, "component")
	expectedComponentEnv := map[string]string{
		"CLI_CREATED":         "${{ steps.release.outputs['sdk/rust/crates/opendart-cli--release_created'] }}",
		"RECOVERY_COMPONENTS": "${{ steps.recovery.outputs.components }}",
		"SDK_CREATED":         "${{ steps.release.outputs['sdk/rust/crates/opendart--release_created'] }}",
		"SDK_SHA":             "${{ steps.release.outputs['sdk/rust/crates/opendart--sha'] }}",
		"SDK_TAG":             "${{ steps.release.outputs['sdk/rust/crates/opendart--tag_name'] }}",
		"SDK_VERSION":         "${{ steps.release.outputs['sdk/rust/crates/opendart--version'] }}",
		"SPEC_CREATED":        "${{ steps.release.outputs['openapi/generated--release_created'] }}",
		"SPEC_SHA":            "${{ steps.release.outputs['openapi/generated--sha'] }}",
		"SPEC_TAG":            "${{ steps.release.outputs['openapi/generated--tag_name'] }}",
		"SPEC_VERSION":        "${{ steps.release.outputs['openapi/generated--version'] }}",
	}
	componentDigestDetail := scriptDigestMismatchDetail(component.Run, componentScriptDigest)
	if err != nil || !defaultStepExecution(component) || !hasScriptDigest(component.Run, componentScriptDigest) || !reflect.DeepEqual(component.Env, expectedComponentEnv) || !containsAll(component.Run, "CLI publication is not authorized", "RECOVERY_COMPONENTS", "^0\\.1\\.0-beta\\.[1-9][0-9]*$", "test \"${SDK_VERSION}\" = 0.1.0", "sdk_prerelease=${sdk_prerelease}") {
		return &Error{Artifact: releaseWorkflowArtifact, Invariant: "normalizes authorized component releases and blocks CLI publication", Detail: componentDigestDetail, Cause: err}
	}
	specCheckoutIndex, specCheckout, err := stepByName(releaseJob.Steps, "Check out specification release commit")
	prepareIndex, prepare, prepareErr := stepByName(releaseJob.Steps, "Prepare specification release assets")
	uploadIndex, upload, uploadErr := stepByName(releaseJob.Steps, "Upload specification release assets")
	publishIndex, publishSpec, publishErr := stepByName(releaseJob.Steps, "Publish immutable specification release")
	specCondition := "steps.component.outputs.spec_release_created == 'true'"
	if err != nil || prepareErr != nil || uploadErr != nil || publishErr != nil ||
		!(releaseStepIndex < specCheckoutIndex && specCheckoutIndex < prepareIndex && prepareIndex < uploadIndex && uploadIndex < publishIndex) ||
		!exactWorkflowExpression(specCheckout.If, specCondition) || !exactWorkflowExpression(prepare.If, specCondition) || !exactWorkflowExpression(upload.If, specCondition) || !exactWorkflowExpression(publishSpec.If, specCondition) ||
		!isCheckoutAction(specCheckout.Uses) || specCheckout.With["ref"] != "${{ steps.component.outputs.spec_sha }}" || specCheckout.With["persist-credentials"] != false ||
		len(specCheckout.Env) != 0 || len(prepare.Env) != 0 ||
		!reflect.DeepEqual(upload.Env, map[string]string{"GH_TOKEN": "${{ secrets.GITHUB_TOKEN }}", "TAG_NAME": "${{ steps.component.outputs.spec_tag_name }}"}) ||
		!reflect.DeepEqual(publishSpec.Env, map[string]string{"GH_TOKEN": "${{ secrets.GITHUB_TOKEN }}", "TAG_NAME": "${{ steps.component.outputs.spec_tag_name }}"}) ||
		!exactScript(prepare.Run, prepareReleaseAssetsScript) || !exactScript(upload.Run, uploadReleaseAssetsScript) || !exactScript(publishSpec.Run, publishReleaseScript) {
		return &Error{Artifact: releaseWorkflowArtifact, Invariant: "publishes immutable specification assets only for the attested component SHA", Cause: errors.Join(err, prepareErr, uploadErr, publishErr)}
	}

	dispatch := release.Jobs["dispatch-release-pr-checks"]
	expectedDispatchPermissions := map[string]string{"actions": "write", "contents": "read", "pull-requests": "read"}
	expectedDispatchOutputs := map[string]string{"proposals": "${{ steps.dispatch.outputs.proposals }}"}
	if !workflowNeedsExactly(dispatch.Needs, "release-please") || !reflect.DeepEqual(dispatch.Permissions, expectedDispatchPermissions) || !reflect.DeepEqual(dispatch.Outputs, expectedDispatchOutputs) || dispatch.RunsOn != "ubuntu-latest" || dispatch.TimeoutMinutes != 10 || !defaultRunSettings(dispatch.Defaults) || dispatch.ContinueOnError || !exactWorkflowExpression(dispatch.If, "needs.release-please.outputs.prs != '[]'") || !reflect.DeepEqual(stepNames(dispatch.Steps), []string{"Dispatch exact-SHA release proposal checks"}) {
		return &Error{Artifact: releaseWorkflowArtifact, Invariant: "isolates actions-write authority in the release proposal dispatcher"}
	}
	dispatchStep := dispatch.Steps[0]
	dispatchRun := dispatchStep.Run
	dispatchDigestDetail := scriptDigestMismatchDetail(dispatchRun, dispatchScriptDigest)
	if !hasScriptDigest(dispatchRun, dispatchScriptDigest) || dispatchStep.ID != "dispatch" || !reflect.DeepEqual(dispatchStep.Env, map[string]string{"GH_TOKEN": "${{ secrets.GITHUB_TOKEN }}", "RELEASE_PRS": "${{ needs.release-please.outputs.prs }}"}) || !defaultStepExecution(dispatchStep) || !containsAll(dispatchRun, "numbers=\"$(jq -er", ".author.login == \"app/github-actions\"", ".baseRefName == \"main\"", "autorelease: pending", ".headRefOid", "verify_after_id", "full_race_after_id", "proposals=${proposals}",
		`gh workflow run verify.yml --repo "${GITHUB_REPOSITORY}" --ref "${branch}" -f expected_sha="${sha}"`,
		`gh workflow run full-race.yml --repo "${GITHUB_REPOSITORY}" --ref "${branch}" -f expected_sha="${sha}"`) {
		return &Error{Artifact: releaseWorkflowArtifact, Invariant: "dispatches required checks for the canonical Release Please PR head SHA", Detail: dispatchDigestDetail}
	}

	report := release.Jobs["report-release-pr-status"]
	expectedReportPermissions := map[string]string{"actions": "read", "contents": "read", "pull-requests": "read", "statuses": "write"}
	expectedReportMatrix := map[string]string{"proposal": "${{ fromJSON(needs.dispatch-release-pr-checks.outputs.proposals) }}"}
	if !workflowNeedsExactly(report.Needs, "dispatch-release-pr-checks") || !reflect.DeepEqual(report.Permissions, expectedReportPermissions) || report.Strategy.FailFast == nil || *report.Strategy.FailFast || !reflect.DeepEqual(report.Strategy.Matrix, expectedReportMatrix) || report.RunsOn != "ubuntu-latest" || report.TimeoutMinutes != 45 || !defaultRunSettings(report.Defaults) || report.ContinueOnError || !exactWorkflowExpression(report.If, "needs.dispatch-release-pr-checks.outputs.proposals != '[]'") || !reflect.DeepEqual(stepNames(report.Steps), []string{"Report exact-SHA release proposal status"}) {
		return &Error{Artifact: releaseWorkflowArtifact, Invariant: "isolates exact-run inspection and commit-status authority in the trusted release orchestrator"}
	}
	reportStep := report.Steps[0]
	reportRun := reportStep.Run
	reportDigestDetail := scriptDigestMismatchDetail(reportRun, reportProposalScriptDigest)
	validationPositions := make([]int, 0, 2)
	offset := 0
	for _, line := range strings.SplitAfter(reportRun, "\n") {
		if strings.TrimSpace(line) == "validate_proposal || fail_proposal" {
			validationPositions = append(validationPositions, offset)
		}
		offset += len(line)
	}
	fullRaceWait := strings.Index(reportRun, `full_race_run="$(find_run full-race.yml`)
	conclusionRead := strings.Index(reportRun, `verify_conclusion="$(jq`)
	revalidatesAfterWaiting := len(validationPositions) == 2 &&
		fullRaceWait >= 0 && validationPositions[1] > fullRaceWait &&
		conclusionRead >= 0 && validationPositions[1] < conclusionRead
	expectedReportEnv := map[string]string{
		"BASE_SHA": "${{ github.sha }}",
		"GH_TOKEN": "${{ secrets.GITHUB_TOKEN }}",
		"PROPOSAL": "${{ toJSON(matrix.proposal) }}",
		"RUN_URL":  "${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}",
	}
	if !hasScriptDigest(reportRun, reportProposalScriptDigest) || !revalidatesAfterWaiting || !reflect.DeepEqual(reportStep.Env, expectedReportEnv) || !defaultStepExecution(reportStep) || !containsAll(reportRun,
		"find_run()", "for _ in $(seq 1 240)", ".id > $after_id", ".head_sha == $sha", `.actor.login == "github-actions[bot]"`, `.triggering_actor.login == "github-actions[bot]"`, ".repository.full_name == env.GITHUB_REPOSITORY", "multiple ${workflow} runs match ${sha}",
		"validate_proposal()", "report_status()", "fail_proposal()", "report_status failure || true", "validate_proposal || fail_proposal",
		`test "$(gh api "repos/${GITHUB_REPOSITORY}/commits/main" --jq .sha)" = "${BASE_SHA}"`, `.author.login == "app/github-actions"`, ".baseRefOid == $base", ".headRefOid == $sha", ".merge_base_commit.sha == $base", ".files[] | {filename, status}",
		`verify_run="$(find_run verify.yml`, `full_race_run="$(find_run full-race.yml`, `test "${verify_conclusion}" = success && test "${full_race_conclusion}" = success`, `-f context=verify`, "exit 1") {
		return &Error{Artifact: releaseWorkflowArtifact, Invariant: "waits for trusted exact-SHA gates and reports only a revalidated proposal status", Detail: reportDigestDetail}
	}

	sdk := release.Jobs["sdk-release"]
	expectedSDKWith := map[string]any{
		"candidate_sha":  "${{ needs.release-please.outputs.sdk_sha }}",
		"expected_owner": "${{ vars.OPENDART_CRATES_IO_OWNER }}",
		"inventory_path": "sdk/rust/package-files.txt",
		"package":        "opendart",
		"package_path":   "sdk/rust/crates/opendart",
		"prerelease":     "${{ needs.release-please.outputs.sdk_prerelease == 'true' }}",
		"tag_name":       "${{ needs.release-please.outputs.sdk_tag_name }}",
		"vcs_path":       "sdk/rust/crates/opendart",
		"version":        "${{ needs.release-please.outputs.sdk_version }}",
	}
	if sdk.Uses != "./.github/workflows/rust-crate-release.yml" || !workflowNeedsExactly(sdk.Needs, "release-please") || !reflect.DeepEqual(sdk.Permissions, map[string]string{"contents": "write"}) || sdk.ContinueOnError || !defaultRunSettings(sdk.Defaults) || !exactWorkflowExpression(sdk.If, "needs.release-please.outputs.sdk_release_created == 'true'") || !reflect.DeepEqual(sdk.With, expectedSDKWith) {
		return &Error{Artifact: releaseWorkflowArtifact, Invariant: "calls only the protected SDK release workflow with fixed component identity"}
	}
	if err := checkActionPins(releaseWorkflowArtifact, release); err != nil {
		return err
	}
	if err := checkCheckoutCredentials(releaseWorkflowArtifact, release); err != nil {
		return err
	}
	if strings.Count(source, "actions: write") != 1 || strings.Count(source, "statuses: write") != 1 {
		return &Error{Artifact: releaseWorkflowArtifact, Invariant: "keeps registry credentials out and actions-write authority isolated"}
	}
	return nil
}

func containsAll(source string, required ...string) bool {
	for _, value := range required {
		if !strings.Contains(source, value) {
			return false
		}
	}
	return true
}

func checkRustCrateWorkflow(release workflow, source string) error {
	if release.Name != "Rust crate release" || !reflect.DeepEqual(sortedKeys(release.On), []string{"workflow_call"}) {
		return &Error{Artifact: rustCrateWorkflowArtifact, Invariant: "is callable only as the protected Rust crate release workflow"}
	}
	if err := checkActionPins(rustCrateWorkflowArtifact, release); err != nil {
		return err
	}
	workflowCall, ok := release.On["workflow_call"].(map[string]any)
	inputs, inputsOK := workflowCall["inputs"].(map[string]any)
	expectedInputs := []string{"candidate_sha", "expected_owner", "inventory_path", "package", "package_path", "prerelease", "tag_name", "vcs_path", "version"}
	if !ok || !inputsOK || !reflect.DeepEqual(sortedKeys(inputs), expectedInputs) {
		return &Error{Artifact: rustCrateWorkflowArtifact, Invariant: "accepts only the fixed release identity inputs"}
	}
	if _, exposesCallerSecrets := workflowCall["secrets"]; exposesCallerSecrets {
		return &Error{Artifact: rustCrateWorkflowArtifact, Invariant: "exposes no caller-provided registry credential interface"}
	}
	if len(release.Permissions) != 0 || release.Concurrency.Group != "rust-crate-${{ inputs.package }}-${{ inputs.version }}" || release.Concurrency.CancelInProgress {
		return &Error{Artifact: rustCrateWorkflowArtifact, Invariant: "starts without authority and serializes an exact crate version"}
	}
	if !reflect.DeepEqual(sortedKeys(release.Jobs), []string{"candidate", "finalize", "publish", "reconcile"}) {
		return &Error{Artifact: rustCrateWorkflowArtifact, Invariant: "contains only candidate, publish, reconcile, and finalize jobs"}
	}

	candidate := release.Jobs["candidate"]
	expectedCandidateSteps := []string{"Validate SDK release inputs", "Check out candidate revision", "Attest candidate revision and package version", "Install pinned Rust toolchain", "Run credential-free Rust verification", "Package and dry-run without credentials", "Prepare immutable candidate evidence", "Upload candidate evidence"}
	expectedCandidateOutputs := map[string]string{
		"artifact_digest": "${{ steps.upload.outputs['artifact-digest'] }}",
		"artifact_name":   "${{ steps.evidence.outputs.artifact_name }}",
	}
	if !workflowNeedsExactly(candidate.Needs) || !reflect.DeepEqual(candidate.Permissions, map[string]string{"contents": "read"}) || !reflect.DeepEqual(candidate.Outputs, expectedCandidateOutputs) || candidate.Environment != "" || candidate.RunsOn != "ubuntu-latest" || candidate.TimeoutMinutes != 35 || !defaultJobExecution(candidate) || !defaultRunSettings(candidate.Defaults) || !reflect.DeepEqual(stepNames(candidate.Steps), expectedCandidateSteps) {
		return &Error{Artifact: rustCrateWorkflowArtifact, Invariant: "builds candidate evidence without publication authority"}
	}
	expectedCandidateCheckout := map[string]any{"fetch-depth": 0, "persist-credentials": false, "ref": "${{ inputs.candidate_sha }}"}
	expectedCandidate := []releaseStepExpectation{
		{name: "Validate SDK release inputs", runDigest: candidateInputScriptDigest, env: map[string]string{
			"CANDIDATE_SHA": "${{ inputs.candidate_sha }}", "INVENTORY_PATH": "${{ inputs.inventory_path }}", "PACKAGE": "${{ inputs.package }}",
			"PACKAGE_PATH": "${{ inputs.package_path }}", "TAG_NAME": "${{ inputs.tag_name }}", "VCS_PATH": "${{ inputs.vcs_path }}", "VERSION": "${{ inputs.version }}",
		}},
		{name: "Check out candidate revision", uses: checkoutAction, with: expectedCandidateCheckout},
		{name: "Attest candidate revision and package version", runDigest: candidateAttestScriptDigest, env: map[string]string{
			"CANDIDATE_SHA": "${{ inputs.candidate_sha }}", "PACKAGE_PATH": "${{ inputs.package_path }}", "VERSION": "${{ inputs.version }}",
		}},
		{name: "Install pinned Rust toolchain", runDigest: candidateToolchainScriptDigest},
		{name: "Run credential-free Rust verification", runDigest: candidateVerifyScriptDigest},
		{name: "Package and dry-run without credentials", runDigest: candidatePackageScriptDigest, env: map[string]string{
			"INVENTORY_PATH": "${{ inputs.inventory_path }}", "PACKAGE": "${{ inputs.package }}", "PACKAGE_PATH": "${{ inputs.package_path }}", "VERSION": "${{ inputs.version }}",
		}},
		{name: "Prepare immutable candidate evidence", id: "evidence", runDigest: candidateEvidenceScriptDigest, env: map[string]string{
			"CANDIDATE_SHA": "${{ inputs.candidate_sha }}", "INVENTORY_PATH": "${{ inputs.inventory_path }}", "PACKAGE": "${{ inputs.package }}", "TAG_NAME": "${{ inputs.tag_name }}", "VERSION": "${{ inputs.version }}",
		}},
		{name: "Upload candidate evidence", id: "upload", uses: uploadArtifactAction, with: map[string]any{
			"name": "${{ steps.evidence.outputs.artifact_name }}", "path": "candidate-evidence", "if-no-files-found": "error", "retention-days": 30,
		}},
	}
	if ok, detail := exactReleaseSteps(candidate.Steps, expectedCandidate); !ok {
		return &Error{Artifact: rustCrateWorkflowArtifact, Invariant: "attests, verifies, packages, and records the exact SDK candidate", Detail: detail}
	}

	publish := release.Jobs["publish"]
	expectedPublishSteps := []string{"Validate publication identity", "Check out candidate revision", "Download candidate evidence", "Recheck candidate evidence", "Install pinned Rust toolchain", "Publish at most once and reconcile"}
	if !workflowNeedsExactly(publish.Needs, "candidate") || !reflect.DeepEqual(publish.Permissions, map[string]string{"contents": "read"}) || publish.Environment != "crates-io-opendart" || publish.RunsOn != "ubuntu-latest" || publish.TimeoutMinutes != 20 || !defaultJobExecution(publish) || !defaultRunSettings(publish.Defaults) || !reflect.DeepEqual(stepNames(publish.Steps), expectedPublishSteps) {
		return &Error{Artifact: rustCrateWorkflowArtifact, Invariant: "isolates registry authority in the literal protected publication environment"}
	}
	expectedPublishEnv := map[string]string{
		"CARGO_REGISTRY_TOKEN": "${{ secrets.CARGO_REGISTRY_TOKEN }}",
		"EXPECTED_OWNER":       "${{ inputs.expected_owner }}",
		"PACKAGE":              "${{ inputs.package }}",
		"PACKAGE_PATH":         "${{ inputs.package_path }}",
		"VERSION":              "${{ inputs.version }}",
	}
	expectedPublish := []releaseStepExpectation{
		{name: "Validate publication identity", runDigest: publishInputScriptDigest, env: map[string]string{
			"EXPECTED_OWNER": "${{ inputs.expected_owner }}", "PACKAGE": "${{ inputs.package }}", "VERSION": "${{ inputs.version }}",
		}},
		{name: "Check out candidate revision", uses: checkoutAction, with: expectedCandidateCheckout},
		{name: "Download candidate evidence", uses: downloadArtifactAction, with: map[string]any{
			"name": "${{ needs.candidate.outputs.artifact_name }}", "path": "candidate-evidence",
		}},
		{name: "Recheck candidate evidence", runDigest: publishEvidenceScriptDigest, env: map[string]string{
			"ARTIFACT_DIGEST": "${{ needs.candidate.outputs.artifact_digest }}", "CANDIDATE_SHA": "${{ inputs.candidate_sha }}", "PACKAGE": "${{ inputs.package }}", "TAG_NAME": "${{ inputs.tag_name }}", "VERSION": "${{ inputs.version }}",
		}},
		{name: "Install pinned Rust toolchain", runDigest: minimalToolchainScriptDigest},
		{name: "Publish at most once and reconcile", runDigest: publishScriptDigest, env: expectedPublishEnv},
	}
	if ok, detail := exactReleaseSteps(publish.Steps, expectedPublish); !ok {
		return &Error{Artifact: rustCrateWorkflowArtifact, Invariant: "publishes at most once and reconciles registry acceptance and ownership", Detail: detail}
	}

	reconcile := release.Jobs["reconcile"]
	expectedReconcileSteps := []string{"Check out candidate revision", "Download candidate evidence", "Set up Go", "Install pinned Rust toolchain", "Acquire and verify accepted crate", "Prove clean registry consumer", "Wait for public source and docs", "Upload accepted release evidence"}
	if !workflowNeedsExactly(reconcile.Needs, "candidate", "publish") || !reflect.DeepEqual(reconcile.Permissions, map[string]string{"contents": "read"}) || reconcile.Environment != "" || reconcile.RunsOn != "ubuntu-latest" || reconcile.TimeoutMinutes != 60 || !defaultJobExecution(reconcile) || !defaultRunSettings(reconcile.Defaults) || !reflect.DeepEqual(stepNames(reconcile.Steps), expectedReconcileSteps) {
		return &Error{Artifact: rustCrateWorkflowArtifact, Invariant: "reconciles accepted public artifacts without registry authority"}
	}
	expectedReconcile := []releaseStepExpectation{
		{name: "Check out candidate revision", uses: checkoutAction, with: expectedCandidateCheckout},
		{name: "Download candidate evidence", uses: downloadArtifactAction, with: map[string]any{
			"name": "${{ needs.candidate.outputs.artifact_name }}", "path": "candidate-evidence",
		}},
		{name: "Set up Go", uses: setupGoAction, with: map[string]any{"go-version-file": "go.mod", "cache": true}},
		{name: "Install pinned Rust toolchain", runDigest: minimalToolchainScriptDigest},
		{name: "Acquire and verify accepted crate", runDigest: reconcileCrateScriptDigest, env: map[string]string{
			"CANDIDATE_SHA": "${{ inputs.candidate_sha }}", "PACKAGE": "${{ inputs.package }}", "VCS_PATH": "${{ inputs.vcs_path }}", "VERSION": "${{ inputs.version }}",
		}},
		{name: "Prove clean registry consumer", runDigest: reconcileConsumerScriptDigest, env: map[string]string{
			"PACKAGE": "${{ inputs.package }}", "VERSION": "${{ inputs.version }}",
		}},
		{name: "Wait for public source and docs", runDigest: reconcileReadinessScriptDigest, env: map[string]string{
			"PACKAGE": "${{ inputs.package }}", "VERSION": "${{ inputs.version }}",
		}},
		{name: "Upload accepted release evidence", uses: uploadArtifactAction, with: map[string]any{
			"name":              "accepted-${{ needs.candidate.outputs.artifact_name }}",
			"path":              "accepted.crate\ncrate-verification.json\nregistry-version.json\n",
			"if-no-files-found": "error", "retention-days": 90,
		}},
	}
	if ok, detail := exactReleaseSteps(reconcile.Steps, expectedReconcile); !ok {
		return &Error{Artifact: rustCrateWorkflowArtifact, Invariant: "verifies the accepted crate, clean consumer, registry page, and docs", Detail: detail}
	}

	finalize := release.Jobs["finalize"]
	expectedFinalizeEnv := map[string]string{"CANDIDATE_SHA": "${{ inputs.candidate_sha }}", "EXPECTED_PRERELEASE": "${{ inputs.prerelease }}", "GH_TOKEN": "${{ github.token }}", "TAG_NAME": "${{ inputs.tag_name }}"}
	expectedFinalize := []releaseStepExpectation{{name: "Validate and publish matching GitHub draft", runDigest: finalizeScriptDigest, env: expectedFinalizeEnv}}
	finalizeStepsOK, finalizeDetail := exactReleaseSteps(finalize.Steps, expectedFinalize)
	if !workflowNeedsExactly(finalize.Needs, "reconcile") || !reflect.DeepEqual(finalize.Permissions, map[string]string{"contents": "write"}) || finalize.Environment != "" || finalize.RunsOn != "ubuntu-latest" || finalize.TimeoutMinutes != 10 || !defaultJobExecution(finalize) || !defaultRunSettings(finalize.Defaults) || !finalizeStepsOK || !containsAll(finalize.Steps[0].Run, ".targetCommitish == $sha", "probe_tag_ref()", "gh api --include", "if test \"${http_status}\" = 404", "tag_ref_status=\"$(probe_tag_ref \"${TAG_NAME}\")\"", "case \"$(jq -r .isDraft", "true)", "false)\n    test \"$(gh api", "--draft=false", "--latest=false", "commits/${TAG_NAME}") {
		return &Error{Artifact: rustCrateWorkflowArtifact, Invariant: "publishes only the matching GitHub draft after registry reconciliation", Detail: finalizeDetail}
	}

	if err := checkCheckoutCredentials(rustCrateWorkflowArtifact, release); err != nil {
		return err
	}
	if strings.Contains(source, "id-token: write") || strings.Contains(source, "secrets: inherit") || strings.Count(source, "${{ secrets.CARGO_REGISTRY_TOKEN }}") != 1 || strings.Count(source, "cargo +1.97.1 publish --locked --no-verify") != 1 || strings.Count(source, "contents: write") != 1 {
		return &Error{Artifact: rustCrateWorkflowArtifact, Invariant: "confines registry and GitHub publication authority to single exact boundaries"}
	}
	for jobName, job := range release.Jobs {
		for _, step := range job.Steps {
			if jobName != "publish" && strings.Contains(step.Run, "cargo +1.97.1 publish --locked --no-verify") {
				return &Error{Artifact: rustCrateWorkflowArtifact, Invariant: "confines registry publication to the protected publication step"}
			}
			if !defaultStepExecution(step) || !defaultStepRunSettings(step) {
				return &Error{Artifact: rustCrateWorkflowArtifact, Invariant: "release steps fail closed with default run settings", Detail: jobName + ": " + step.Name}
			}
		}
	}
	return nil
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
	if err := require(
		verifyWorkflowArtifact,
		"cancels only superseded verification runs",
		verify.Concurrency.Group == verifyConcurrencyGroup && verify.Concurrency.CancelInProgress,
		"",
	); err != nil {
		return err
	}
	expectedTriggers := []string{"pull_request", "workflow_call", "workflow_dispatch"}
	for _, trigger := range expectedTriggers {
		if _, exists := verify.On[trigger]; !exists {
			return &Error{Artifact: verifyWorkflowArtifact, Invariant: "supports " + trigger}
		}
	}
	if err := require(
		verifyWorkflowArtifact,
		"supports only approved triggers",
		reflect.DeepEqual(sortedKeys(verify.On), expectedTriggers),
		"",
	); err != nil {
		return err
	}
	if err := require(verifyWorkflowArtifact, "contains only approved verification jobs", reflect.DeepEqual(sortedKeys(verify.Jobs), []string{"artifact-macos", "artifact-windows", "go", "rust", "verify"}), ""); err != nil {
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
	if err := checkCredentialFreeSource(verifyWorkflowArtifact, source); err != nil {
		return err
	}
	checkout := workflowStepExpectation{
		name: "Check out repository",
		uses: "actions/checkout",
		with: map[string]any{"fetch-depth": 0, "persist-credentials": false, "ref": "${{ inputs.expected_sha || github.sha }}"},
	}
	attest := workflowStepExpectation{name: "Attest checked-out revision", run: `test "${GITHUB_SHA}" = "${EXPECTED_SHA}"
test "$(git rev-parse HEAD)" = "${EXPECTED_SHA}"`, env: map[string]string{"EXPECTED_SHA": "${{ inputs.expected_sha || github.sha }}"}}
	goSteps := []workflowStepExpectation{
		checkout,
		attest,
		{name: "Set up Go", uses: "actions/setup-go", with: map[string]any{"go-version-file": "go.mod", "cache": true}},
		{name: "Run required Go verification", run: "./scripts/verify go"},
	}
	rustSteps := []workflowStepExpectation{
		checkout,
		attest,
		{name: "Run required Rust verification", run: "./scripts/verify rust"},
	}
	if err := checkVerificationJob("go", verify.Jobs["go"], goSteps); err != nil {
		return err
	}
	if err := checkVerificationJob("rust", verify.Jobs["rust"], rustSteps); err != nil {
		return err
	}
	if err := checkVerifyAggregateJob(verify.Jobs["verify"]); err != nil {
		return err
	}
	if err := checkActionPins(verifyWorkflowArtifact, verify); err != nil {
		return err
	}
	if err := checkCheckoutCredentials(verifyWorkflowArtifact, verify); err != nil {
		return err
	}
	return nil
}

func checkVerificationJob(name string, job workflowJob, expected []workflowStepExpectation) error {
	if !workflowNeedsExactly(job.Needs) || !defaultJobExecution(job) || job.Environment != "" || job.Uses != "" || len(job.Permissions) != 0 {
		return &Error{Artifact: verifyWorkflowArtifact, Invariant: "verification work jobs use default execution controls", Detail: name}
	}
	if !defaultRunSettings(job.Defaults) {
		return &Error{Artifact: verifyWorkflowArtifact, Invariant: "verification work jobs use default run settings", Detail: name}
	}
	if job.RunsOn != "ubuntu-latest" || job.TimeoutMinutes != 30 {
		return &Error{Artifact: verifyWorkflowArtifact, Invariant: "verification work jobs use the approved runner and timeout", Detail: name}
	}
	if len(job.Steps) != len(expected) {
		return &Error{Artifact: verifyWorkflowArtifact, Invariant: "uses only the approved verification steps", Detail: name}
	}
	for index, want := range expected {
		step := job.Steps[index]
		if step.Name != want.name || !exactScript(step.Run, want.run) {
			return &Error{Artifact: verifyWorkflowArtifact, Invariant: "uses only the approved verification steps", Detail: name + ": " + step.Name}
		}
		if want.uses == "" {
			if step.Uses != "" || len(step.With) != 0 {
				return &Error{Artifact: verifyWorkflowArtifact, Invariant: "uses only the approved verification steps", Detail: name + ": " + step.Name}
			}
		} else if !strings.HasPrefix(step.Uses, want.uses+"@") || !reflect.DeepEqual(step.With, want.with) {
			return &Error{Artifact: verifyWorkflowArtifact, Invariant: "uses only the approved verification actions", Detail: name + ": " + step.Name}
		}
		if !defaultStepExecution(step) {
			return &Error{Artifact: verifyWorkflowArtifact, Invariant: "verification steps use default execution controls", Detail: name + ": " + step.Name}
		}
		if !defaultStepRunSettings(step) {
			return &Error{Artifact: verifyWorkflowArtifact, Invariant: "verification steps use default run settings", Detail: name + ": " + step.Name}
		}
		if !reflect.DeepEqual(step.Env, want.env) || step.Shell != want.shell {
			return &Error{Artifact: verifyWorkflowArtifact, Invariant: "verification steps use only the approved environment and shell", Detail: name + ": " + step.Name}
		}
	}
	return nil
}

func checkVerifyAggregateJob(job workflowJob) error {
	requiredJobs := []string{"go", "rust", "artifact-macos", "artifact-windows"}
	if !exactWorkflowExpression(job.If, "always()") || job.ContinueOnError || job.Environment != "" || job.Uses != "" || len(job.Permissions) != 0 {
		return &Error{Artifact: verifyWorkflowArtifact, Invariant: "aggregate verify job always evaluates dependency results"}
	}
	if !workflowNeedsExactly(job.Needs, requiredJobs...) {
		return &Error{Artifact: verifyWorkflowArtifact, Invariant: "aggregate verify job depends on every required job"}
	}
	if !defaultRunSettings(job.Defaults) || job.RunsOn != "ubuntu-latest" || job.TimeoutMinutes != 5 {
		return &Error{Artifact: verifyWorkflowArtifact, Invariant: "aggregate verify job uses only approved runtime settings"}
	}
	if len(job.Steps) != 1 {
		return &Error{Artifact: verifyWorkflowArtifact, Invariant: "aggregate verify job has only the result check"}
	}
	step := job.Steps[0]
	expectedEnvironment := map[string]string{
		"GO_RESULT":      "${{ needs.go.result }}",
		"RUST_RESULT":    "${{ needs.rust.result }}",
		"MACOS_RESULT":   "${{ needs.artifact-macos.result }}",
		"WINDOWS_RESULT": "${{ needs.artifact-windows.result }}",
	}
	if step.Name != "Require successful verification jobs" ||
		!exactScript(step.Run, verifyAggregateScript) ||
		!reflect.DeepEqual(step.Env, expectedEnvironment) ||
		step.Uses != "" ||
		len(step.With) != 0 ||
		!defaultStepExecution(step) ||
		!defaultStepRunSettings(step) {
		return &Error{Artifact: verifyWorkflowArtifact, Invariant: "aggregate verify job rejects every non-success result"}
	}
	return nil
}

func checkNativeArtifactJob(name, runner string, job workflowJob) error {
	if !defaultJobExecution(job) || !workflowNeedsExactly(job.Needs) || job.Environment != "" || job.Uses != "" || len(job.Permissions) != 0 || !defaultRunSettings(job.Defaults) {
		return &Error{Artifact: verifyWorkflowArtifact, Invariant: "native artifact jobs use default execution controls", Detail: name}
	}
	if job.RunsOn != runner || job.TimeoutMinutes != 20 {
		return &Error{Artifact: verifyWorkflowArtifact, Invariant: "native artifact jobs use approved runners and timeouts", Detail: name}
	}
	installScript := sourceInstallScript
	if runner == "windows-latest" {
		installScript = windowsSourceInstallScript
	}
	expected := []workflowStepExpectation{
		{name: "Check out repository", uses: "actions/checkout", with: map[string]any{"fetch-depth": 0, "persist-credentials": false, "ref": "${{ inputs.expected_sha || github.sha }}"}},
		{name: "Attest checked-out revision", run: `test "${GITHUB_SHA}" = "${EXPECTED_SHA}"
test "$(git rev-parse HEAD)" = "${EXPECTED_SHA}"`, env: map[string]string{"EXPECTED_SHA": "${{ inputs.expected_sha || github.sha }}"}},
		{name: "Install pinned Rust toolchain", run: "rustup toolchain install 1.97.1 --profile minimal"},
		{name: "Fetch locked Rust dependencies", run: nativeArtifactFetchScript},
		{name: "Verify native binary artifact behavior", run: nativeArtifactTestScript, env: map[string]string{"RUSTFLAGS": "--cfg opendart_compat"}},
		{name: "Install CLI from reviewed source offline", run: installScript},
	}
	if runner == "windows-latest" {
		expected[1].shell = "bash"
	}
	if len(job.Steps) != len(expected) {
		return &Error{Artifact: verifyWorkflowArtifact, Invariant: "native artifact jobs use only approved steps", Detail: name}
	}
	for index, want := range expected {
		step := job.Steps[index]
		if step.Name != want.name || !exactScript(step.Run, want.run) || !reflect.DeepEqual(step.Env, want.env) || step.Shell != want.shell || !defaultStepExecution(step) || strings.TrimSpace(step.WorkingDirectory) != "" {
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
	if !standardJobControls(job, "ubuntu-latest", 30) {
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
	if err := checkStepPolicies(liveWorkflowArtifact, job.Steps, "Run live conformance and Rust CLI smoke", "producer steps use default run settings", "API key is absent outside the request boundary"); err != nil {
		return err
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
	if !standardJobControls(job, "ubuntu-latest", 20) {
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
	if err := checkStepPolicies(notifyWorkflowArtifact, job.Steps, "Update live conformance issue", "notifier steps use default run settings", "job token environment is confined to the notifier boundary"); err != nil {
		return err
	}
	if strings.Contains(source, "OPENDART_API_KEY") || strings.Contains(source, "secrets.") || strings.Contains(source, "secrets[") || strings.Contains(source, "environment:") {
		return &Error{Artifact: notifyWorkflowArtifact, Invariant: "notifier cannot access the OpenDART credential or protected environment"}
	}
	if err := checkActionPins(notifyWorkflowArtifact, notify); err != nil {
		return err
	}
	return checkCheckoutCredentials(notifyWorkflowArtifact, notify)
}

func checkDriftWorkflow(drift workflow, source string) error {
	if err := require(driftWorkflowArtifact, "has the expected workflow name", drift.Name == "Public Guide Drift", ""); err != nil {
		return err
	}
	if err := require(driftWorkflowArtifact, "is manual only", reflect.DeepEqual(sortedKeys(drift.On), []string{"workflow_dispatch"}), ""); err != nil {
		return err
	}
	if err := require(driftWorkflowArtifact, "root permissions are empty", len(drift.Permissions) == 0, ""); err != nil {
		return err
	}
	if err := require(driftWorkflowArtifact, "serializes public guide drift runs", drift.Concurrency.Group == "public-guide-drift" && !drift.Concurrency.CancelInProgress, ""); err != nil {
		return err
	}
	if err := require(driftWorkflowArtifact, "workflow uses default run settings", defaultRunSettings(drift.Defaults), ""); err != nil {
		return err
	}
	if err := require(driftWorkflowArtifact, "contains only the drift job", reflect.DeepEqual(sortedKeys(drift.Jobs), []string{"drift"}), ""); err != nil {
		return err
	}
	job := drift.Jobs["drift"]
	if !exactWorkflowExpression(job.If, "github.repository == 'cpaikr/opendart' && github.ref == 'refs/heads/main'") {
		return &Error{Artifact: driftWorkflowArtifact, Invariant: "runs only trusted main code in the canonical repository"}
	}
	if !reflect.DeepEqual(job.Permissions, map[string]string{"contents": "read"}) || job.Environment != "" {
		return &Error{Artifact: driftWorkflowArtifact, Invariant: "uses only read-only repository authority without a protected environment"}
	}
	if !standardJobControls(job, "ubuntu-latest", 30) {
		return &Error{Artifact: driftWorkflowArtifact, Invariant: "producer uses only approved execution controls"}
	}
	expectedNames := []string{"Check out trusted revision", "Set up Go", "Recheck credential-free repository gates", "Compare the public guide", "Upload sanitized report"}
	if !reflect.DeepEqual(stepNames(job.Steps), expectedNames) {
		return &Error{Artifact: driftWorkflowArtifact, Invariant: "uses only approved producer steps in order"}
	}
	checkout := job.Steps[0]
	setup := job.Steps[1]
	preflight := job.Steps[2]
	run := job.Steps[3]
	upload := job.Steps[4]
	if checkout.Uses != checkoutAction || checkout.Run != "" || !reflect.DeepEqual(checkout.With, map[string]any{"fetch-depth": 0, "persist-credentials": false}) || !defaultStepExecution(checkout) {
		return &Error{Artifact: driftWorkflowArtifact, Invariant: "checks out the trusted dispatched revision without credentials"}
	}
	if setup.Uses != setupGoAction || setup.Run != "" || !reflect.DeepEqual(setup.With, map[string]any{"cache": true, "go-version-file": "go.mod"}) || !defaultStepExecution(setup) {
		return &Error{Artifact: driftWorkflowArtifact, Invariant: "uses the approved Go setup"}
	}
	if preflight.Run != "go run ./cmd/opendart-tool verify --repository-root ." || preflight.Uses != "" || !defaultStepExecution(preflight) {
		return &Error{Artifact: driftWorkflowArtifact, Invariant: "rechecks credential-free repository gates before acquisition"}
	}
	if run.Run != driftRunScript || run.Uses != "" || !defaultStepExecution(run) {
		return &Error{Artifact: driftWorkflowArtifact, Invariant: "runs only the canonical credential-free drift command"}
	}
	expectedUpload := map[string]any{
		"name": "guide-drift-report-${{ github.run_attempt }}", "path": "guide-drift-report.json",
		"if-no-files-found": "error", "retention-days": 7, "compression-level": 0,
		"overwrite": false, "include-hidden-files": false,
	}
	if upload.Uses != uploadArtifactAction || upload.Run != "" || !exactWorkflowExpression(upload.If, "always()") || upload.ContinueOnError || !reflect.DeepEqual(upload.With, expectedUpload) {
		return &Error{Artifact: driftWorkflowArtifact, Invariant: "uploads only the bounded sanitized report"}
	}
	if err := checkStepPolicies(driftWorkflowArtifact, job.Steps, "", "producer steps use default credential-free execution settings", "producer steps use default credential-free execution settings"); err != nil {
		return err
	}
	if strings.Contains(source, "secrets.") || strings.Contains(source, "secrets[") || strings.Contains(source, "github.token") || strings.Contains(source, "GITHUB_TOKEN") || strings.Contains(source, "issues: write") || strings.Contains(source, "environment:") || strings.Contains(source, "OPENDART_API_KEY") {
		return &Error{Artifact: driftWorkflowArtifact, Invariant: "contains no credentials, issue authority, or protected environment"}
	}
	if err := checkActionPins(driftWorkflowArtifact, drift); err != nil {
		return err
	}
	return checkCheckoutCredentials(driftWorkflowArtifact, drift)
}

func checkDriftNotifyWorkflow(notify workflow, source string) error {
	if err := require(driftNotifyArtifact, "has the expected workflow name", notify.Name == "Public Guide Drift Notifier", ""); err != nil {
		return err
	}
	workflowRun, ok := notify.On["workflow_run"].(map[string]any)
	expectedTrigger := map[string]any{"types": []any{"completed"}, "workflows": []any{"Public Guide Drift"}}
	if err := require(driftNotifyArtifact, "runs only after the public guide drift producer completes", reflect.DeepEqual(sortedKeys(notify.On), []string{"workflow_run"}) && ok && reflect.DeepEqual(workflowRun, expectedTrigger), ""); err != nil {
		return err
	}
	if err := require(driftNotifyArtifact, "root permissions are empty", len(notify.Permissions) == 0, ""); err != nil {
		return err
	}
	if err := require(driftNotifyArtifact, "serializes notifier runs", notify.Concurrency.Group == "public-guide-drift-notifier" && !notify.Concurrency.CancelInProgress, ""); err != nil {
		return err
	}
	if err := require(driftNotifyArtifact, "workflow uses default run settings", defaultRunSettings(notify.Defaults), ""); err != nil {
		return err
	}
	if err := require(driftNotifyArtifact, "contains only the notifier job", reflect.DeepEqual(sortedKeys(notify.Jobs), []string{"notify"}), ""); err != nil {
		return err
	}
	job := notify.Jobs["notify"]
	expectedCondition := "github.repository == 'cpaikr/opendart' && github.event.workflow_run.event == 'workflow_dispatch' && github.event.workflow_run.head_repository.full_name == github.repository && github.event.workflow_run.head_branch == github.event.repository.default_branch"
	if !exactWorkflowExpression(job.If, expectedCondition) {
		return &Error{Artifact: driftNotifyArtifact, Invariant: "accepts only manual trusted default-branch producer runs"}
	}
	expectedPermissions := map[string]string{"actions": "read", "contents": "read", "issues": "write"}
	if !reflect.DeepEqual(job.Permissions, expectedPermissions) || job.Environment != "" {
		return &Error{Artifact: driftNotifyArtifact, Invariant: "isolates minimal issue authority without a protected environment"}
	}
	if !standardJobControls(job, "ubuntu-latest", 20) {
		return &Error{Artifact: driftNotifyArtifact, Invariant: "notifier uses only approved execution controls"}
	}
	expectedNames := []string{"Check out trusted producer revision", "Set up Go", "Download sanitized report", "Update public guide drift issue"}
	if !reflect.DeepEqual(stepNames(job.Steps), expectedNames) {
		return &Error{Artifact: driftNotifyArtifact, Invariant: "uses only approved notifier steps in order"}
	}
	checkout := job.Steps[0]
	setup := job.Steps[1]
	download := job.Steps[2]
	run := job.Steps[3]
	if checkout.Uses != checkoutAction || checkout.Run != "" || !defaultStepExecution(checkout) || !reflect.DeepEqual(checkout.With, map[string]any{"persist-credentials": false, "ref": "${{ github.event.workflow_run.head_sha }}"}) {
		return &Error{Artifact: driftNotifyArtifact, Invariant: "checks out the exact trusted producer revision without credentials"}
	}
	if setup.Uses != setupGoAction || setup.Run != "" || !defaultStepExecution(setup) || !reflect.DeepEqual(setup.With, map[string]any{"cache": true, "go-version-file": "go.mod"}) {
		return &Error{Artifact: driftNotifyArtifact, Invariant: "uses the approved Go setup"}
	}
	expectedDownload := map[string]any{
		"name": "guide-drift-report-${{ github.event.workflow_run.run_attempt }}", "path": ".", "github-token": "${{ github.token }}",
		"run-id": "${{ github.event.workflow_run.id }}",
	}
	if download.ID != "report" || download.Uses != downloadArtifactAction || download.Run != "" || !download.ContinueOnError || strings.TrimSpace(download.If) != "" || !reflect.DeepEqual(download.With, expectedDownload) {
		return &Error{Artifact: driftNotifyArtifact, Invariant: "downloads only the producer report with fixed-failure fallback"}
	}
	expectedEnv := map[string]string{
		"GITHUB_TOKEN":               "${{ github.token }}",
		"NOTIFY_REPOSITORY":          "${{ github.repository }}",
		"NOTIFY_PRODUCER_CONCLUSION": "${{ github.event.workflow_run.conclusion }}",
		"NOTIFY_ARTIFACT_OUTCOME":    "${{ steps.report.outcome }}",
		"NOTIFY_RUN_ID":              "${{ github.event.workflow_run.id }}",
		"NOTIFY_RUN_ATTEMPT":         "${{ github.event.workflow_run.run_attempt }}",
	}
	if !exactScript(run.Run, driftNotifyRunScript) || run.Uses != "" || !exactWorkflowExpression(run.If, "always()") || run.ContinueOnError || !reflect.DeepEqual(run.Env, expectedEnv) {
		return &Error{Artifact: driftNotifyArtifact, Invariant: "invokes only the isolated notifier with trusted metadata"}
	}
	if err := checkStepPolicies(driftNotifyArtifact, job.Steps, "Update public guide drift issue", "notifier steps use default run settings", "job token environment is confined to the notifier boundary"); err != nil {
		return err
	}
	if strings.Contains(source, "OPENDART_API_KEY") || strings.Contains(source, "secrets.") || strings.Contains(source, "secrets[") || strings.Contains(source, "environment:") {
		return &Error{Artifact: driftNotifyArtifact, Invariant: "notifier cannot access OpenDART credentials or a protected environment"}
	}
	if err := checkActionPins(driftNotifyArtifact, notify); err != nil {
		return err
	}
	return checkCheckoutCredentials(driftNotifyArtifact, notify)
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

func exactWorkflowExpression(condition, expected string) bool {
	condition = strings.TrimSpace(condition)
	if strings.HasPrefix(condition, "${{") && strings.HasSuffix(condition, "}}") {
		condition = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(condition, "${{"), "}}"))
	}
	return strings.Join(strings.Fields(condition), " ") == expected
}

func workflowNeedsExactly(actual workflowNeeds, expected ...string) bool {
	if len(actual) != len(expected) {
		return false
	}
	for index := range expected {
		if actual[index] != expected[index] {
			return false
		}
	}
	return true
}

func defaultJobExecution(job workflowJob) bool {
	return strings.TrimSpace(job.If) == "" && !job.ContinueOnError
}

func standardJobControls(job workflowJob, runsOn string, timeoutMinutes int) bool {
	return workflowNeedsExactly(job.Needs) && !job.ContinueOnError && job.Uses == "" &&
		defaultRunSettings(job.Defaults) && job.RunsOn == runsOn && job.TimeoutMinutes == timeoutMinutes
}

func checkStepPolicies(artifact string, steps []workflowStep, environmentStep, runInvariant, environmentInvariant string) error {
	for _, step := range steps {
		if !defaultStepRunSettings(step) {
			return &Error{Artifact: artifact, Invariant: runInvariant, Detail: "step " + step.Name}
		}
		if step.Name != environmentStep && len(step.Env) != 0 {
			return &Error{Artifact: artifact, Invariant: environmentInvariant, Detail: "step " + step.Name}
		}
	}
	return nil
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

func exactScript(actual, expected string) bool {
	normalize := func(script string) string {
		return strings.TrimSpace(strings.ReplaceAll(script, "\r\n", "\n"))
	}
	return normalize(actual) == normalize(expected)
}

func hasScriptDigest(script, expected string) bool {
	return scriptDigest(script) == expected
}

func scriptDigest(script string) string {
	normalized := strings.TrimSpace(strings.ReplaceAll(script, "\r\n", "\n"))
	return fmt.Sprintf("%x", sha256.Sum256([]byte(normalized)))
}

func scriptDigestMismatchDetail(script, expected string) string {
	actual := scriptDigest(script)
	if actual == expected {
		return ""
	}
	return "computed script SHA-256: " + actual
}

func exactReleaseSteps(actual []workflowStep, expected []releaseStepExpectation) (bool, string) {
	if len(actual) != len(expected) {
		return false, ""
	}
	for index, want := range expected {
		step := actual[index]
		if step.Name != want.name || step.ID != want.id || step.Uses != want.uses ||
			!reflect.DeepEqual(step.With, want.with) || !reflect.DeepEqual(step.Env, want.env) ||
			!defaultStepExecution(step) || !defaultStepRunSettings(step) {
			return false, ""
		}
		if want.uses != "" {
			if step.Run != "" || want.runDigest != "" {
				return false, ""
			}
			continue
		}
		if step.Run == "" || want.runDigest == "" {
			return false, ""
		}
		if !hasScriptDigest(step.Run, want.runDigest) {
			return false, step.Name + ": " + scriptDigestMismatchDetail(step.Run, want.runDigest)
		}
	}
	return true, ""
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
