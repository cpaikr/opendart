package releaseguard

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

func TestCheckAcceptsRepositoryReleasePolicy(t *testing.T) {
	if err := Check(repositoryRoot(t)); err != nil {
		t.Fatalf("Check() error = %v", err)
	}
}

func TestSemanticVersionPolicy(t *testing.T) {
	for _, version := range []string{"0.1.0", "1.2.3-rc.1", "1.2.3+build.01", "1.2.3-1a"} {
		if !semanticVersion.MatchString(version) {
			t.Errorf("semanticVersion does not accept %q", version)
		}
	}
	for _, version := range []string{"01.2.3", "1.02.3", "1.2.03", "1.2.3-01", "1.2.3-"} {
		if semanticVersion.MatchString(version) {
			t.Errorf("semanticVersion accepts invalid %q", version)
		}
	}
}

func TestCheckRejectsRustReleaseManifestUntilPublicationRecoveryExists(t *testing.T) {
	root := copyReleaseArtifacts(t)
	path := filepath.Join(root, manifestArtifact)
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.Replace(
		string(source),
		`"openapi/generated": "0.1.0"`,
		`"openapi/generated": "0.1.0", "sdk/rust/crates/opendart": "0.1.0"`,
		1,
	)
	if err := os.WriteFile(path, []byte(updated), 0o600); err != nil {
		t.Fatal(err)
	}
	err = Check(root)
	var guardError *Error
	if !errors.As(err, &guardError) || guardError.Artifact != manifestArtifact || !strings.Contains(guardError.Invariant, "existing-draft recovery") {
		t.Fatalf("Check() error = %#v", err)
	}
}

func TestCheckRejectsRustReleaseOwnershipMutations(t *testing.T) {
	tests := []struct {
		name        string
		artifact    string
		old         string
		replacement string
		invariant   string
	}{
		{name: "Rust release type", artifact: configArtifact, old: `"release-type": "rust"`, replacement: `"release-type": "simple"`, invariant: "Rust package release-type"},
		{name: "Rust component", artifact: configArtifact, old: `"component": "opendart"`, replacement: `"component": "other"`, invariant: "Rust package component"},
		{name: "Rust component tag", artifact: configArtifact, old: `"component": "opendart",
      "include-component-in-tag": true`, replacement: `"component": "opendart",
      "include-component-in-tag": false`, invariant: "Rust package include-component-in-tag"},
		{name: "Rust forced tag", artifact: configArtifact, old: `"force-tag-creation": false`, replacement: `"force-tag-creation": true`, invariant: "Rust package force-tag-creation"},
		{name: "Rust lock path", artifact: configArtifact, old: `"path": "/sdk/rust/Cargo.lock"`, replacement: `"path": "Cargo.lock"`, invariant: "updates the workspace lock version"},
		{name: "Rust lock selector", artifact: configArtifact, old: `$.package[?(@.name == \"opendart\")].version`, replacement: `$.package.version`, invariant: "updates the workspace lock version"},
		{name: "Cargo lock mismatch", artifact: rustLockArtifact, old: "name = \"opendart\"\nversion = \"0.1.0\"", replacement: "name = \"opendart\"\nversion = \"0.1.1\"", invariant: "matches the crate package version"},
		{name: "registry publish in release", artifact: releaseWorkflowArtifact, old: "mkdir release-assets", replacement: "cargo publish\n          mkdir release-assets", invariant: "does not publish packages"},
		{name: "registry publish in verify", artifact: verifyWorkflowArtifact, old: "go vet ./...", replacement: "cargo publish", invariant: "excludes package publication"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := copyReleaseArtifacts(t)
			path := filepath.Join(root, filepath.FromSlash(test.artifact))
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			updated := strings.Replace(string(source), test.old, test.replacement, 1)
			if updated == string(source) {
				t.Fatalf("mutation source %q not found in %s", test.old, test.artifact)
			}
			if err := os.WriteFile(path, []byte(updated), 0o600); err != nil {
				t.Fatal(err)
			}
			err = Check(root)
			var guardError *Error
			if !errors.As(err, &guardError) || guardError.Artifact != test.artifact || !strings.Contains(guardError.Invariant, test.invariant) {
				t.Fatalf("Check() error = %#v, want %s invariant containing %q", err, test.artifact, test.invariant)
			}
		})
	}
}

func TestCheckRejectsRustPackageMutations(t *testing.T) {
	tests := []struct {
		name        string
		artifact    string
		old         string
		replacement string
		invariant   string
	}{
		{
			name: "registry scope", artifact: rustCargoArtifact,
			old: `publish = ["crates-io"]`, replacement: `publish = true`,
			invariant: "authorizes only the crates.io registry",
		},
		{
			name: "release documentation", artifact: rustCargoArtifact,
			old: `, "CHANGELOG.md"`, replacement: ``,
			invariant: "packages release documentation and provenance",
		},
		{
			name: "required package evidence", artifact: rustPackageListArtifact,
			old: "src/provenance.rs\n", replacement: "",
			invariant: "contains required package evidence",
		},
		{
			name: "private package input", artifact: rustPackageListArtifact,
			old: "tests/public_contract.rs\n", replacement: "target/secret\ntests/public_contract.rs\n",
			invariant: "excludes repository-private inputs",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := copyReleaseArtifacts(t)
			path := filepath.Join(root, filepath.FromSlash(test.artifact))
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			updated := strings.Replace(string(source), test.old, test.replacement, 1)
			if updated == string(source) {
				t.Fatalf("mutation source %q not found in %s", test.old, test.artifact)
			}
			if err := os.WriteFile(path, []byte(updated), 0o600); err != nil {
				t.Fatal(err)
			}
			err = Check(root)
			var guardError *Error
			if !errors.As(err, &guardError) || guardError.Artifact != test.artifact || !strings.Contains(guardError.Invariant, test.invariant) {
				t.Fatalf("Check() error = %#v, want %s invariant containing %q", err, test.artifact, test.invariant)
			}
		})
	}
}

