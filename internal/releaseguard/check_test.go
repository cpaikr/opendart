package releaseguard

import (
	"errors"
	"os"
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
			old: `".": "0.1.0"`, replacement: `".": "0.1.0", "extra": "0.1.0"`,
			invariant: "contains only the root component",
		},
		{
			name: "manifest SemVer", artifact: manifestArtifact,
			old: `"0.1.0"`, replacement: `"01.1.0"`,
			invariant: "root version is SemVer",
		},
		{
			name: "config package scope", artifact: configArtifact,
			old: `"packages": {`, replacement: `"packages": { "extra": {},`,
			invariant: "contains only the root package",
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
			name: "exclusions", artifact: configArtifact,
			old: `"internal"`, replacement: `"internal", "scripts"`,
			invariant: "root package exclude-paths",
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
			name: "release checkout credentials", artifact: releaseWorkflowArtifact,
			old: "persist-credentials: false", replacement: "persist-credentials: true",
			invariant: "checkout disables persisted credentials",
		},
		{
			name: "package publication", artifact: releaseWorkflowArtifact,
			old: "mkdir release-assets", replacement: "npm publish\n          mkdir release-assets",
			invariant: "does not publish packages or replace assets",
		},
		{
			name: "draft detection", artifact: releaseWorkflowArtifact,
			old: "gh release view", replacement: "gh release inspect",
			invariant: "draft recovery records gh release view",
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
			name: "released commit checkout", artifact: releaseWorkflowArtifact,
			old: "ref: ${{ steps.release.outputs.sha || steps.draft.outputs.sha }}", replacement: "ref: main",
			invariant: "released checkout uses the created or recovered SHA",
		},
		{
			name: "released commit uses checkout action", artifact: releaseWorkflowArtifact,
			old:         "      - name: Check out released commit\n        if: ${{ steps.release.outputs.release_created == 'true' || steps.draft.outputs.recovering == 'true' }}\n        uses: actions/checkout@9c091bb21b7c1c1d1991bb908d89e4e9dddfe3e0",
			replacement: "      - name: Check out released commit\n        if: ${{ steps.release.outputs.release_created == 'true' || steps.draft.outputs.recovering == 'true' }}\n        uses: actions/setup-node@820762786026740c76f36085b0efc47a31fe5020",
			invariant:   "released checkout uses the created or recovered SHA",
		},
		{
			name: "release asset condition", artifact: releaseWorkflowArtifact,
			old: "if: ${{ steps.release.outputs.release_created == 'true' || steps.draft.outputs.recovering == 'true' }}", replacement: "if: ${{ steps.release.outputs.release_created == 'true' }}",
			invariant: "runs only for a created or recovered release",
		},
		{
			name: "release asset condition extra clause", artifact: releaseWorkflowArtifact,
			old: "if: ${{ steps.release.outputs.release_created == 'true' || steps.draft.outputs.recovering == 'true' }}", replacement: "if: ${{ steps.release.outputs.release_created == 'true' || steps.draft.outputs.recovering == 'true' || always() }}",
			invariant: "runs only for a created or recovered release",
		},
		{
			name: "release asset continue-on-error bypass", artifact: releaseWorkflowArtifact,
			old: "      - name: Prepare release assets\n        if:", replacement: "      - name: Prepare release assets\n        continue-on-error: true\n        if:",
			invariant: "release step failures stop the job",
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
			name: "publish immutable release", artifact: releaseWorkflowArtifact,
			old: "--draft=false --latest", replacement: "--draft=false",
			invariant: "publishes the draft only after assets are verified",
		},
		{
			name: "verify workflow permissions", artifact: verifyWorkflowArtifact,
			old: "permissions:\n  contents: read", replacement: "permissions:\n  contents: write",
			invariant: "permissions are read-only",
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
			invariant: "runs the canonical repository verification command",
		},
		{
			name: "Go vet command", artifact: verifyWorkflowArtifact,
			old: "go vet ./...", replacement: "go vet ./cmd/...",
			invariant: "runs Go vet",
		},
		{
			name: "race-enabled Go tests", artifact: verifyWorkflowArtifact,
			old: "go test -race ./...", replacement: "go test ./...",
			invariant: "runs race-enabled Go tests",
		},
		{
			name: "verify job condition bypass", artifact: verifyWorkflowArtifact,
			old: "  verify:\n    runs-on:", replacement: "  verify:\n    if: always()\n    runs-on:",
			invariant: "verify job uses default execution controls",
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
			invariant: "runs the canonical repository verification command",
		},
		{
			name: "verify step continue-on-error bypass", artifact: verifyWorkflowArtifact,
			old: "      - name: Verify repository\n        run:", replacement: "      - name: Verify repository\n        continue-on-error: true\n        run:",
			invariant: "runs the canonical repository verification command",
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
			invariant: "checkout disables persisted credentials",
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
			invariant: "uses only approved Go verification steps",
		},
		{
			name: "verify local action", artifact: verifyWorkflowArtifact,
			old: "      - name: Set up Go", replacement: "      - name: Run local action\n        uses: ./actions/check\n\n      - name: Set up Go",
			invariant: "uses only approved Go verification steps",
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
		releaseWorkflowArtifact,
		verifyWorkflowArtifact,
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
	return targetRoot
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
