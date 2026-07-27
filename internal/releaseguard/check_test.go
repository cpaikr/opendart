package releaseguard

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"go.yaml.in/yaml/v4"
)

func TestCheckAcceptsRepositoryReleasePolicy(t *testing.T) {
	if err := Check(repositoryRoot(t)); err != nil {
		t.Fatalf("Check() error = %v", err)
	}
}

func TestVerifyAggregateScriptFailsClosed(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the GitHub aggregate runs with a POSIX shell on ubuntu-latest")
	}
	tests := []struct {
		name    string
		results map[string]string
		wantOK  bool
	}{
		{
			name: "success",
			results: map[string]string{
				"GO_RESULT": "success", "RUST_RESULT": "success",
				"MACOS_RESULT": "success", "WINDOWS_RESULT": "success",
			},
			wantOK: true,
		},
		{
			name: "failure",
			results: map[string]string{
				"GO_RESULT": "failure", "RUST_RESULT": "success",
				"MACOS_RESULT": "success", "WINDOWS_RESULT": "success",
			},
		},
		{
			name: "cancelled",
			results: map[string]string{
				"GO_RESULT": "success", "RUST_RESULT": "cancelled",
				"MACOS_RESULT": "success", "WINDOWS_RESULT": "success",
			},
		},
		{
			name: "unexpected skip",
			results: map[string]string{
				"GO_RESULT": "success", "RUST_RESULT": "success",
				"MACOS_RESULT": "skipped", "WINDOWS_RESULT": "success",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()

			command := exec.CommandContext(ctx, "sh", "-c", verifyAggregateScript)
			command.Env = os.Environ()
			for name, value := range test.results {
				command.Env = append(command.Env, name+"="+value)
			}
			err := command.Run()
			if ctx.Err() != nil {
				t.Fatalf("aggregate script did not complete within timeout: %v", ctx.Err())
			}
			if (err == nil) != test.wantOK {
				t.Fatalf("aggregate result error = %v, want success %t", err, test.wantOK)
			}
		})
	}
}