func TestCheckRejectsRustPackageBundleProvenanceMismatch(t *testing.T) {
	root := copyReleaseArtifacts(t)
	path := filepath.Join(root, rustProvenanceArtifact)
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := os.ReadFile(filepath.Join(root, canonicalBundleArtifact))
	if err != nil {
		t.Fatal(err)
	}
	checksum := fmt.Sprintf("%x", sha256.Sum256(bundle))
	if !strings.Contains(string(source), checksum) {
		t.Fatal("fixture provenance does not contain the canonical bundle checksum")
	}
	replacement := "0" + checksum[1:]
	if replacement == checksum {
		replacement = "1" + checksum[1:]
	}
	spoofed := strings.Replace(string(source), checksum, replacement, 1) + "\n// stale checksum: " + checksum + "\n"
	if err := os.WriteFile(path, []byte(spoofed), 0o600); err != nil {
		t.Fatal(err)
	}
	err = Check(root)
	var guardError *Error
	if !errors.As(err, &guardError) || guardError.Artifact != rustProvenanceArtifact || !strings.Contains(guardError.Invariant, "matches the canonical bundle SHA-256") {
		t.Fatalf("Check() error = %#v", err)
	}
}

func TestCheckAllowsSpecificationSourcesToAdvanceAfterSelectedRelease(t *testing.T) {
	root := copyReleaseArtifacts(t)
	path := filepath.Join(root, "openapi", "components", "schemas.yaml")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(source, []byte("\n# changed after the selected source release\n")...), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := Check(root); err != nil {
		t.Fatalf("Check() rejected post-release specification evolution: %v", err)
	}
}

func TestCheckRejectsUnavailableSpecificationSourceRelease(t *testing.T) {
	root := copyReleaseArtifacts(t)
	path := filepath.Join(root, rustProvenanceArtifact)
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.Replace(string(source), `Some("v0.1.0")`, `Some("v9.9.9")`, 1)
	if updated == string(source) {
		t.Fatal("fixture provenance does not contain the specification source release")
	}
	if err := os.WriteFile(path, []byte(updated), 0o600); err != nil {
		t.Fatal(err)
	}

	err = Check(root)
	var guardError *Error
	if !errors.As(err, &guardError) || guardError.Artifact != rustProvenanceArtifact || !strings.Contains(guardError.Invariant, "available specification source-release tag") {
		t.Fatalf("Check() error = %#v", err)
	}
}

