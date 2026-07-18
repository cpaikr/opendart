package guide

import (
	"context"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
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
		validate: func(staging string, complete bool) error {
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
		validate: func(string, bool) error { return validationFailure },
		publish:  publishGenerated,
	}
	_, err := syncWithDependencies(context.Background(), SyncOptions{RepositoryRoot: root, Output: output, CheckedAt: "2026-07-17"}, dependencies)
	if !errors.Is(err, validationFailure) {
		t.Fatalf("sync error = %v", err)
	}
	if _, err := os.Stat(output); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("output was published: %v", err)
	}
}

func TestSyncDoesNotPublishWhenCanceledDuringValidation(t *testing.T) {
	root := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	published := false
	dependencies := syncDependencies{
		fetcher: inertFetcher{},
		acquire: func(context.Context, Fetcher, []string) ([]Endpoint, error) {
			return []Endpoint{{}}, nil
		},
		generate: func(_ []Endpoint, options GenerateOptions) (GenerationResult, error) {
			writeManagedFixture(t, options.OutputDir, "new")
			return GenerationResult{}, nil
		},
		validate: func(string, bool) error {
			cancel()
			return nil
		},
		publish: func(string, string, string) error {
			published = true
			return nil
		},
	}
	_, err := syncWithDependencies(ctx, SyncOptions{RepositoryRoot: root, Output: filepath.Join(root, "openapi"), CheckedAt: "2026-07-17"}, dependencies)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("sync error = %v", err)
	}
	if published {
		t.Fatal("canceled sync published staged output")
	}
}

func TestSyncRejectsBroadOutputBeforeAcquisition(t *testing.T) {
	root := t.TempDir()
	acquired := false
	dependencies := syncDependencies{
		fetcher: inertFetcher{},
		acquire: func(context.Context, Fetcher, []string) ([]Endpoint, error) {
			acquired = true
			return nil, nil
		},
		generate: Generate,
		validate: validateStaging,
		publish:  publishGenerated,
	}
	_, err := syncWithDependencies(context.Background(), SyncOptions{RepositoryRoot: root, Output: root, CheckedAt: "2026-07-17"}, dependencies)
	if err == nil || !strings.Contains(err.Error(), "broad directory") {
		t.Fatalf("error = %v", err)
	}
	if acquired {
		t.Fatal("acquisition ran for a broad output target")
	}
}

func TestSyncRejectsPartialCanonicalOutputBeforeAcquisition(t *testing.T) {
	root := t.TempDir()
	acquired := false
	dependencies := syncDependencies{
		fetcher: inertFetcher{},
		acquire: func(context.Context, Fetcher, []string) ([]Endpoint, error) {
			acquired = true
			return nil, nil
		},
		generate: Generate,
		validate: validateStaging,
		publish:  publishGenerated,
	}
	_, err := syncWithDependencies(context.Background(), SyncOptions{
		RepositoryRoot: root, Output: filepath.Join(root, "openapi"), CheckedAt: "2026-07-17", Only: []string{"DS001-2019001"},
	}, dependencies)
	if err == nil || !strings.Contains(err.Error(), "non-canonical") {
		t.Fatalf("error = %v", err)
	}
	if acquired {
		t.Fatal("acquisition ran for a partial canonical output target")
	}
}