func TestCheckRejectsVerificationPortfolioMutations(t *testing.T) {
	fixture := newReleaseArtifactFixture(t)
	tests := []struct {
		name        string
		artifact    string
		old         string
		replacement string
		invariant   string
	}{
		{
			name: "normal Go tests", artifact: verificationScriptArtifact,
			old: "  go test -vet=off ./...\n  phase \"targeted Go race tests\"", replacement: "  phase \"targeted Go race tests\"",
			invariant: "matches the reviewed fast, pull-request, and exhaustive tier contract",
		},
		{
			name: "normal Go tests repeat vet", artifact: verificationScriptArtifact,
			old: "go test -vet=off ./...", replacement: "go test ./...",
			invariant: "matches the reviewed fast, pull-request, and exhaustive tier contract",
		},
		{
			name: "targeted race tests repeat vet", artifact: verificationScriptArtifact,
			old: "go test -race -vet=off \\", replacement: "go test -race \\",
			invariant: "matches the reviewed fast, pull-request, and exhaustive tier contract",
		},
		{
			name: "targeted race package", artifact: verificationScriptArtifact,
			old: "    ./internal/guide \\\n", replacement: "",
			invariant: "matches the reviewed fast, pull-request, and exhaustive tier contract",
		},
		{
			name: "Rust compatibility", artifact: verificationScriptArtifact,
			old: "  verify_reqwest_compatibility\n", replacement: "",
			invariant: "matches the reviewed fast, pull-request, and exhaustive tier contract",
		},
		{
			name: "WebAssembly contract", artifact: verificationScriptArtifact,
			old: "  verify_wasm\n", replacement: "",
			invariant: "matches the reviewed fast, pull-request, and exhaustive tier contract",
		},
		{
			name: "WebAssembly futures family", artifact: verificationScriptArtifact,
			old: "futures(-[^ ]+)?", replacement: "futures-(core|io|sink|task|util)",
			invariant: "matches the reviewed fast, pull-request, and exhaustive tier contract",
		},
		{
			name: "pre-push composition", artifact: verificationScriptArtifact,
			old: "verify_pre_push() {\n  verify_go\n  verify_rust\n}", replacement: "verify_pre_push() {\n  verify_go\n}",
			invariant: "matches the reviewed fast, pull-request, and exhaustive tier contract",
		},
		{
			name: "full race command", artifact: verificationScriptArtifact,
			old: "  go test -race ./...\n}", replacement: "  go test ./...\n}",
			invariant: "matches the reviewed fast, pull-request, and exhaustive tier contract",
		},
		{
			name: "exhaustive composition", artifact: verificationScriptArtifact,
			old: "verify_exhaustive() {\n  verify_pre_push\n  verify_full_race\n}", replacement: "verify_exhaustive() {\n  verify_pre_push\n}",
			invariant: "matches the reviewed fast, pull-request, and exhaustive tier contract",
		},
		{
			name: "Go workflow entrypoint", artifact: verifyWorkflowArtifact,
			old: "run: ./scripts/verify go", replacement: "run: go test ./...",
			invariant: "uses only the approved verification steps",
		},
		{
			name: "Rust workflow entrypoint", artifact: verifyWorkflowArtifact,
			old: "run: ./scripts/verify rust", replacement: "run: cargo test",
			invariant: "uses only the approved verification steps",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := fixture.copy(t)
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

func TestCheckRejectsNonExecutableVerificationScript(t *testing.T) {
	fixture := newReleaseArtifactFixture(t)
	root := fixture.copy(t)
	path := filepath.Join(root, filepath.FromSlash(verificationScriptArtifact))
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	err := Check(root)
	var guardError *Error
	if !errors.As(err, &guardError) || guardError.Artifact != verificationScriptArtifact || guardError.Invariant != "is executable" {
		t.Fatalf("Check() error = %#v", err)
	}
}

func TestCheckRejectsUnauditedGoOwnership(t *testing.T) {
	fixture := newReleaseArtifactFixture(t)
	tests := []struct {
		name      string
		source    string
		invariant string
	}{
		{
			name:      "goroutine",
			source:    "package newownership\n\nfunc start(work func()) { go work() }\n",
			invariant: "AST-discovered direct concurrency ownership",
		},
		{
			name:      "test server lifecycle",
			source:    "package newownership\n\nimport \"net/http/httptest\"\n\nfunc start() { _ = httptest.NewUnstartedServer(nil) }\n",
			invariant: "AST-discovered direct concurrency ownership",
		},
		{
			name:      "cancellation",
			source:    "package newownership\n\nimport \"context\"\n\nfunc start() { _, _ = context.WithCancel(context.Background()) }\n",
			invariant: "cancellation ownership",
		},
		{
			name:      "package state",
			source:    "package newownership\n\nvar state = 1\n",
			invariant: "package-level state ownership",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := fixture.copy(t)
			path := filepath.Join(root, "internal", "newownership", "worker.go")
			if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(test.source), 0o600); err != nil {
				t.Fatal(err)
			}
			err := Check(root)
			var guardError *Error
			if !errors.As(err, &guardError) ||
				guardError.Artifact != verificationScriptArtifact ||
				!strings.Contains(guardError.Invariant, test.invariant) {
				t.Fatalf("Check() error = %#v", err)
			}
		})
	}
}

func TestCheckRejectsFullRaceWorkflowMutations(t *testing.T) {
	fixture := newReleaseArtifactFixture(t)
	tests := []struct {
		name        string
		old         string
		replacement string
		invariant   string
	}{
		{name: "schedule", old: `cron: "17 3 * * 1"`, replacement: `cron: "17 3 * * 2"`, invariant: "weekly schedule and manual dispatch only"},
		{name: "manual trigger", old: "  workflow_dispatch:", replacement: "  pull_request:", invariant: "weekly schedule and manual dispatch only"},
		{name: "permissions", old: "contents: read", replacement: "contents: write", invariant: "permissions are read-only"},
		{name: "cancellation", old: "cancel-in-progress: false", replacement: "cancel-in-progress: true", invariant: "non-cancelling"},
		{name: "runner", old: "runs-on: ubuntu-latest", replacement: "runs-on: macos-latest", invariant: "approved runner and timeout"},
		{name: "condition bypass", old: "  full-race:\n    runs-on:", replacement: "  full-race:\n    if: always()\n    runs-on:", invariant: "default execution controls"},
		{name: "script entrypoint", old: "run: ./scripts/verify full-race", replacement: "run: go test ./...", invariant: "approved full-race steps"},
		{name: "unpinned setup", old: "actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e", replacement: "actions/setup-go@v7", invariant: "third-party step action is pinned"},
		{name: "checkout credentials", old: "persist-credentials: false", replacement: "persist-credentials: true", invariant: "approved full-race actions"},
		{name: "trigger SHA binding", old: `test "${GITHUB_SHA}" = "${EXPECTED_SHA}"`, replacement: `test -n "${GITHUB_SHA}"`, invariant: "approved full-race steps"},
		{name: "credential access", old: "run: ./scripts/verify full-race", replacement: "run: echo ${{ secrets.GITHUB_TOKEN }}", invariant: "excludes GitHub secrets"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := fixture.copy(t)
			path := filepath.Join(root, filepath.FromSlash(fullRaceWorkflowArtifact))
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			updated := strings.Replace(string(source), test.old, test.replacement, 1)
			if updated == string(source) {
				t.Fatalf("mutation source %q not found", test.old)
			}
			if err := os.WriteFile(path, []byte(updated), 0o600); err != nil {
				t.Fatal(err)
			}
			err = Check(root)
			var guardError *Error
			if !errors.As(err, &guardError) ||
				guardError.Artifact != fullRaceWorkflowArtifact ||
				!strings.Contains(guardError.Invariant, test.invariant) {
				t.Fatalf("Check() error = %#v, want invariant containing %q", err, test.invariant)
			}
		})
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

func TestCheckRejectsUnpublishedRustReleaseManifestEntries(t *testing.T) {
	fixture := newReleaseArtifactFixture(t)
	for _, packagePath := range []string{rustPackagePath, rustCLIPackagePath} {
		t.Run(packagePath, func(t *testing.T) {
			root := fixture.copy(t)
			path := filepath.Join(root, manifestArtifact)
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			updated := strings.Replace(
				string(source),
				`"openapi/generated": "0.1.0"`,
				`"openapi/generated": "0.1.0", "`+packagePath+`": "0.1.0"`,
				1,
			)
			if err := os.WriteFile(path, []byte(updated), 0o600); err != nil {
				t.Fatal(err)
			}
			err = Check(root)
			var guardError *Error
			expectedInvariant := "published SDK beta version"
			if packagePath == rustCLIPackagePath {
				expectedInvariant = "optionally published SDK component"
			}
			if !errors.As(err, &guardError) || guardError.Artifact != manifestArtifact || !strings.Contains(guardError.Invariant, expectedInvariant) {
				t.Fatalf("Check() error = %#v", err)
			}
		})
	}
}

func TestCheckRejectsRustReleaseOwnershipMutations(t *testing.T) {
	fixture := newReleaseArtifactFixture(t)
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
		{name: "Rust lock path", artifact: configArtifact, old: `"path": "/sdk/rust/Cargo.lock"`, replacement: `"path": "Cargo.lock"`, invariant: "updates the workspace lock and CLI SDK pin"},
		{name: "Rust lock selector", artifact: configArtifact, old: `$.package[?(@.name.value == \"opendart\")].version`, replacement: `$.package[?(@.name == \"opendart\")].version`, invariant: "updates the workspace lock and CLI SDK pin"},
		{name: "Rust CLI pin updater", artifact: configArtifact, old: `"type": "generic",
          "path": "/sdk/rust/crates/opendart-cli/Cargo.toml"`, replacement: `"type": "generic",
          "path": "/sdk/rust/crates/other/Cargo.toml"`, invariant: "updates the workspace lock and CLI SDK pin"},
		// The inserted duplicate follows the original release-type, so JSON's last-key-wins behavior targets only the CLI object.
		{name: "CLI release type", artifact: configArtifact, old: `"component": "opendart-cli"`, replacement: `"release-type": "simple",
      "component": "opendart-cli"`, invariant: "CLI package release-type"},
		{name: "CLI component", artifact: configArtifact, old: `"component": "opendart-cli"`, replacement: `"component": "other-cli"`, invariant: "CLI package component"},
		{name: "CLI component tag", artifact: configArtifact, old: `"component": "opendart-cli",
      "include-component-in-tag": true`, replacement: `"component": "opendart-cli",
      "include-component-in-tag": false`, invariant: "CLI package include-component-in-tag"},
		{name: "CLI lock selector", artifact: configArtifact, old: `$.package[?(@.name.value == \"opendart-cli\")].version`, replacement: `$.package[?(@.name == \"opendart-cli\")].version`, invariant: "CLI package updates its workspace lock version"},
		{name: "Cargo lock mismatch", artifact: rustLockArtifact, old: "name = \"opendart\"\nversion = \"0.1.0\"", replacement: "name = \"opendart\"\nversion = \"0.1.1\"", invariant: "matches the crate package version"},
		{name: "CLI Cargo lock mismatch", artifact: rustLockArtifact, old: "name = \"opendart-cli\"\nversion = \"0.1.0\"", replacement: "name = \"opendart-cli\"\nversion = \"0.1.1\"", invariant: "matches the CLI crate package version"},
		{name: "duplicate CLI Cargo lock package", artifact: rustLockArtifact, old: "[[package]]\nname = \"opendart-cli\"\nversion = \"0.1.0\"", replacement: "[[package]]\nname = \"opendart-cli\"\nversion = \"0.1.0\"\n\n[[package]]\nname = \"opendart-cli\"\nversion = \"0.1.0\"", invariant: "contains one opendart-cli package version"},
		{name: "registry publish in release", artifact: releaseWorkflowArtifact, old: "mkdir release-assets", replacement: "cargo publish\n          mkdir release-assets", invariant: "keeps registry credentials out"},
		{name: "registry publish in verify", artifact: verificationScriptArtifact, old: "go vet ./...", replacement: "cargo publish", invariant: "excludes package publication"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := fixture.copy(t)
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
	fixture := newReleaseArtifactFixture(t)
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
		{
			name: "CLI inventory order", artifact: rustCLIPackageListArtifact,
			old: ".cargo_vcs_info.json\nCHANGELOG.md\n", replacement: "CHANGELOG.md\n.cargo_vcs_info.json\n",
			invariant: "is sorted for deterministic comparison",
		},
		{
			name: "required CLI package evidence", artifact: rustCLIPackageListArtifact,
			old: "tests/live_smoke.rs\n", replacement: "",
			invariant: "contains required package evidence",
		},
		{
			name: "private CLI package input", artifact: rustCLIPackageListArtifact,
			old: ".cargo_vcs_info.json\nCHANGELOG.md\n", replacement: ".cargo_vcs_info.json\n.github/secret\nCHANGELOG.md\n",
			invariant: "excludes repository-private inputs",
		},
		{
			name: "CLI SDK timeout mirror", artifact: rustCLIExecutionArtifact,
			old:         `const SDK_DEFAULT_TOTAL_TIMEOUT: Duration = Duration::from_secs(60);`,
			replacement: `const SDK_DEFAULT_TOTAL_TIMEOUT: Duration = Duration::from_secs(61);`,
			invariant:   "mirrors the published SDK total timeout default",
		},
		{
			name: "CLI registry scope", artifact: rustCLICargoArtifact,
			old: `publish = ["crates-io"]`, replacement: `publish = true`,
			invariant: "authorizes only the crates.io registry",
		},
		{
			name: "CLI packaged release documentation", artifact: rustCLICargoArtifact,
			old: `, "CHANGELOG.md"`, replacement: ``,
			invariant: "packages the reviewed source distribution",
		},
		{
			name: "CLI package include allowlist", artifact: rustCLICargoArtifact,
			old: `, "LICENSE"]`, replacement: `, "LICENSE", ".*"]`,
			invariant: "packages the reviewed source distribution",
		},
		{
			name: "CLI package duplicate include", artifact: rustCLICargoArtifact,
			old: `, "LICENSE"]`, replacement: `, "src/**"]`,
			invariant: "packages the reviewed source distribution",
		},
		{
			name: "CLI SDK exact pin", artifact: rustCLICargoArtifact,
			old: `version = "=0.1.0"`, replacement: `version = "0.1.0"`,
			invariant: "exact-pins the workspace SDK version",
		},
		{
			name: "CLI SDK release marker", artifact: rustCLICargoArtifact,
			old: ` # x-release-please-version`, replacement: ``,
			invariant: "marks the exact SDK pin",
		},
		{
			name: "CLI JSON workspace inheritance", artifact: rustCLICargoArtifact,
			old: `serde_json.workspace = true`, replacement: `serde_json = "1"`,
			invariant: "inherits the reviewed JSON encoder behavior",
		},
		{
			name: "JSON encoder exact pin", artifact: rustWorkspaceArtifact,
			old: `serde_json = { version = "=1.0.150", features = ["arbitrary_precision", "preserve_order"] }`, replacement: `serde_json = "1"`,
			invariant: "exact-pins the reviewed JSON encoder behavior",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := fixture.copy(t)
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
	fixture := newReleaseArtifactFixture(t)
	root := fixture.copy(t)
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
	fixture := newReleaseArtifactFixture(t)
	root := fixture.copy(t)
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
	fixture := newReleaseArtifactFixture(t)
	root := fixture.copy(t)
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
	fixture := newReleaseArtifactFixture(t)
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
			invariant: "optionally published SDK component",
		},
		{
			name: "manifest SemVer", artifact: manifestArtifact,
			old: `"0.1.0"`, replacement: `"01.1.0"`,
			invariant: "specification version is SemVer",
		},
		{
			name: "config package scope", artifact: configArtifact,
			old: `"packages": {`, replacement: `"packages": { "extra": {},`,
			invariant: "contains only the specification, SDK, and CLI packages",
		},
		{
			name: "specification component path", artifact: configArtifact,
			old: `"openapi/generated": {`, replacement: `".": {`,
			invariant: "contains only the specification, SDK, and CLI packages",
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
			name: "separate component proposals", artifact: configArtifact,
			old: "\"separate-pull-requests\": true", replacement: "\"separate-pull-requests\": false",
			invariant: "isolates component release proposals",
		},
		{
			name: "SDK beta release", artifact: configArtifact,
			old: "\"release-as\": \"0.1.0-beta.1\"", replacement: "\"release-as\": \"0.1.0\"",
			invariant: "Rust package release-as",
		},
		{
			name: "CLI publication exclusion", artifact: configArtifact,
			old: "\"exclude-paths\": [", replacement: "\"exclude-paths\": [\"other\",",
			invariant: "CLI package remains excluded",
		},
		{
			name: "main-only release", artifact: releaseWorkflowArtifact,
			old: "      - main", replacement: "      - dev",
			invariant: "runs only for pushes to main",
		},
		{
			name: "release root permissions", artifact: releaseWorkflowArtifact,
			old: "permissions: {}", replacement: "permissions:\n  contents: write",
			invariant: "root permissions are empty",
		},
		{
			name: "release workflow shell bypass", artifact: releaseWorkflowArtifact,
			old: "permissions: {}", replacement: "defaults:\n  run:\n    shell: bash {0} || true\n\npermissions: {}",
			invariant: "workflow uses default run settings",
		},
		{
			name: "release extra job", artifact: releaseWorkflowArtifact,
			old: "jobs:\n  verify:", replacement: "jobs:\n  extra:\n    runs-on: ubuntu-latest\n    steps:\n      - run: echo unsafe\n\n  verify:",
			invariant: "contains only approved release jobs",
		},
		{
			name: "exact pushed verification", artifact: releaseWorkflowArtifact,
			old: "expected_sha: ${{ github.sha }}", replacement: "expected_sha: main",
			invariant: "verifies the exact pushed revision",
		},
		{
			name: "release job failure bypass", artifact: releaseWorkflowArtifact,
			old: "  release-please:\n    needs: verify", replacement: "  release-please:\n    continue-on-error: true\n    needs: verify",
			invariant: "release proposal and specification publication",
		},
		{
			name: "release SDK SHA output", artifact: releaseWorkflowArtifact,
			old: "sdk_sha: ${{ steps.component.outputs.sdk_sha }}", replacement: "sdk_sha: ${{ github.sha }}",
			invariant: "release proposal and specification publication",
		},
		{
			name: "pinned release action", artifact: releaseWorkflowArtifact,
			old: "googleapis/release-please-action@45996ed1f6d02564a971a2fa1b5860e934307cf7", replacement: "googleapis/release-please-action@v5",
			invariant: "approved pinned Release Please action",
		},
		{
			name: "tag-only recovery", artifact: releaseWorkflowArtifact,
			old: "tag exists without a matching GitHub release", replacement: "ignoring tag-only state",
			invariant: "component recovery fails closed",
		},
		{
			name: "tag probe API failure classification", artifact: releaseWorkflowArtifact,
			old: `if test "${http_status}" = 404`, replacement: `if test "${http_status}" = 403`,
			invariant: "component recovery fails closed",
		},
		{
			name: "SDK draft tag rejection", artifact: releaseWorkflowArtifact,
			old: `if test "${component}" = sdk; then`, replacement: `if test "${component}" = other; then`,
			invariant: "component recovery fails closed",
		},
		{
			name: "proposal number normalization", artifact: releaseWorkflowArtifact,
			old: "[.[].number]", replacement: "[.[]]",
			invariant: "validated release proposal numbers",
		},
		{
			name: "CLI release authorization", artifact: releaseWorkflowArtifact,
			old: "CLI publication is not authorized", replacement: "CLI publication accepted",
			invariant: "blocks CLI publication",
		},
		{
			name: "specification checksum", artifact: releaseWorkflowArtifact,
			old: "sha256sum openapi.bundle.yaml", replacement: "sha1sum openapi.bundle.yaml",
			invariant: "publishes immutable specification assets",
		},
		{
			name: "dispatcher bot identity", artifact: releaseWorkflowArtifact,
			old: ".author.login == \"app/github-actions\"", replacement: ".author.login != \"app/github-actions\"",
			invariant: "canonical Release Please PR head SHA",
		},
		{
			name: "dispatcher exact branch", artifact: releaseWorkflowArtifact,
			old: "--ref \"${branch}\"", replacement: "--ref main",
			invariant: "canonical Release Please PR head SHA",
		},
		{
			name: "dispatcher authority", artifact: releaseWorkflowArtifact,
			old: "      actions: write", replacement: "      actions: read",
			invariant: "isolates actions-write authority",
		},
		{
			name: "fixed SDK environment", artifact: releaseWorkflowArtifact,
			old: "environment: crates-io-opendart", replacement: "environment: unprotected",
			invariant: "fixed component identity",
		},
		{
			name: "release workflow registry credential", artifact: releaseWorkflowArtifact,
			old: "      issues: write", replacement: "      issues: write\n      id-token: write",
			invariant: "keeps registry credentials out and actions-write authority isolated",
		},
		{
			name: "crate workflow callable only", artifact: rustCrateWorkflowArtifact,
			old: "  workflow_call:", replacement: "  workflow_dispatch:",
			invariant: "callable only",
		},
		{
			name: "crate workflow extra job", artifact: rustCrateWorkflowArtifact,
			old: "jobs:\n  candidate:", replacement: "jobs:\n  unsafe:\n    runs-on: ubuntu-latest\n    steps:\n      - run: echo unsafe\n\n  candidate:",
			invariant: "contains only candidate",
		},
		{
			name: "crate finalizer condition bypass", artifact: rustCrateWorkflowArtifact,
			old: "  finalize:\n    needs: reconcile", replacement: "  finalize:\n    if: ${{ always() }}\n    needs: reconcile",
			invariant: "matching GitHub draft",
		},
		{
			name: "crate reconciliation step bypass", artifact: rustCrateWorkflowArtifact,
			old: "      - name: Acquire and verify accepted crate", replacement: "      - name: Acquire and verify accepted crate\n        if: false",
			invariant: "verifies the accepted crate",
		},
		{
			name: "crate candidate exact identity", artifact: rustCrateWorkflowArtifact,
			old: "test \"${PACKAGE}\" = opendart", replacement: "test -n \"${PACKAGE}\"",
			invariant: "attests, verifies, packages",
		},
		{
			name: "crate candidate checkout", artifact: rustCrateWorkflowArtifact,
			old: "ref: ${{ inputs.candidate_sha }}", replacement: "ref: main",
			invariant: "exact SDK candidate",
		},
		{
			name: "crate protected environment", artifact: rustCrateWorkflowArtifact,
			old: "environment: ${{ inputs.environment }}", replacement: "environment: unprotected",
			invariant: "protected publication job",
		},
		{
			name: "crate registry token boundary", artifact: rustCrateWorkflowArtifact,
			old: "CARGO_REGISTRY_TOKEN: ${{ secrets.CARGO_REGISTRY_TOKEN }}", replacement: "CARGO_REGISTRY_TOKEN: ${{ secrets.OTHER_TOKEN }}",
			invariant: "publishes at most once",
		},
		{
			name: "crate tokenless reconciliation", artifact: rustCrateWorkflowArtifact,
			old: "if test \"${publish_required}\" = true; then\n            test -n \"${CARGO_REGISTRY_TOKEN}\"", replacement: "test -n \"${CARGO_REGISTRY_TOKEN}\"\n          if test \"${publish_required}\" = true; then",
			invariant: "publishes at most once",
		},
		{
			name: "crate artifact digest format", artifact: rustCrateWorkflowArtifact,
			old: "grep -Eq '^[0-9a-f]{64}$'", replacement: "grep -Eq '^sha256:[0-9a-f]{64}$'",
			invariant: "publishes at most once and reconciles registry acceptance and ownership",
		},
		{
			name: "crate one-shot publish", artifact: rustCrateWorkflowArtifact,
			old: "cargo +1.97.1 publish --locked --no-verify", replacement: "cargo +1.97.1 publish",
			invariant: "publishes at most once",
		},
		{
			name: "crate appended CLI publish", artifact: rustCrateWorkflowArtifact,
			old: "cargo +1.97.1 publish --locked --no-verify --manifest-path \"${PACKAGE_PATH}/Cargo.toml\"", replacement: "cargo +1.97.1 publish --locked --no-verify --manifest-path \"${PACKAGE_PATH}/Cargo.toml\"\n            cargo +1.97.1 publish --locked --no-verify --manifest-path sdk/rust/crates/opendart-cli/Cargo.toml",
			invariant: "publishes at most once",
		},
		{
			name: "crate accepted artifact verification", artifact: rustCrateWorkflowArtifact,
			old: "go run ./cmd/opendart-tool verify-crate-artifact", replacement: "echo skip-verification",
			invariant: "verifies the accepted crate",
		},
		{
			name:     "crate placeholder accepted evidence",
			artifact: rustCrateWorkflowArtifact,
			old: `go run ./cmd/opendart-tool verify-crate-artifact \
            --candidate candidate-evidence/candidate.crate \
            --accepted accepted.crate \
            --inventory candidate-evidence/inventory.txt \
            --package "${PACKAGE}" \
            --version "${VERSION}" \
            --revision "${CANDIDATE_SHA}" \
            --vcs-path "${VCS_PATH}" \
            --registry-checksum "${registry_checksum}" \
            > crate-verification.json`,
			replacement: `if false; then
            go run ./cmd/opendart-tool verify-crate-artifact \
              --candidate candidate-evidence/candidate.crate \
              --accepted accepted.crate \
              --inventory candidate-evidence/inventory.txt \
              --package "${PACKAGE}" \
              --version "${VERSION}" \
              --revision "${CANDIDATE_SHA}" \
              --vcs-path "${VCS_PATH}" \
              --registry-checksum "${registry_checksum}" \
              > crate-verification.json
          fi
          printf '{}\n' > crate-verification.json`,
			invariant: "verifies the accepted crate",
		},
		{
			name: "crate clean consumer", artifact: rustCrateWorkflowArtifact,
			old: "default-features = false", replacement: "default-features = true",
			invariant: "verifies the accepted crate",
		},
		{
			name: "crate clean consumer execution", artifact: rustCrateWorkflowArtifact,
			old: "cargo +1.97.1 run --locked", replacement: "cargo +1.97.1 check --locked",
			invariant: "verifies the accepted crate",
		},
		{
			name: "crate docs readiness", artifact: rustCrateWorkflowArtifact,
			old: "https://docs.rs/${PACKAGE}/${VERSION}/${PACKAGE}/", replacement: "https://example.com/",
			invariant: "verifies the accepted crate",
		},
		{
			name: "crate finalizer latest policy", artifact: rustCrateWorkflowArtifact,
			old: "--latest=false", replacement: "--latest",
			invariant: "matching GitHub draft",
		},
		{
			name: "crate finalizer tag probe API failure classification", artifact: rustCrateWorkflowArtifact,
			old: `if test "${http_status}" = 404`, replacement: `if test "${http_status}" = 403`,
			invariant: "matching GitHub draft",
		},
		{
			name: "crate finalizer published recovery", artifact: rustCrateWorkflowArtifact,
			old: "false)\n              test \"$(gh api", replacement: "false)\n              exit 1\n              test \"$(gh api",
			invariant: "matching GitHub draft",
		},
		{
			name: "crate finalizer permission", artifact: rustCrateWorkflowArtifact,
			old: "      contents: write", replacement: "      contents: read",
			invariant: "matching GitHub draft",
		},
		{
			name: "crate OIDC not yet authorized", artifact: rustCrateWorkflowArtifact,
			old: "permissions: {}", replacement: "permissions:\n  id-token: write",
			invariant: "starts without authority",
		},
		{
			name: "crate pinned action", artifact: rustCrateWorkflowArtifact,
			old: "actions/upload-artifact@043fb46d1a93c77aae656e7c1c64a875d1fc6a0a", replacement: "actions/upload-artifact@v4",
			invariant: "third-party step action is pinned",
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
			old: "jobs:\n  go:", replacement: "jobs:\n  extra:\n    runs-on: ubuntu-latest\n    timeout-minutes: 5\n    steps:\n      - name: Unexpected\n        run: echo unexpected\n\n  go:",
			invariant: "contains only approved verification jobs",
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
			name: "verify trigger SHA binding", artifact: verifyWorkflowArtifact,
			old: `test "${GITHUB_SHA}" = "${EXPECTED_SHA}"`, replacement: `test -n "${GITHUB_SHA}"`,
			invariant: "approved verification steps",
		},
		{
			name: "verify triggers", artifact: verifyWorkflowArtifact,
			old: "  workflow_dispatch:", replacement: "  schedule:",
			invariant: "supports workflow_dispatch",
		},
		{
			name: "verify extra trigger", artifact: verifyWorkflowArtifact,
			old: "  pull_request:", replacement: "  pull_request:\n  push:",
			invariant: "supports only approved triggers",
		},
		{
			name: "verify concurrency group", artifact: verifyWorkflowArtifact,
			old: "group: ${{ github.workflow }}-${{ github.event.pull_request.number || github.run_id }}", replacement: "group: ${{ github.workflow }}-${{ github.ref }}",
			invariant: "cancels only superseded verification runs",
		},
		{
			name: "verify concurrency cancellation", artifact: verifyWorkflowArtifact,
			old: "cancel-in-progress: true", replacement: "cancel-in-progress: false",
			invariant: "cancels only superseded verification runs",
		},
		{
			name: "canonical verify command", artifact: verificationScriptArtifact,
			old: "go run ./cmd/opendart-tool verify --repository-root .", replacement: "go run ./cmd/opendart-tool lint --root openapi/openapi.yaml",
			invariant: "matches the reviewed fast, pull-request, and exhaustive tier contract",
		},
		{
			name: "Go vet command", artifact: verificationScriptArtifact,
			old: "go vet ./...", replacement: "go vet ./cmd/...",
			invariant: "matches the reviewed fast, pull-request, and exhaustive tier contract",
		},
		{
			name: "race-enabled Go tests", artifact: verificationScriptArtifact,
			old: "go test -race ./...", replacement: "go test ./...",
			invariant: "matches the reviewed fast, pull-request, and exhaustive tier contract",
		},
		{
			name: "structured CLI loopback tests", artifact: verificationScriptArtifact,
			old: "RUSTFLAGS=\"--cfg opendart_compat\" cargo +1.97.1 test --locked --offline --manifest-path sdk/rust/Cargo.toml -p opendart-cli --test structured_loopback", replacement: "cargo +1.97.1 test --locked --offline --manifest-path sdk/rust/Cargo.toml -p opendart-cli --test structured_loopback",
			invariant: "matches the reviewed fast, pull-request, and exhaustive tier contract",
		},
		{
			name: "binary CLI loopback tests", artifact: verificationScriptArtifact,
			old: "RUSTFLAGS=\"--cfg opendart_compat\" cargo +1.97.1 test --locked --offline --manifest-path sdk/rust/Cargo.toml -p opendart-cli --test binary_loopback", replacement: "cargo +1.97.1 test --locked --offline --manifest-path sdk/rust/Cargo.toml -p opendart-cli --test binary_loopback",
			invariant: "matches the reviewed fast, pull-request, and exhaustive tier contract",
		},
		{
			name: "CLI no-default-features tests", artifact: verificationScriptArtifact,
			old: "cargo +1.97.1 test --locked --offline --manifest-path sdk/rust/Cargo.toml -p opendart-cli --no-default-features", replacement: "cargo +1.97.1 test --locked --offline --manifest-path sdk/rust/Cargo.toml -p opendart --no-default-features",
			invariant: "matches the reviewed fast, pull-request, and exhaustive tier contract",
		},
		{
			name: "CLI MSRV no-default-features check", artifact: verificationScriptArtifact,
			old: "cargo +1.85.0 check --locked --offline --manifest-path sdk/rust/Cargo.toml -p opendart-cli --all-targets --no-default-features", replacement: "cargo +1.85.0 check --locked --offline --manifest-path sdk/rust/Cargo.toml -p opendart-cli --no-default-features",
			invariant: "matches the reviewed fast, pull-request, and exhaustive tier contract",
		},
		{
			name: "CLI package inventory diff", artifact: verificationScriptArtifact,
			old: "diff -u sdk/rust/opendart-cli-package-files.txt \"${cli_package_files}\"", replacement: "test -s \"${cli_package_files}\"",
			invariant: "matches the reviewed fast, pull-request, and exhaustive tier contract",
		},
		{
			name: "workspace package dry run", artifact: verificationScriptArtifact,
			old: "cargo +1.97.1 package --workspace --locked --offline --manifest-path sdk/rust/Cargo.toml", replacement: "cargo +1.97.1 package --locked --offline --manifest-path sdk/rust/crates/opendart/Cargo.toml",
			invariant: "matches the reviewed fast, pull-request, and exhaustive tier contract",
		},
		{
			name: "Linux locked source install", artifact: verificationScriptArtifact,
			old: `CARGO_TARGET_DIR="${install_workspace}/target" cargo +1.97.1 install --locked --offline --path sdk/rust/crates/opendart-cli --root "${install_workspace}/root"`, replacement: `CARGO_TARGET_DIR="${install_workspace}/target" cargo +1.97.1 install --offline --path sdk/rust/crates/opendart-cli --root "${install_workspace}/root"`,
			invariant: "matches the reviewed fast, pull-request, and exhaustive tier contract",
		},
		{
			name: "Windows locked source install", artifact: verifyWorkflowArtifact,
			old: `cargo +1.97.1 install --locked --offline --path sdk/rust/crates/opendart-cli --root $installRoot`, replacement: `cargo +1.97.1 install --offline --path sdk/rust/crates/opendart-cli --root $installRoot`,
			invariant: "native artifact jobs use only approved steps",
		},
		{
			name: "Windows installed discovery", artifact: verifyWorkflowArtifact,
			old: `& $binary operations list | Out-Null`, replacement: `& $binary --help | Out-Null`,
			invariant: "native artifact jobs use only approved steps",
		},
		{
			name: "Windows source install failure", artifact: verifyWorkflowArtifact,
			old: `cargo +1.97.1 install --locked --offline --path sdk/rust/crates/opendart-cli --root $installRoot
          if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }`, replacement: `cargo +1.97.1 install --locked --offline --path sdk/rust/crates/opendart-cli --root $installRoot`,
			invariant: "native artifact jobs use only approved steps",
		},
		{
			name: "Windows installed version failure", artifact: verifyWorkflowArtifact,
			old: `& $binary --version
          if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }`, replacement: `& $binary --version`,
			invariant: "native artifact jobs use only approved steps",
		},
		{
			name: "Windows installed discovery failure", artifact: verifyWorkflowArtifact,
			old: `& $binary operations list | Out-Null
          if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }`, replacement: `& $binary operations list | Out-Null`,
			invariant: "native artifact jobs use only approved steps",
		},
		{
			name: "Windows native home discovery", artifact: verifyWorkflowArtifact,
			old: `$env:USERPROFILE = $installRoot`, replacement: `$env:USERPROFILE = $installWorkspace`,
			invariant: "native artifact jobs use only approved steps",
		},
		{
			name: "Windows home document avoids automatic variable", artifact: verifyWorkflowArtifact,
			old: `$homeDocument = & $binary | ConvertFrom-Json`, replacement: `$home = & $binary | ConvertFrom-Json`,
			invariant: "native artifact jobs use only approved steps",
		},
		{
			name: "native artifact compatibility cfg", artifact: verifyWorkflowArtifact,
			old: "RUSTFLAGS: --cfg opendart_compat", replacement: "RUSTFLAGS: --cfg other",
			invariant: "native artifact jobs use only approved steps",
		},
		{
			name: "native artifact command", artifact: verifyWorkflowArtifact,
			old: "cargo +1.97.1 test --locked --offline --manifest-path sdk/rust/Cargo.toml -p opendart-cli --test binary_loopback", replacement: "cargo +1.97.1 test --locked --offline --manifest-path sdk/rust/Cargo.toml -p opendart-cli",
			invariant: "native artifact jobs use only approved steps",
		},
		{
			name: "native artifact runner", artifact: verifyWorkflowArtifact,
			old: "runs-on: macos-latest", replacement: "runs-on: ubuntu-latest",
			invariant: "native artifact jobs use approved runners and timeouts",
		},
		{
			name: "native artifact protected environment", artifact: verifyWorkflowArtifact,
			old: "  artifact-macos:\n    runs-on:", replacement: "  artifact-macos:\n    environment: protected\n    runs-on:",
			invariant: "native artifact jobs use default execution controls",
		},
		{
			name: "Go job condition bypass", artifact: verifyWorkflowArtifact,
			old: "  go:\n    runs-on:", replacement: "  go:\n    if: always()\n    runs-on:",
			invariant: "verification work jobs use default execution controls",
		},
		{
			name: "Go job unsupported env", artifact: verifyWorkflowArtifact,
			old: "  go:\n    runs-on:", replacement: "  go:\n    env:\n      SAFE: value\n    runs-on:",
			invariant: "uses only supported YAML fields",
		},
		{
			name: "Go protected environment", artifact: verifyWorkflowArtifact,
			old: "  go:\n    runs-on:", replacement: "  go:\n    environment: protected\n    runs-on:",
			invariant: "verification work jobs use default execution controls",
		},
		{
			name: "Go runner", artifact: verifyWorkflowArtifact,
			old: "runs-on: ubuntu-latest", replacement: "runs-on: macos-latest",
			invariant: "verification work jobs use the approved runner and timeout",
		},
		{
			name: "Go timeout", artifact: verifyWorkflowArtifact,
			old: "timeout-minutes: 30", replacement: "timeout-minutes: 60",
			invariant: "verification work jobs use the approved runner and timeout",
		},
		{
			name: "Go job shell bypass", artifact: verifyWorkflowArtifact,
			old: "  go:\n    runs-on:", replacement: "  go:\n    defaults:\n      run:\n        shell: bash {0} || true\n    runs-on:",
			invariant: "verification work jobs use default run settings",
		},
		{
			name: "Go job working-directory bypass", artifact: verifyWorkflowArtifact,
			old: "  go:\n    runs-on:", replacement: "  go:\n    defaults:\n      run:\n        working-directory: nested\n    runs-on:",
			invariant: "verification work jobs use default run settings",
		},
		{
			name: "aggregate condition", artifact: verifyWorkflowArtifact,
			old: "    if: ${{ always() }}", replacement: "    if: ${{ success() }}",
			invariant: "aggregate verify job always evaluates dependency results",
		},
		{
			name: "aggregate missing dependency", artifact: verifyWorkflowArtifact,
			old: "      - artifact-windows\n    runs-on:", replacement: "    runs-on:",
			invariant: "aggregate verify job depends on every required job",
		},
		{
			name: "aggregate accepts skipped dependency", artifact: verifyWorkflowArtifact,
			old: "*=success) ;;", replacement: "*=success|*=skipped) ;;",
			invariant: "aggregate verify job rejects every non-success result",
		},
		{
			name: "aggregate result expression", artifact: verifyWorkflowArtifact,
			old: "GO_RESULT: ${{ needs.go.result }}", replacement: "GO_RESULT: success",
			invariant: "aggregate verify job rejects every non-success result",
		},
		{
			name: "aggregate continue-on-error", artifact: verifyWorkflowArtifact,
			old: "  verify:\n    if:", replacement: "  verify:\n    continue-on-error: true\n    if:",
			invariant: "aggregate verify job always evaluates dependency results",
		},
		{
			name: "aggregate protected environment", artifact: verifyWorkflowArtifact,
			old: "  verify:\n    if:", replacement: "  verify:\n    environment: protected\n    if:",
			invariant: "aggregate verify job always evaluates dependency results",
		},
		{
			name: "canonical verify step condition bypass", artifact: verifyWorkflowArtifact,
			old: "      - name: Run required Go verification\n        run:", replacement: "      - name: Run required Go verification\n        if: always()\n        run:",
			invariant: "verification steps use default execution controls",
		},
		{
			name: "verify step continue-on-error bypass", artifact: verifyWorkflowArtifact,
			old: "      - name: Run required Go verification\n        run:", replacement: "      - name: Run required Go verification\n        continue-on-error: true\n        run:",
			invariant: "verification steps use default execution controls",
		},
		{
			name: "verify step shell bypass", artifact: verifyWorkflowArtifact,
			old: "      - name: Run required Go verification\n        run:", replacement: "      - name: Run required Go verification\n        shell: bash {0} || true\n        run:",
			invariant: "verification steps use default run settings",
		},
		{
			name: "verify step working-directory bypass", artifact: verifyWorkflowArtifact,
			old: "      - name: Run required Go verification\n        run:", replacement: "      - name: Run required Go verification\n        working-directory: nested\n        run:",
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
			old: "ref: ${{ inputs.expected_sha || github.sha }}", replacement: "ref: main",
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
			old: "      - name: Run required Go verification\n        run:", replacement: "      - name: Run required Go verification\n        env:\n          SAFE: value\n        run:",
			invariant: "approved environment and shell",
		},
		{
			name: "verify secrets", artifact: verifyWorkflowArtifact,
			old: "run: ./scripts/verify go", replacement: "env:\n          TOKEN: ${{ secrets.GITHUB_TOKEN }}\n        run: ./scripts/verify go",
			invariant: "excludes GitHub secrets",
		},
		{
			name: "verify secrets bracket access", artifact: verifyWorkflowArtifact,
			old: "run: ./scripts/verify go", replacement: "env:\n          TOKEN: ${{ secrets['GITHUB_TOKEN'] }}\n        run: ./scripts/verify go",
			invariant: "excludes GitHub secrets",
		},
		{
			name: "verify github token property access", artifact: verifyWorkflowArtifact,
			old: "run: ./scripts/verify go", replacement: "env:\n          TOKEN: ${{ github.token }}\n        run: ./scripts/verify go",
			invariant: "excludes GitHub token",
		},
		{
			name: "verify github token index access", artifact: verifyWorkflowArtifact,
			old: "run: ./scripts/verify go", replacement: "env:\n          TOKEN: ${{ github[\"token\"] }}\n        run: ./scripts/verify go",
			invariant: "excludes GitHub token",
		},
		{
			name: "verify API key", artifact: verifyWorkflowArtifact,
			old: "run: ./scripts/verify go", replacement: "env:\n          OPENDART_API_KEY: unsafe\n        run: ./scripts/verify go",
			invariant: "excludes OpenDART API key",
		},
		{
			name: "verify sync", artifact: verifyWorkflowArtifact,
			old: "run: ./scripts/verify go", replacement: "run: ./scripts/verify go && go run ./cmd/opendart-tool sync",
			invariant: "excludes guide synchronization",
		},
		{
			name: "verify Node setup", artifact: verifyWorkflowArtifact,
			old: "      - name: Set up Go", replacement: "      - name: Set up Node.js\n        uses: actions/setup-node@820762786026740c76f36085b0efc47a31fe5020\n\n      - name: Set up Go",
			invariant: "excludes JavaScript or Node package tooling",
		},
		{
			name: "verify npm command", artifact: verifyWorkflowArtifact,
			old: "run: ./scripts/verify go", replacement: "run: npm run verify:opendart",
			invariant: "excludes JavaScript or Node package tooling",
		},
		{
			name: "verify nodejs command", artifact: verifyWorkflowArtifact,
			old: "run: ./scripts/verify go", replacement: "run: nodejs scripts/verify.js",
			invariant: "excludes JavaScript or Node package tooling",
		},
		{
			name: "verify alternate package manager", artifact: verifyWorkflowArtifact,
			old: "run: ./scripts/verify go", replacement: "run: yarn verify",
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
			name: "live Rust toolchain", artifact: liveWorkflowArtifact,
			old: "rustup toolchain install 1.97.1 --profile minimal", replacement: "rustup toolchain install stable --profile minimal",
			invariant: "installs the approved Rust toolchain",
		},
		{
			name: "live Rust dependency fetch", artifact: liveWorkflowArtifact,
			old: "cargo +1.97.1 fetch --locked --manifest-path sdk/rust/Cargo.toml", replacement: "cargo +1.97.1 fetch --manifest-path sdk/rust/Cargo.toml",
			invariant: "fetches locked Rust dependencies",
		},
		{
			name: "live credential-free runner build", artifact: liveWorkflowArtifact,
			old: "go build -o .live-bin/opendart-tool ./cmd/opendart-tool", replacement: "go build ./cmd/opendart-tool",
			invariant: "builds only the approved live runners",
		},
		{
			name: "live Rust CLI smoke", artifact: liveWorkflowArtifact,
			old: ".live-bin/live-smoke --exact structured_and_binary_live_paths_are_read_only_and_sanitized", replacement: ".live-bin/live-smoke",
			invariant: "runs only the approved live conformance commands",
		},
		{
			name: "live Rust CLI gate", artifact: liveWorkflowArtifact,
			old: "OPENDART_LIVE_TESTS: \"1\"", replacement: "OPENDART_LIVE_TESTS: \"true\"",
			invariant: "sets only the approved live request gates",
		},
		{
			name: "live preflight runner", artifact: liveWorkflowArtifact,
			old: ".live-bin/opendart-tool live-conformance --preflight-only --repository-root .", replacement: "go run ./cmd/opendart-tool live-conformance --preflight-only --repository-root .",
			invariant: "rechecks credential-free live gates",
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
			name: "drift shallow checkout", artifact: driftWorkflowArtifact,
			old: "fetch-depth: 0", replacement: "fetch-depth: 1",
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
			root := fixture.copy(t)
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

func TestScriptDigestMismatchDetailReportsComputedDigest(t *testing.T) {
	const script = "echo changed"
	want := fmt.Sprintf("computed script SHA-256: %x", sha256.Sum256([]byte(script)))
	if got := scriptDigestMismatchDetail(script, "different"); got != want {
		t.Fatalf("script digest detail = %q, want %q", got, want)
	}
	if got := scriptDigestMismatchDetail(script, scriptDigest(script)); got != "" {
		t.Fatalf("matching script digest detail = %q, want empty", got)
	}
}

func TestReleaseComponentRecoveryScenarios(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the release workflow runs with a POSIX shell on ubuntu-latest")
	}
	for _, tool := range []string{"bash", "jq"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("release workflow fixture requires %s: %v", tool, err)
		}
	}

	releaseSource, err := os.ReadFile(filepath.Join(repositoryRoot(t), filepath.FromSlash(releaseWorkflowArtifact)))
	if err != nil {
		t.Fatal(err)
	}
	var release workflow
	if err := yaml.Unmarshal(releaseSource, &release); err != nil {
		t.Fatal(err)
	}
	job := release.Jobs["release-please"]
	_, recovery, err := stepByID(job.Steps, "recovery")
	if err != nil {
		t.Fatal(err)
	}
	_, component, err := stepByID(job.Steps, "component")
	if err != nil {
		t.Fatal(err)
	}
	_, releaseStep, err := stepByID(job.Steps, "release")
	if err != nil {
		t.Fatal(err)
	}
	if !exactWorkflowExpression(releaseStep.If, "steps.recovery.outputs.components == '[]'") {
		t.Fatalf("Release Please condition = %q", releaseStep.If)
	}

	const (
		currentSHA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		sdkSHA     = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	)
	draftSpec := fmt.Sprintf(`{"tag_name":"v0.1.0","draft":true,"prerelease":false,"target_commitish":%q}`, currentSHA)
	draftBetaSDK := fmt.Sprintf(`{"tag_name":"opendart-v0.1.0-beta.1","draft":true,"prerelease":true,"target_commitish":%q}`, sdkSHA)
	draftStableSDK := fmt.Sprintf(`{"tag_name":"opendart-v0.1.0","draft":true,"prerelease":false,"target_commitish":%q}`, sdkSHA)
	completeSpec := fmt.Sprintf(`{"tag_name":"v0.1.0","draft":false,"prerelease":false,"target_commitish":%q}`, currentSHA)

	tests := []struct {
		name                   string
		sdkVersion             string
		releases               string
		specCreated            string
		sdkCreated             string
		specTag                string
		sdkTag                 string
		specSHA                string
		freshSDKVersion        string
		freshSDKSHA            string
		wantRunReleasePlease   bool
		wantSpecCreated        string
		wantSDKCreated         string
		wantSDKPrerelease      string
		wantSpecTag            string
		wantSpecVersion        string
		wantSpecSHA            string
		wantSDKTag             string
		wantSDKVersion         string
		wantSDKSHA             string
		wantRecoveryComponents int
	}{
		{
			name: "fresh dual component beta", sdkVersion: "0.1.0-beta.1", releases: `[[]]`,
			specCreated: "true", sdkCreated: "true", specTag: "v0.1.0", sdkTag: "opendart-v0.1.0-beta.1",
			specSHA: currentSHA, freshSDKVersion: "0.1.0-beta.1", freshSDKSHA: sdkSHA,
			wantRunReleasePlease: true, wantSpecCreated: "true", wantSDKCreated: "true", wantSDKPrerelease: "true",
			wantSpecTag: "v0.1.0", wantSpecVersion: "0.1.0", wantSpecSHA: currentSHA,
			wantSDKTag: "opendart-v0.1.0-beta.1", wantSDKVersion: "0.1.0-beta.1", wantSDKSHA: sdkSHA,
		},
		{
			name: "dual draft recovery", sdkVersion: "0.1.0-beta.1", releases: `[[` + draftSpec + `,` + draftBetaSDK + `]]`,
			wantSpecCreated: "true", wantSDKCreated: "true", wantSDKPrerelease: "true", wantRecoveryComponents: 2,
			wantSpecTag: "v0.1.0", wantSpecVersion: "0.1.0", wantSpecSHA: currentSHA,
			wantSDKTag: "opendart-v0.1.0-beta.1", wantSDKVersion: "0.1.0-beta.1", wantSDKSHA: sdkSHA,
		},
		{
			name: "mixed complete and stable draft recovery", sdkVersion: "0.1.0", releases: `[[` + completeSpec + `,` + draftStableSDK + `]]`,
			wantSpecCreated: "false", wantSDKCreated: "true", wantSDKPrerelease: "false", wantRecoveryComponents: 2,
			wantSpecVersion: "0.1.0",
			wantSDKTag:      "opendart-v0.1.0", wantSDKVersion: "0.1.0", wantSDKSHA: sdkSHA,
		},
		{
			name: "fresh stable SDK", sdkVersion: "0.1.0", releases: `[[]]`,
			sdkCreated: "true", sdkTag: "opendart-v0.1.0", freshSDKVersion: "0.1.0", freshSDKSHA: sdkSHA,
			wantRunReleasePlease: true, wantSpecCreated: "false", wantSDKCreated: "true", wantSDKPrerelease: "false",
			wantSpecVersion: "0.1.0",
			wantSDKTag:      "opendart-v0.1.0", wantSDKVersion: "0.1.0", wantSDKSHA: sdkSHA,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			manifest := fmt.Sprintf("{\n  \"openapi/generated\": \"0.1.0\",\n  \"sdk/rust/crates/opendart\": %q\n}\n", test.sdkVersion)
			if err := os.WriteFile(filepath.Join(root, manifestArtifact), []byte(manifest), 0o600); err != nil {
				t.Fatal(err)
			}
			mockPath := installReleaseWorkflowMocks(t, root)
			recoveryOutput := runReleaseWorkflowScript(t, root, recovery.Run, map[string]string{
				"GH_TOKEN":          "test-token",
				"GITHUB_REPOSITORY": "cpaikr/opendart",
				"GITHUB_SHA":        currentSHA,
				"MOCK_RELEASES":     test.releases,
				"MOCK_TAG_SHA":      currentSHA,
				"PATH":              mockPath + string(os.PathListSeparator) + os.Getenv("PATH"),
			})

			var recovered []map[string]any
			if err := json.Unmarshal([]byte(recoveryOutput["components"]), &recovered); err != nil {
				t.Fatalf("recovery components = %q: %v", recoveryOutput["components"], err)
			}
			if len(recovered) != test.wantRecoveryComponents {
				t.Fatalf("recovery component count = %d, want %d: %v", len(recovered), test.wantRecoveryComponents, recovered)
			}
			if got := recoveryOutput["components"] == "[]"; got != test.wantRunReleasePlease {
				t.Fatalf("run Release Please = %t, want %t", got, test.wantRunReleasePlease)
			}

			componentOutput := runReleaseWorkflowScript(t, root, component.Run, map[string]string{
				"CLI_CREATED":         "false",
				"RECOVERY_COMPONENTS": recoveryOutput["components"],
				"SDK_CREATED":         test.sdkCreated,
				"SDK_SHA":             test.freshSDKSHA,
				"SDK_TAG":             test.sdkTag,
				"SDK_VERSION":         test.freshSDKVersion,
				"SPEC_CREATED":        test.specCreated,
				"SPEC_SHA":            test.specSHA,
				"SPEC_TAG":            test.specTag,
				"SPEC_VERSION":        "0.1.0",
			})
			for key, want := range map[string]string{
				"spec_release_created": test.wantSpecCreated,
				"spec_tag_name":        test.wantSpecTag,
				"spec_version":         test.wantSpecVersion,
				"spec_sha":             test.wantSpecSHA,
				"sdk_release_created":  test.wantSDKCreated,
				"sdk_tag_name":         test.wantSDKTag,
				"sdk_version":          test.wantSDKVersion,
				"sdk_sha":              test.wantSDKSHA,
				"sdk_prerelease":       test.wantSDKPrerelease,
			} {
				if got := componentOutput[key]; got != want {
					t.Errorf("%s = %q, want %q", key, got, want)
				}
			}
		})
	}
}

func installReleaseWorkflowMocks(t *testing.T, root string) string {
	t.Helper()
	bin := filepath.Join(root, "mock-bin")
	if err := os.Mkdir(bin, 0o700); err != nil {
		t.Fatal(err)
	}
	gh := `#!/usr/bin/env bash
set -e
case "$1" in
  api)
    case "$2" in
      --paginate)
        printf '%s\n' "${MOCK_RELEASES}"
        ;;
      --include)
        printf 'HTTP/2 404\n'
        exit 1
        ;;
      repos/*/commits/*)
        printf '%s\n' "${MOCK_TAG_SHA}"
        ;;
      *)
        echo "unsupported gh api invocation: $*" >&2
        exit 2
        ;;
    esac
    ;;
  *)
    echo "unsupported gh invocation: $*" >&2
    exit 2
    ;;
esac
`
	git := `#!/usr/bin/env bash
set -e
case "$1:$2" in
  rev-parse:HEAD)
    printf '%s\n' "${GITHUB_SHA}"
    ;;
  merge-base:--is-ancestor)
    exit 0
    ;;
  *)
    echo "unsupported git invocation: $*" >&2
    exit 2
    ;;
esac
`
	for name, source := range map[string]string{"gh": gh, "git": git} {
		if err := os.WriteFile(filepath.Join(bin, name), []byte(source), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	return bin
}

func runReleaseWorkflowScript(t *testing.T, dir, script string, environment map[string]string) map[string]string {
	t.Helper()
	outputPath := filepath.Join(t.TempDir(), "github-output")
	if err := os.WriteFile(outputPath, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "bash", "-e", "-o", "pipefail", "-c", script)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GITHUB_OUTPUT="+outputPath)
	for key, value := range environment {
		cmd.Env = append(cmd.Env, key+"="+value)
	}
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("workflow script failed: %v\n%s", err, output)
	}
	source, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	result := make(map[string]string)
	for line := range strings.SplitSeq(strings.TrimSpace(string(source)), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			t.Fatalf("invalid workflow output line %q", line)
		}
		result[key] = value
	}
	return result
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
		{name: "recovery after release", first: "Detect interrupted component release", last: "Run Release Please"},
		{name: "upload before prepare", first: "Prepare specification release assets", last: "Upload specification release assets"},
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

			if err := checkReleasePipelineWorkflow(release, string(releaseSource)); err == nil {
				t.Fatal("checkReleasePipelineWorkflow() error = nil")
			}
		})
	}
}

