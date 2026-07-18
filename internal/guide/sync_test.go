package guide

import (
	"context"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

type inertFetcher struct{}

func (inertFetcher) Fetch(context.Context, *url.URL) ([]byte, error) { return nil, nil }

func TestSyncStagesValidatesComparesAndPublishes(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "openapi")
	var stagedDirectory string
	var compared bool
	dependencies := syncDependencies{
		fetcher: inertFetcher{},
		acquire: func(context.Context, Fetcher, []string) ([]Endpoint, error) {
			return []Endpoint{{EndpointSummary: EndpointSummary{LogicalOperationID: "DS001-1"}}}, nil
		},
		generate: func(_ []Endpoint, options GenerateOptions) (GenerationResult, error) {
			stagedDirectory = options.OutputDir
			writeManagedFixture(t, options.OutputDir, "new")
			return GenerationResult{PhysicalPaths: 2, SchemaFiles: 1}, nil
		},
		validate: func(_ context.Context, staging, _ string, complete bool, _ commandRunner) error {
			if staging != stagedDirectory || !complete {
				t.Fatalf("validation staging=%q complete=%v", staging, complete)
			}
			return nil
		},
		compare: func(staging, baseline string) error {
			compared = staging == stagedDirectory && baseline == filepath.Join(root, "accepted")
			return nil
		},
		publish: publishGenerated,
		runner:  &recordingRunner{},
	}
	report, err := syncWithDependencies(context.Background(), SyncOptions{
		RepositoryRoot: root,
		Output:         output,
		CheckedAt:      "2026-07-17",
		ParityBaseline: "accepted",
	}, dependencies)
	if err != nil {
		t.Fatal(err)
	}
	if !compared || report.LogicalEndpoints != 1 || report.PhysicalPaths != 2 || report.Schemas != 1 {
		t.Fatalf("compared=%v report=%+v", compared, report)
	}
	assertFileContent(t, filepath.Join(output, "openapi.yaml"), "new")
	if _, err := os.Stat(stagedDirectory); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("staging was not removed: %v", err)
	}
}

func TestSyncDoesNotPublishAfterValidationFailure(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "openapi")
	validationFailure := errors.New("validation failed")
	dependencies := syncDependencies{
		fetcher: inertFetcher{},
		acquire: func(context.Context, Fetcher, []string) ([]Endpoint, error) {
			return []Endpoint{{}}, nil
		},
		generate: func(_ []Endpoint, options GenerateOptions) (GenerationResult, error) {
			writeManagedFixture(t, options.OutputDir, "new")
			return GenerationResult{}, nil
		},
		validate: func(context.Context, string, string, bool, commandRunner) error { return validationFailure },
		publish:  publishGenerated,
		runner:   &recordingRunner{},
	}
	_, err := syncWithDependencies(context.Background(), SyncOptions{RepositoryRoot: root, Output: output, CheckedAt: "2026-07-17"}, dependencies)
	if !errors.Is(err, validationFailure) {
		t.Fatalf("sync error = %v", err)
	}
	if _, err := os.Stat(output); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("output was published: %v", err)
	}
}