func TestCheckRejectsReleasePolicyMutations(t *testing.T) {
	tests := []struct {
		name        string
		artifact    string
		old         string
		replacement string
		invariant   string
	}{
		{
			name: "manifest component scope", artifact: manifestArtifact,
			old: `"openapi/generated": "0.1.0"`, replacement: `"openapi/generated": "0.1.0", "extra": "0.1.0"`,
			invariant: "contains the specification and optional bootstrapped Rust component",
		},
		{
			name: "manifest SemVer", artifact: manifestArtifact,
			old: `"0.1.0"`, replacement: `"01.1.0"`,
			invariant: "specification version is SemVer",
		},
		{
			name: "config package scope", artifact: configArtifact,
			old: `"packages": {`, replacement: `"packages": { "extra": {},`,
			invariant: "contains only the specification and Rust packages",
		},
		{
			name: "specification component path", artifact: configArtifact,
			old: `"openapi/generated": {`, replacement: `".": {`,
			invariant: "contains only the specification and Rust packages",
		},
		{
			name: "config top-level option allowlist", artifact: configArtifact,
			old: `"bootstrap-sha":`, replacement: `"extra-files": ["VERSION"], "bootstrap-sha":`,
			invariant: "contains only supported top-level options",
		},
		{
			name: "config root package option allowlist", artifact: configArtifact,
			old: `"release-type": "simple"`, replacement: `"extra-files": ["VERSION"], "release-type": "simple"`,
			invariant: "root package contains only supported options",
		},
		{
			name: "release type spelling", artifact: configArtifact,
			old: `"release-type": "simple"`, replacement: `"releaseType": "simple"`,
			invariant: "uses kebab-case release-type",
		},
		{
			name: "release type value", artifact: configArtifact,
			old: `"release-type": "simple"`, replacement: `"release-type": "node"`,
			invariant: "root package release-type",
		},
		{
			name: "package name", artifact: configArtifact,
			old: `"package-name": "opendart-spec"`, replacement: `"package-name": "other"`,
			invariant: "root package package-name",
		},
		{
			name: "tag component", artifact: configArtifact,
			old: `"include-component-in-tag": false`, replacement: `"include-component-in-tag": true`,
			invariant: "root package include-component-in-tag",
		},
		{
			name: "tag v prefix", artifact: configArtifact,
			old: `"include-v-in-tag": true`, replacement: `"include-v-in-tag": false`,
			invariant: "root package include-v-in-tag",
		},
		{
			name: "pre-major minor policy", artifact: configArtifact,
			old: `"bump-minor-pre-major": true`, replacement: `"bump-minor-pre-major": false`,
			invariant: "root package bump-minor-pre-major",
		},
		{
			name: "pre-major patch policy", artifact: configArtifact,
			old: `"bump-patch-for-minor-pre-major": true`, replacement: `"bump-patch-for-minor-pre-major": false`,
			invariant: "root package bump-patch-for-minor-pre-major",
		},
		{
			name: "draft release", artifact: configArtifact,
			old: `"draft": true`, replacement: `"draft": false`,
			invariant: "root package draft",
		},
		{
			name: "forced tag creation", artifact: configArtifact,
			old: `"force-tag-creation": true`, replacement: `"force-tag-creation": false`,
			invariant: "root package force-tag-creation",
		},
		{
			name: "main-only release", artifact: releaseWorkflowArtifact,
			old: "      - main", replacement: "      - dev",
			invariant: "runs only for pushes to main",
		},
		{
			name: "release workflow name", artifact: releaseWorkflowArtifact,
			old: "name: Release Please", replacement: "name: Other",
			invariant: "has the expected workflow name",
		},
		{
			name: "release unknown top-level field", artifact: releaseWorkflowArtifact,
			old: "permissions: {}", replacement: "env:\n  SAFE: value\n\npermissions: {}",
			invariant: "uses only supported YAML fields",
		},
		{
			name: "no manual release", artifact: releaseWorkflowArtifact,
			old: "permissions: {}", replacement: "  workflow_dispatch:\n\npermissions: {}",
			invariant: "runs only for pushes to main",
		},
		{
			name: "no scheduled release", artifact: releaseWorkflowArtifact,
			old: "permissions: {}", replacement: "  schedule:\n    - cron: '0 0 * * *'\n\npermissions: {}",
			invariant: "runs only for pushes to main",
		},
		{
			name: "release concurrency", artifact: releaseWorkflowArtifact,
			old: "group: release-please", replacement: "group: another-release",
			invariant: "serializes release runs",
		},
		{
			name: "release root permissions", artifact: releaseWorkflowArtifact,
			old: "permissions: {}", replacement: "permissions:\n  contents: write",
			invariant: "root permissions are empty",
		},
		{
			name: "release extra job", artifact: releaseWorkflowArtifact,
			old: "jobs:\n  verify:", replacement: "jobs:\n  extra:\n    permissions:\n      contents: write\n    runs-on: ubuntu-latest\n    timeout-minutes: 5\n    steps:\n      - name: Unexpected\n        run: echo unexpected\n\n  verify:",
			invariant: "contains only the verify and release-please jobs",
		},
		{
			name: "release workflow shell bypass", artifact: releaseWorkflowArtifact,
			old: "permissions: {}", replacement: "defaults:\n  run:\n    shell: bash {0} || true\n\npermissions: {}",
			invariant: "workflow uses default run settings",
		},
		{
			name: "release workflow working-directory bypass", artifact: releaseWorkflowArtifact,
			old: "permissions: {}", replacement: "defaults:\n  run:\n    working-directory: nested\n\npermissions: {}",
			invariant: "workflow uses default run settings",
		},
		{
			name: "reusable verify", artifact: releaseWorkflowArtifact,
			old: "uses: ./.github/workflows/verify.yml", replacement: "uses: ./.github/workflows/other.yml",
			invariant: "has the reusable verify job",
		},
		{
			name: "release verify job condition bypass", artifact: releaseWorkflowArtifact,
			old: "  verify:\n    permissions:", replacement: "  verify:\n    if: always()\n    permissions:",
			invariant: "verify job uses default execution controls",
		},
		{
			name: "release job continue-on-error bypass", artifact: releaseWorkflowArtifact,
			old: "  release-please:\n    needs: verify", replacement: "  release-please:\n    continue-on-error: true\n    needs: verify",
			invariant: "release job uses default execution controls",
		},
		{
			name: "release job unknown field", artifact: releaseWorkflowArtifact,
			old: "    runs-on: blacksmith-2vcpu-ubuntu-2404", replacement: "    container: ubuntu:latest\n    runs-on: blacksmith-2vcpu-ubuntu-2404",
			invariant: "uses only supported YAML fields",
		},
		{
			name: "release runner", artifact: releaseWorkflowArtifact,
			old: "runs-on: blacksmith-2vcpu-ubuntu-2404", replacement: "runs-on: ubuntu-latest",
			invariant: "release job uses the approved runner and timeout",
		},
		{
			name: "release timeout", artifact: releaseWorkflowArtifact,
			old: "timeout-minutes: 20", replacement: "timeout-minutes: 60",
			invariant: "release job uses the approved runner and timeout",
		},
		{
			name: "release job shell bypass", artifact: releaseWorkflowArtifact,
			old: "  release-please:\n    needs: verify", replacement: "  release-please:\n    defaults:\n      run:\n        shell: bash {0} || true\n    needs: verify",
			invariant: "release job uses default run settings",
		},
		{
			name: "release job working-directory bypass", artifact: releaseWorkflowArtifact,
			old: "  release-please:\n    needs: verify", replacement: "  release-please:\n    defaults:\n      run:\n        working-directory: nested\n    needs: verify",
			invariant: "release job uses default run settings",
		},
		{
			name: "read-only release verify", artifact: releaseWorkflowArtifact,
			old: "  verify:\n    permissions:\n      contents: read", replacement: "  verify:\n    permissions:\n      contents: write",
			invariant: "verify job is read-only",
		},
		{
			name: "release waits for verify", artifact: releaseWorkflowArtifact,
			old: "needs: verify", replacement: "needs: build",
			invariant: "release waits for verification",
		},
		{
			name: "minimal release permissions", artifact: releaseWorkflowArtifact,
			old: "      pull-requests: write", replacement: "      pull-requests: write\n      actions: write",
			invariant: "release job has only required write permissions",
		},
		{
			name: "pinned release action", artifact: releaseWorkflowArtifact,
			old: "googleapis/release-please-action@45996ed1f6d02564a971a2fa1b5860e934307cf7", replacement: "googleapis/release-please-action@v5",
			invariant: "third-party step action is pinned",
		},
		{
			name: "approved release action", artifact: releaseWorkflowArtifact,
			old: "googleapis/release-please-action@45996ed1f6d02564a971a2fa1b5860e934307cf7", replacement: "actions/setup-node@820762786026740c76f36085b0efc47a31fe5020",
			invariant: "uses the approved pinned Release Please action",
		},
		{
			name: "release action cannot run a script", artifact: releaseWorkflowArtifact,
			old: "        uses: googleapis/release-please-action@45996ed1f6d02564a971a2fa1b5860e934307cf7 # v5", replacement: "        uses: googleapis/release-please-action@45996ed1f6d02564a971a2fa1b5860e934307cf7 # v5\n        run: echo unexpected",
			invariant: "uses the approved pinned Release Please action",
		},
		{
			name: "release checkout credentials", artifact: releaseWorkflowArtifact,
			old: "persist-credentials: false", replacement: "persist-credentials: true",
			invariant: "checkout disables persisted credentials",
		},
		{
			name: "package publication", artifact: releaseWorkflowArtifact,
			old: "mkdir release-assets", replacement: "npm publish\n          mkdir release-assets",
			invariant: "does not publish packages, grant registry authority, or replace assets",
		},
		{
			name: "draft detection", artifact: releaseWorkflowArtifact,
			old: "gh release view", replacement: "gh release inspect",
			invariant: "draft recovery uses the canonical script",
		},
		{
			name: "draft recovery appended command", artifact: releaseWorkflowArtifact,
			old: "          fi\n\n      - name: Run Release Please", replacement: "          fi\n          echo unexpected\n\n      - name: Run Release Please",
			invariant: "draft recovery uses the canonical script",
		},
		{
			name: "release recovery skip", artifact: releaseWorkflowArtifact,
			old: "steps.draft.outputs.recovering != 'true'", replacement: "steps.draft.outputs.recovering == 'true'",
			invariant: "Release Please is skipped during recovery",
		},
		{
			name: "release recovery condition extra clause", artifact: releaseWorkflowArtifact,
			old: "steps.draft.outputs.recovering != 'true' }}", replacement: "steps.draft.outputs.recovering != 'true' || always() }}",
			invariant: "Release Please is skipped during recovery",
		},
		{
			name: "release action continue-on-error bypass", artifact: releaseWorkflowArtifact,
			old: "      - name: Run Release Please\n        if:", replacement: "      - name: Run Release Please\n        continue-on-error: true\n        if:",
			invariant: "release step failures stop the job",
		},
		{
			name: "draft detector condition bypass", artifact: releaseWorkflowArtifact,
			old: "      - name: Detect interrupted draft release\n        id:", replacement: "      - name: Detect interrupted draft release\n        if: false\n        id:",
			invariant: "draft detector uses default execution controls",
		},
		{
			name: "release token", artifact: releaseWorkflowArtifact,
			old: "token: ${{ secrets.GITHUB_TOKEN }}", replacement: "token: ${{ secrets.OTHER_TOKEN }}",
			invariant: "Release Please uses GITHUB_TOKEN",
		},
		{
			name: "release action extra input", artifact: releaseWorkflowArtifact,
			old: "token: ${{ secrets.GITHUB_TOKEN }}", replacement: "token: ${{ secrets.GITHUB_TOKEN }}\n          fork: false",
			invariant: "Release Please uses GITHUB_TOKEN as its only input",
		},
		{
			name: "released commit checkout", artifact: releaseWorkflowArtifact,
			old: "ref: ${{ steps.release.outputs['openapi/generated--sha'] || steps.draft.outputs.sha }}", replacement: "ref: main",
			invariant: "released checkout uses the created or recovered SHA",
		},
		{
			name: "generic release output", artifact: releaseWorkflowArtifact,
			old: "steps.release.outputs['openapi/generated--release_created']", replacement: "steps.release.outputs.release_created",
			invariant: "runs only for a created or recovered release",
		},
		{
			name: "released commit uses checkout action", artifact: releaseWorkflowArtifact,
			old:         "      - name: Check out released commit\n        if: ${{ steps.release.outputs['openapi/generated--release_created'] == 'true' || steps.draft.outputs.recovering == 'true' }}\n        uses: actions/checkout@9c091bb21b7c1c1d1991bb908d89e4e9dddfe3e0",
			replacement: "      - name: Check out released commit\n        if: ${{ steps.release.outputs['openapi/generated--release_created'] == 'true' || steps.draft.outputs.recovering == 'true' }}\n        uses: actions/setup-node@820762786026740c76f36085b0efc47a31fe5020",
			invariant:   "released checkout uses the created or recovered SHA",
		},
		{
			name: "release asset condition", artifact: releaseWorkflowArtifact,
			old: "if: ${{ steps.release.outputs['openapi/generated--release_created'] == 'true' || steps.draft.outputs.recovering == 'true' }}", replacement: "if: ${{ steps.release.outputs['openapi/generated--release_created'] == 'true' }}",
			invariant: "runs only for a created or recovered release",
		},
		{
			name: "release asset condition extra clause", artifact: releaseWorkflowArtifact,
			old: "if: ${{ steps.release.outputs['openapi/generated--release_created'] == 'true' || steps.draft.outputs.recovering == 'true' }}", replacement: "if: ${{ steps.release.outputs['openapi/generated--release_created'] == 'true' || steps.draft.outputs.recovering == 'true' || always() }}",
			invariant: "runs only for a created or recovered release",
		},
		{
			name: "release asset continue-on-error bypass", artifact: releaseWorkflowArtifact,
			old: "      - name: Prepare release assets\n        if:", replacement: "      - name: Prepare release assets\n        continue-on-error: true\n        if:",
			invariant: "release step failures stop the job",
		},
		{
			name: "release extra step", artifact: releaseWorkflowArtifact,
			old: "      - name: Publish immutable release", replacement: "      - name: Modify prepared assets\n        run: echo unsafe >> release-assets/openapi.bundle.yaml\n\n      - name: Publish immutable release",
			invariant: "uses only the approved release steps in order",
		},
		{
			name: "release step unknown field", artifact: releaseWorkflowArtifact,
			old: "      - name: Prepare release assets\n        if:", replacement: "      - name: Prepare release assets\n        timeout-minutes: 1\n        if:",
			invariant: "uses only supported YAML fields",
		},
		{
			name: "release step environment", artifact: releaseWorkflowArtifact,
			old: "      - name: Prepare release assets\n        if:", replacement: "      - name: Prepare release assets\n        env:\n          SAFE: value\n        if:",
			invariant: "release steps use only approved environment variables",
		},
		{
			name: "release step shell bypass", artifact: releaseWorkflowArtifact,
			old: "      - name: Prepare release assets\n        if:", replacement: "      - name: Prepare release assets\n        shell: bash {0} || true\n        if:",
			invariant: "release steps use default run settings",
		},
		{
			name: "release step working-directory bypass", artifact: releaseWorkflowArtifact,
			old: "      - name: Prepare release assets\n        if:", replacement: "      - name: Prepare release assets\n        working-directory: nested\n        if:",
			invariant: "release steps use default run settings",
		},
		{
			name: "bundle checksum", artifact: releaseWorkflowArtifact,
			old: "sha256sum openapi.bundle.yaml > openapi.bundle.yaml.sha256", replacement: "sha1sum openapi.bundle.yaml > openapi.bundle.yaml.sha256",
			invariant: "prepares the versioned bundle and SHA-256 checksum",
		},
		{
			name: "bundle modified after checksum", artifact: releaseWorkflowArtifact,
			old: "sha256sum openapi.bundle.yaml > openapi.bundle.yaml.sha256", replacement: "sha256sum openapi.bundle.yaml > openapi.bundle.yaml.sha256\n          echo unsafe >> openapi.bundle.yaml",
			invariant: "prepares the versioned bundle and SHA-256 checksum",
		},
		{
			name: "only versioned assets", artifact: releaseWorkflowArtifact,
			old: "release-assets/openapi.bundle.yaml.sha256", replacement: "release-assets/CHANGELOG.md",
			invariant: "uploads only the bundle and checksum",
		},
		{
			name: "recovery compares assets", artifact: releaseWorkflowArtifact,
			old: "cmp -s", replacement: "diff -q",
			invariant: "asset upload preserves immutable recovery semantics",
		},
		{
			name: "upload script appended command", artifact: releaseWorkflowArtifact,
			old: "          done", replacement: "          done\n          echo unexpected",
			invariant: "uploads only the bundle and checksum",
		},
		{
			name: "publish immutable release", artifact: releaseWorkflowArtifact,
			old: "--draft=false --latest", replacement: "--draft=false",
			invariant: "publishes the draft only after assets are verified",
		},
		{
			name: "publish failure bypass", artifact: releaseWorkflowArtifact,
			old: "--draft=false --latest", replacement: "--draft=false --latest || true",
			invariant: "publishes the draft only after assets are verified",
		},
		{
			name: "verify workflow permissions", artifact: verifyWorkflowArtifact,
			old: "permissions:\n  contents: read", replacement: "permissions:\n  contents: write",
			invariant: "permissions are read-only",
		},
		{
			name: "verify workflow name", artifact: verifyWorkflowArtifact,
			old: "name: Verify", replacement: "name: Other",
			invariant: "has the expected workflow name",
		},
		{
			name: "verify extra job", artifact: verifyWorkflowArtifact,
			old: "jobs:\n  verify:", replacement: "jobs:\n  extra:\n    runs-on: ubuntu-latest\n    timeout-minutes: 5\n    steps:\n      - name: Unexpected\n        run: echo unexpected\n\n  verify:",
			invariant: "contains only the verify job",
		},
		{
			name: "verify workflow shell bypass", artifact: verifyWorkflowArtifact,
			old: "permissions:\n  contents: read", replacement: "defaults:\n  run:\n    shell: bash {0} || true\n\npermissions:\n  contents: read",
			invariant: "workflow uses default run settings",
		},
		{
			name: "verify workflow working-directory bypass", artifact: verifyWorkflowArtifact,
			old: "permissions:\n  contents: read", replacement: "defaults:\n  run:\n    working-directory: nested\n\npermissions:\n  contents: read",
			invariant: "workflow uses default run settings",
		},
		{
			name: "verify triggers", artifact: verifyWorkflowArtifact,
			old: "  workflow_dispatch:", replacement: "  schedule:",
			invariant: "supports workflow_dispatch",
		},
		{
			name: "canonical verify command", artifact: verifyWorkflowArtifact,
			old: "go run ./cmd/opendart-tool verify --repository-root .", replacement: "go run ./cmd/opendart-tool lint --root openapi/openapi.yaml",
			invariant: "uses only the approved verification steps",
		},
		{
			name: "Go vet command", artifact: verifyWorkflowArtifact,
			old: "go vet ./...", replacement: "go vet ./cmd/...",
			invariant: "uses only the approved verification steps",
		},
		{
			name: "race-enabled Go tests", artifact: verifyWorkflowArtifact,
			old: "go test -race ./...", replacement: "go test ./...",
			invariant: "uses only the approved verification steps",
		},
		{
			name: "verify job condition bypass", artifact: verifyWorkflowArtifact,
			old: "  verify:\n    runs-on:", replacement: "  verify:\n    if: always()\n    runs-on:",
			invariant: "verify job uses default execution controls",
		},
		{
			name: "verify job environment", artifact: verifyWorkflowArtifact,
			old: "  verify:\n    runs-on:", replacement: "  verify:\n    env:\n      SAFE: value\n    runs-on:",
			invariant: "uses only supported YAML fields",
		},
		{
			name: "verify runner", artifact: verifyWorkflowArtifact,
			old: "runs-on: ubuntu-latest", replacement: "runs-on: macos-latest",
			invariant: "verify job uses the approved runner and timeout",
		},
		{
			name: "verify timeout", artifact: verifyWorkflowArtifact,
			old: "timeout-minutes: 30", replacement: "timeout-minutes: 60",
			invariant: "verify job uses the approved runner and timeout",
		},
		{
			name: "verify job shell bypass", artifact: verifyWorkflowArtifact,
			old: "  verify:\n    runs-on:", replacement: "  verify:\n    defaults:\n      run:\n        shell: bash {0} || true\n    runs-on:",
			invariant: "verify job uses default run settings",
		},
		{
			name: "verify job working-directory bypass", artifact: verifyWorkflowArtifact,
			old: "  verify:\n    runs-on:", replacement: "  verify:\n    defaults:\n      run:\n        working-directory: nested\n    runs-on:",
			invariant: "verify job uses default run settings",
		},
		{
			name: "canonical verify step condition bypass", artifact: verifyWorkflowArtifact,
			old: "      - name: Verify repository\n        run:", replacement: "      - name: Verify repository\n        if: always()\n        run:",
			invariant: "verification steps use default execution controls",
		},
		{
			name: "verify step continue-on-error bypass", artifact: verifyWorkflowArtifact,
			old: "      - name: Verify repository\n        run:", replacement: "      - name: Verify repository\n        continue-on-error: true\n        run:",
			invariant: "verification steps use default execution controls",
		},
		{
			name: "verify step shell bypass", artifact: verifyWorkflowArtifact,
			old: "      - name: Verify repository\n        run:", replacement: "      - name: Verify repository\n        shell: bash {0} || true\n        run:",
			invariant: "verification steps use default run settings",
		},
		{
			name: "verify step working-directory bypass", artifact: verifyWorkflowArtifact,
			old: "      - name: Verify repository\n        run:", replacement: "      - name: Verify repository\n        working-directory: nested\n        run:",
			invariant: "verification steps use default run settings",
		},
		{
			name: "pinned verify action", artifact: verifyWorkflowArtifact,
			old: "actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e", replacement: "actions/setup-go@v7",
			invariant: "third-party step action is pinned",
		},
		{
			name: "verify checkout credentials", artifact: verifyWorkflowArtifact,
			old: "persist-credentials: false", replacement: "persist-credentials: true",
			invariant: "uses only the approved verification actions",
		},
		{
			name: "verify checkout ref", artifact: verifyWorkflowArtifact,
			old: "persist-credentials: false", replacement: "persist-credentials: false\n          ref: main",
			invariant: "uses only the approved verification actions",
		},
		{
			name: "verify checkout repository", artifact: verifyWorkflowArtifact,
			old: "persist-credentials: false", replacement: "persist-credentials: false\n          repository: cpaikr/other",
			invariant: "uses only the approved verification actions",
		},
		{
			name: "verify checkout extra input", artifact: verifyWorkflowArtifact,
			old: "persist-credentials: false", replacement: "persist-credentials: false\n          show-progress: false",
			invariant: "uses only the approved verification actions",
		},
		{
			name: "verify step environment", artifact: verifyWorkflowArtifact,
			old: "      - name: Vet Go\n        run:", replacement: "      - name: Vet Go\n        env:\n          SAFE: value\n        run:",
			invariant: "verification steps do not override the environment",
		},
		{
			name: "verify secrets", artifact: verifyWorkflowArtifact,
			old: "run: go run ./cmd/opendart-tool verify --repository-root .", replacement: "env:\n          TOKEN: ${{ secrets.GITHUB_TOKEN }}\n        run: go run ./cmd/opendart-tool verify --repository-root .",
			invariant: "excludes GitHub secrets",
		},
		{
			name: "verify secrets bracket access", artifact: verifyWorkflowArtifact,
			old: "run: go run ./cmd/opendart-tool verify --repository-root .", replacement: "env:\n          TOKEN: ${{ secrets['GITHUB_TOKEN'] }}\n        run: go run ./cmd/opendart-tool verify --repository-root .",
			invariant: "excludes GitHub secrets",
		},
		{
			name: "verify github token property access", artifact: verifyWorkflowArtifact,
			old: "run: go run ./cmd/opendart-tool verify --repository-root .", replacement: "env:\n          TOKEN: ${{ github.token }}\n        run: go run ./cmd/opendart-tool verify --repository-root .",
			invariant: "excludes GitHub token",
		},
		{
			name: "verify github token index access", artifact: verifyWorkflowArtifact,
			old: "run: go run ./cmd/opendart-tool verify --repository-root .", replacement: "env:\n          TOKEN: ${{ github[\"token\"] }}\n        run: go run ./cmd/opendart-tool verify --repository-root .",
			invariant: "excludes GitHub token",
		},
		{
			name: "verify API key", artifact: verifyWorkflowArtifact,
			old: "run: go run ./cmd/opendart-tool verify --repository-root .", replacement: "env:\n          OPENDART_API_KEY: unsafe\n        run: go run ./cmd/opendart-tool verify --repository-root .",
			invariant: "excludes OpenDART API key",
		},
		{
			name: "verify sync", artifact: verifyWorkflowArtifact,
			old: "run: go run ./cmd/opendart-tool verify --repository-root .", replacement: "run: go run ./cmd/opendart-tool verify --repository-root . && go run ./cmd/opendart-tool sync",
			invariant: "excludes guide synchronization",
		},
		{
			name: "verify Node setup", artifact: verifyWorkflowArtifact,
			old: "      - name: Set up Go", replacement: "      - name: Set up Node.js\n        uses: actions/setup-node@820762786026740c76f36085b0efc47a31fe5020\n\n      - name: Set up Go",
			invariant: "excludes JavaScript or Node package tooling",
		},
		{
			name: "verify npm command", artifact: verifyWorkflowArtifact,
			old: "run: go run ./cmd/opendart-tool verify --repository-root .", replacement: "run: npm run verify:opendart",
			invariant: "excludes JavaScript or Node package tooling",
		},
		{
			name: "verify nodejs command", artifact: verifyWorkflowArtifact,
			old: "run: go run ./cmd/opendart-tool verify --repository-root .", replacement: "run: nodejs scripts/verify.js",
			invariant: "excludes JavaScript or Node package tooling",
		},
		{
			name: "verify alternate package manager", artifact: verifyWorkflowArtifact,
			old: "run: go run ./cmd/opendart-tool verify --repository-root .", replacement: "run: yarn verify",
			invariant: "excludes JavaScript or Node package tooling",
		},
		{
			name: "verify JavaScript script", artifact: verifyWorkflowArtifact,
			old: "      - name: Set up Go", replacement: "      - name: Run repository script\n        run: ./scripts/check.mjs\n\n      - name: Set up Go",
			invariant: "uses only the approved verification steps",
		},
		{
			name: "verify local action", artifact: verifyWorkflowArtifact,
			old: "      - name: Set up Go", replacement: "      - name: Run local action\n        uses: ./actions/check\n\n      - name: Set up Go",
			invariant: "uses only the approved verification steps",
		},
		{
			name: "live workflow schedule", artifact: liveWorkflowArtifact,
			old: "  workflow_dispatch:", replacement: "  schedule:\n    - cron: '0 0 * * *'",
			invariant: "is manual only",
		},
		{
			name: "live non-main ref", artifact: liveWorkflowArtifact,
			old: "github.ref == 'refs/heads/main'", replacement: "github.ref != ''",
			invariant: "runs only trusted main code",
		},
		{
			name: "live unprotected environment", artifact: liveWorkflowArtifact,
			old: "environment: opendart-live-conformance", replacement: "environment: other",
			invariant: "uses only the protected live environment",
		},
		{
			name: "live issue permission", artifact: liveWorkflowArtifact,
			old: "      contents: read", replacement: "      contents: read\n      issues: write",
			invariant: "producer has read-only repository permission",
		},
		{
			name: "live secret at preflight", artifact: liveWorkflowArtifact,
			old: "      - name: Recheck offline gates\n        run:", replacement: "      - name: Recheck offline gates\n        env:\n          OPENDART_API_KEY: ${{ secrets.OPENDART_API_KEY }}\n        run:",
			invariant: "API key is absent outside the request boundary",
		},
		{
			name: "live arbitrary artifact", artifact: liveWorkflowArtifact,
			old: "path: live-conformance-report.json", replacement: "path: .",
			invariant: "uploads only the bounded sanitized report",
		},
		{
			name: "live artifact is not attempt scoped", artifact: liveWorkflowArtifact,
			old: "name: live-conformance-report-${{ github.run_attempt }}", replacement: "name: live-conformance-report",
			invariant: "uploads only the bounded sanitized report",
		},
		{
			name: "live unpinned upload", artifact: liveWorkflowArtifact,
			old: uploadArtifactAction, replacement: "actions/upload-artifact@v7",
			invariant: "uploads only the bounded sanitized report",
		},
		{
			name: "notifier manual trigger", artifact: notifyWorkflowArtifact,
			old: "  workflow_run:", replacement: "  workflow_dispatch:",
			invariant: "runs only after the live producer completes",
		},
		{
			name: "notifier accepts branch", artifact: notifyWorkflowArtifact,
			old: "github.event.workflow_run.head_branch == github.event.repository.default_branch", replacement: "github.event.workflow_run.head_branch != ''",
			invariant: "accepts only manual trusted default-branch producer runs",
		},
		{
			name: "notifier artifact is not attempt scoped", artifact: notifyWorkflowArtifact,
			old: "name: live-conformance-report-${{ github.event.workflow_run.run_attempt }}", replacement: "name: live-conformance-report",
			invariant: "downloads only the producer report with fixed-failure fallback",
		},
		{
			name: "notifier protected environment", artifact: notifyWorkflowArtifact,
			old: "    permissions:\n      actions: read", replacement: "    environment: opendart-live-conformance\n    permissions:\n      actions: read",
			invariant: "isolates minimal issue authority",
		},
		{
			name: "notifier excessive permission", artifact: notifyWorkflowArtifact,
			old: "      issues: write", replacement: "      issues: write\n      pull-requests: write",
			invariant: "isolates minimal issue authority",
		},
		{
			name: "notifier download failure bypass", artifact: notifyWorkflowArtifact,
			old: "continue-on-error: true", replacement: "continue-on-error: false",
			invariant: "downloads only the producer report with fixed-failure fallback",
		},
		{
			name: "notifier untrusted checkout", artifact: notifyWorkflowArtifact,
			old: "ref: ${{ github.event.workflow_run.head_sha }}", replacement: "ref: main",
			invariant: "checks out the exact trusted producer revision",
		},
		{
			name: "notifier credential access", artifact: notifyWorkflowArtifact,
			old: "GITHUB_TOKEN: ${{ github.token }}", replacement: "GITHUB_TOKEN: ${{ github.token }}\n          OPENDART_API_KEY: ${{ secrets.OPENDART_API_KEY }}",
			invariant: "invokes only the isolated notifier with trusted metadata",
		},
		{
			name: "notifier arbitrary producer error", artifact: notifyWorkflowArtifact,
			old: "NOTIFY_ARTIFACT_OUTCOME: ${{ steps.report.outcome }}", replacement: "NOTIFY_ARTIFACT_OUTCOME: ${{ steps.report.outputs.error }}",
			invariant: "invokes only the isolated notifier with trusted metadata",
		},
		{
			name: "drift workflow schedule", artifact: driftWorkflowArtifact,
			old: "  workflow_dispatch:", replacement: "  schedule:\n    - cron: '0 0 * * *'",
			invariant: "is manual only",
		},
		{
			name: "drift non-main ref", artifact: driftWorkflowArtifact,
			old: "github.ref == 'refs/heads/main'", replacement: "github.ref != ''",
			invariant: "runs only trusted main code",
		},
		{
			name: "drift non-canonical repository", artifact: driftWorkflowArtifact,
			old: "github.repository == 'cpaikr/opendart'", replacement: "github.repository != ''",
			invariant: "runs only trusted main code",
		},
		{
			name: "drift issue permission", artifact: driftWorkflowArtifact,
			old: "      contents: read", replacement: "      contents: read\n      issues: write",
			invariant: "uses only read-only repository authority",
		},
		{
			name: "drift credential environment", artifact: driftWorkflowArtifact,
			old: "      - name: Compare the public guide\n        run:", replacement: "      - name: Compare the public guide\n        env:\n          OPENDART_API_KEY: ${{ secrets.OPENDART_API_KEY }}\n        run:",
			invariant: "producer steps use default credential-free execution settings",
		},
		{
			name: "drift checkout credentials", artifact: driftWorkflowArtifact,
			old: "persist-credentials: false", replacement: "persist-credentials: true",
			invariant: "checks out the trusted dispatched revision without credentials",
		},
		{
			name: "drift altered command", artifact: driftWorkflowArtifact,
			old: driftRunScript, replacement: "go run ./cmd/opendart-tool sync > guide-drift-report.json",
			invariant: "runs only the canonical credential-free drift command",
		},
		{
			name: "drift arbitrary artifact", artifact: driftWorkflowArtifact,
			old: "path: guide-drift-report.json", replacement: "path: .",
			invariant: "uploads only the bounded sanitized report",
		},
		{
			name: "drift artifact is not attempt scoped", artifact: driftWorkflowArtifact,
			old: "name: guide-drift-report-${{ github.run_attempt }}", replacement: "name: guide-drift-report",
			invariant: "uploads only the bounded sanitized report",
		},
		{
			name: "drift unpinned upload", artifact: driftWorkflowArtifact,
			old: uploadArtifactAction, replacement: "actions/upload-artifact@v7",
			invariant: "uploads only the bounded sanitized report",
		},
		{
			name: "drift upload condition", artifact: driftWorkflowArtifact,
			old: "if: ${{ always() }}", replacement: "if: ${{ success() }}",
			invariant: "uploads only the bounded sanitized report",
		},
		{
			name: "drift notifier manual trigger", artifact: driftNotifyArtifact,
			old: "  workflow_run:", replacement: "  workflow_dispatch:",
			invariant: "runs only after the public guide drift producer completes",
		},
		{
			name: "drift notifier wrong producer", artifact: driftNotifyArtifact,
			old: "      - Public Guide Drift", replacement: "      - Other Workflow",
			invariant: "runs only after the public guide drift producer completes",
		},
		{
			name: "drift notifier accepts branch", artifact: driftNotifyArtifact,
			old: "github.event.workflow_run.head_branch == github.event.repository.default_branch", replacement: "github.event.workflow_run.head_branch != ''",
			invariant: "accepts only manual trusted default-branch producer runs",
		},
		{
			name: "drift notifier accepts non-canonical repository", artifact: driftNotifyArtifact,
			old: "github.repository == 'cpaikr/opendart'", replacement: "github.repository != ''",
			invariant: "accepts only manual trusted default-branch producer runs",
		},
		{
			name: "drift notifier artifact is not attempt scoped", artifact: driftNotifyArtifact,
			old: "name: guide-drift-report-${{ github.event.workflow_run.run_attempt }}", replacement: "name: guide-drift-report",
			invariant: "downloads only the producer report with fixed-failure fallback",
		},
		{
			name: "drift notifier excessive permission", artifact: driftNotifyArtifact,
			old: "      issues: write", replacement: "      issues: write\n      pull-requests: write",
			invariant: "isolates minimal issue authority",
		},
		{
			name: "drift notifier protected environment", artifact: driftNotifyArtifact,
			old: "    permissions:\n      actions: read", replacement: "    environment: opendart-live-conformance\n    permissions:\n      actions: read",
			invariant: "isolates minimal issue authority",
		},
		{
			name: "drift notifier download failure bypass", artifact: driftNotifyArtifact,
			old: "continue-on-error: true", replacement: "continue-on-error: false",
			invariant: "downloads only the producer report with fixed-failure fallback",
		},
		{
			name: "drift notifier untrusted checkout", artifact: driftNotifyArtifact,
			old: "ref: ${{ github.event.workflow_run.head_sha }}", replacement: "ref: main",
			invariant: "checks out the exact trusted producer revision",
		},
		{
			name: "drift notifier credential access", artifact: driftNotifyArtifact,
			old: "GITHUB_TOKEN: ${{ github.token }}", replacement: "GITHUB_TOKEN: ${{ github.token }}\n          OPENDART_API_KEY: ${{ secrets.OPENDART_API_KEY }}",
			invariant: "invokes only the isolated notifier with trusted metadata",
		},
		{
			name: "drift notifier arbitrary producer error", artifact: driftNotifyArtifact,
			old: "NOTIFY_ARTIFACT_OUTCOME: ${{ steps.report.outcome }}", replacement: "NOTIFY_ARTIFACT_OUTCOME: ${{ steps.report.outputs.error }}",
			invariant: "invokes only the isolated notifier with trusted metadata",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := copyReleaseArtifacts(t)
			path := filepath.Join(root, filepath.FromSlash(test.artifact))
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			updated := strings.Replace(string(source), test.old, test.replacement, 1)
			if updated == string(source) {
				t.Fatalf("mutation source %q not found in %s", test.old, test.artifact)
			}
			if err := os.WriteFile(path, []byte(updated), 0o600); err != nil {
				t.Fatal(err)
			}

			err = Check(root)
			if err == nil {
				t.Fatal("Check() error = nil")
			}
			var guardError *Error
			if !errors.As(err, &guardError) {
				t.Fatalf("Check() error type = %T, want *Error", err)
			}
			if guardError.Artifact != test.artifact {
				t.Fatalf("Check() artifact = %q, want %q; error = %v", guardError.Artifact, test.artifact, err)
			}
			if !strings.Contains(guardError.Invariant, test.invariant) {
				t.Fatalf("Check() invariant = %q, want substring %q", guardError.Invariant, test.invariant)
			}
		})
	}
}