type releaseArtifactFixture struct {
	sourceRoot   string
	packageNames map[string]string
}

func newReleaseArtifactFixture(t *testing.T) releaseArtifactFixture {
	t.Helper()
	sourceRoot := repositoryRoot(t)
	packageNames, err := loadAuditedOwnershipPackageNames(sourceRoot)
	if err != nil {
		t.Fatal(err)
	}
	return releaseArtifactFixture{sourceRoot: sourceRoot, packageNames: packageNames}
}

func (fixture releaseArtifactFixture) copy(t *testing.T) string {
	t.Helper()
	targetRoot := t.TempDir()
	for _, artifact := range []string{
		configArtifact,
		manifestArtifact,
		rustCargoArtifact,
		rustCLICargoArtifact,
		rustWorkspaceArtifact,
		rustLockArtifact,
		rustProvenanceArtifact,
		rustSDKClientArtifact,
		rustCLIExecutionArtifact,
		rustPackageListArtifact,
		rustCLIPackageListArtifact,
		canonicalBundleArtifact,
		releaseWorkflowArtifact,
		rustCrateWorkflowArtifact,
		verifyWorkflowArtifact,
		fullRaceWorkflowArtifact,
		verificationScriptArtifact,
		liveWorkflowArtifact,
		notifyWorkflowArtifact,
		driftWorkflowArtifact,
		driftNotifyArtifact,
	} {
		source, err := os.ReadFile(filepath.Join(fixture.sourceRoot, filepath.FromSlash(artifact)))
		if err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(targetRoot, filepath.FromSlash(artifact))
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			t.Fatal(err)
		}
		mode := os.FileMode(0o600)
		if artifact == verificationScriptArtifact {
			mode = 0o700
		}
		if err := os.WriteFile(target, source, mode); err != nil {
			t.Fatal(err)
		}
	}
	writeOwnershipFixture(t, targetRoot, fixture.packageNames)
	for _, sourcePath := range canonicalSpecificationSources {
		copyPath(t, fixture.sourceRoot, targetRoot, sourcePath)
	}
	gitDirectory, err := exec.Command("git", "-C", fixture.sourceRoot, "rev-parse", "--absolute-git-dir").Output()
	if err != nil {
		t.Fatal(err)
	}
	gitFile := []byte("gitdir: " + strings.TrimSpace(string(gitDirectory)) + "\n")
	if err := os.WriteFile(filepath.Join(targetRoot, ".git"), gitFile, 0o600); err != nil {
		t.Fatal(err)
	}
	return targetRoot
}

