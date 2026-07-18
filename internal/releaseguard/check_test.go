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
			old: `"scripts"`, replacement: `"scripts", "openapi"`,
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
			invariant: "manual dispatch is disabled",
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
			name: "reusable verify", artifact: releaseWorkflowArtifact,
			old: "uses: ./.github/workflows/verify.yml", replacement: "uses: ./.github/workflows/other.yml",
			invariant: "has the reusable verify job",
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
			name: "release asset condition", artifact: releaseWorkflowArtifact,
			old: "if: ${{ steps.release.outputs.release_created == 'true' || steps.draft.outputs.recovering == 'true' }}", replacement: "if: ${{ steps.release.outputs.release_created == 'true' }}",
			invariant: "runs only for a created or recovered release",
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
			name: "verify triggers", artifact: verifyWorkflowArtifact,
			old: "  workflow_dispatch:", replacement: "  schedule:",
			invariant: "supports workflow_dispatch",
		},
		{
			name: "canonical verify command", artifact: verifyWorkflowArtifact,
			old: "npm run verify:opendart", replacement: "npm test",
			invariant: "runs the canonical repository verification command",
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
			old: "run: npm run verify:opendart", replacement: "env:\n          TOKEN: ${{ secrets.GITHUB_TOKEN }}\n        run: npm run verify:opendart",
			invariant: "excludes GitHub secrets",
		},
		{
			name: "verify API key", artifact: verifyWorkflowArtifact,
			old: "run: npm run verify:opendart", replacement: "env:\n          OPENDART_API_KEY: unsafe\n        run: npm run verify:opendart",
			invariant: "excludes OpenDART API key",
		},
		{
			name: "verify sync", artifact: verifyWorkflowArtifact,
			old: "run: npm run verify:opendart", replacement: "run: npm run verify:opendart && npm run sync:opendart",
			invariant: "excludes guide synchronization",
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