func TestCheckReturnsContextForMissingArtifact(t *testing.T) {
	err := Check(t.TempDir())
	var guardError *Error
	if !errors.As(err, &guardError) {
		t.Fatalf("Check() error = %v, want *Error", err)
	}
	if guardError.Artifact != configArtifact || guardError.Invariant != "can be read" || guardError.Cause == nil {
		t.Fatalf("Check() error = %#v", guardError)
	}
}

func TestReleaseWorkflowOrderingFailsClosed(t *testing.T) {
	releaseSource, err := os.ReadFile(filepath.Join(repositoryRoot(t), filepath.FromSlash(releaseWorkflowArtifact)))
	if err != nil {
		t.Fatal(err)
	}
	var baselineRelease workflow
	if err := yaml.Unmarshal(releaseSource, &baselineRelease); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name  string
		first string
		last  string
	}{
		{name: "draft after release", first: "Detect interrupted draft release", last: "Run Release Please"},
		{name: "upload before prepare", first: "Prepare release assets", last: "Upload release assets"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			release := baselineRelease
			job := release.Jobs["release-please"]
			job.Steps = append([]workflowStep(nil), job.Steps...)
			first, _, err := stepByName(job.Steps, test.first)
			if err != nil {
				t.Fatal(err)
			}
			last, _, err := stepByName(job.Steps, test.last)
			if err != nil {
				t.Fatal(err)
			}
			job.Steps[first], job.Steps[last] = job.Steps[last], job.Steps[first]
			release.Jobs = cloneJobs(baselineRelease.Jobs)
			release.Jobs["release-please"] = job

			if err := checkReleaseWorkflow(release, string(releaseSource)); err == nil {
				t.Fatal("checkReleaseWorkflow() error = nil")
			}
		})
	}
}