func writeOwnershipFixture(t *testing.T, targetRoot string, packageNames map[string]string) {
	t.Helper()
	direct := make(map[string]bool)
	cancellation := make(map[string]bool)
	global := make(map[string]bool)
	for _, packagePath := range targetedRacePackages {
		direct[packagePath] = true
		global[packagePath] = true
	}
	for _, packagePath := range reviewedCancellationPackages {
		cancellation[packagePath] = true
	}
	for _, packagePath := range reviewedReadOnlyGlobalPackages {
		global[packagePath] = true
	}
	packages := make(map[string]bool)
	for packagePath := range direct {
		packages[packagePath] = true
	}
	for packagePath := range cancellation {
		packages[packagePath] = true
	}
	for packagePath := range global {
		packages[packagePath] = true
	}
	for packagePath := range packages {
		name, ok := packageNames[packagePath]
		if !ok {
			t.Fatalf("audited package %s has no source-derived package name", packagePath)
		}
		var source strings.Builder
		source.WriteString("package " + name + "\n")
		if cancellation[packagePath] {
			source.WriteString("\nimport \"context\"\n")
		}
		if direct[packagePath] {
			source.WriteString("\nvar raceOwned chan struct{}\n")
		} else if global[packagePath] {
			source.WriteString("\nvar reviewedState = 1\n")
		}
		if cancellation[packagePath] {
			source.WriteString("\nfunc cancelOwned() { _, _ = context.WithCancel(context.Background()) }\n")
		}
		path := filepath.Join(targetRoot, filepath.FromSlash(strings.TrimPrefix(packagePath, "./")), "ownership.go")
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(source.String()), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func loadAuditedOwnershipPackageNames(sourceRoot string) (map[string]string, error) {
	packages := make(map[string]bool)
	for _, packagePath := range targetedRacePackages {
		packages[packagePath] = true
	}
	for _, packagePath := range reviewedCancellationPackages {
		packages[packagePath] = true
	}
	for _, packagePath := range reviewedReadOnlyGlobalPackages {
		packages[packagePath] = true
	}
	names := make(map[string]string, len(packages))
	for packagePath := range packages {
		name, err := sourcePackageName(sourceRoot, packagePath)
		if err != nil {
			return nil, err
		}
		names[packagePath] = name
	}
	return names, nil
}

func sourcePackageName(sourceRoot, packagePath string) (string, error) {
	directory := filepath.Join(sourceRoot, filepath.FromSlash(strings.TrimPrefix(packagePath, "./")))
	entries, err := os.ReadDir(directory)
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(directory, entry.Name())
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.PackageClauseOnly)
		if err != nil {
			return "", err
		}
		return file.Name.Name, nil
	}
	return "", fmt.Errorf("audited package %s has no production Go source", packagePath)
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