func copyReleaseArtifacts(t *testing.T) string {
	t.Helper()
	sourceRoot := repositoryRoot(t)
	targetRoot := t.TempDir()
	for _, artifact := range []string{
		configArtifact,
		manifestArtifact,
		rustCargoArtifact,
		rustLockArtifact,
		rustProvenanceArtifact,
		rustPackageListArtifact,
		canonicalBundleArtifact,
		releaseWorkflowArtifact,
		verifyWorkflowArtifact,
		liveWorkflowArtifact,
		notifyWorkflowArtifact,
		driftWorkflowArtifact,
		driftNotifyArtifact,
	} {
		source, err := os.ReadFile(filepath.Join(sourceRoot, filepath.FromSlash(artifact)))
		if err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(targetRoot, filepath.FromSlash(artifact))
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, source, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for _, sourcePath := range canonicalSpecificationSources {
		copyPath(t, sourceRoot, targetRoot, sourcePath)
	}
	gitDirectory, err := exec.Command("git", "-C", sourceRoot, "rev-parse", "--absolute-git-dir").Output()
	if err != nil {
		t.Fatal(err)
	}
	gitFile := []byte("gitdir: " + strings.TrimSpace(string(gitDirectory)) + "\n")
	if err := os.WriteFile(filepath.Join(targetRoot, ".git"), gitFile, 0o600); err != nil {
		t.Fatal(err)
	}
	return targetRoot
}

func copyPath(t *testing.T, sourceRoot, targetRoot, relativePath string) {
	t.Helper()
	sourcePath := filepath.Join(sourceRoot, filepath.FromSlash(relativePath))
	info, err := os.Stat(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDir() {
		copyFile(t, sourceRoot, targetRoot, relativePath)
		return
	}
	if err := filepath.WalkDir(sourcePath, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(sourceRoot, path)
		if err != nil {
			return err
		}
		copyFile(t, sourceRoot, targetRoot, filepath.ToSlash(relative))
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func copyFile(t *testing.T, sourceRoot, targetRoot, relativePath string) {
	t.Helper()
	source, err := os.ReadFile(filepath.Join(sourceRoot, filepath.FromSlash(relativePath)))
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(targetRoot, filepath.FromSlash(relativePath))
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, source, 0o600); err != nil {
		t.Fatal(err)
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func cloneJobs(source map[string]workflowJob) map[string]workflowJob {
	result := make(map[string]workflowJob, len(source))
	for name, job := range source {
		result[name] = job
	}
	return result
}
