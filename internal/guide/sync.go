package guide

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type SyncOptions struct {
	RepositoryRoot string
	Output         string
	CheckedAt      string
	Only           []string
	ParityBaseline string
}

type SyncReport struct {
	Output           string `json:"output"`
	CheckedAt        string `json:"checkedAt"`
	LogicalEndpoints int    `json:"logicalEndpoints"`
	PhysicalPaths    int    `json:"physicalPaths"`
	Schemas          int    `json:"schemas"`
}

type syncDependencies struct {
	fetcher  Fetcher
	acquire  func(context.Context, Fetcher, []string) ([]Endpoint, error)
	generate func([]Endpoint, GenerateOptions) (GenerationResult, error)
	validate func(string, bool) error
	compare  func(string, string) error
	publish  func(string, string, string) error
}

func Sync(ctx context.Context, options SyncOptions) (SyncReport, error) {
	return syncWithDependencies(ctx, options, syncDependencies{
		fetcher:  NewHTTPFetcher(),
		acquire:  Acquire,
		generate: Generate,
		validate: validateStaging,
		compare:  compareStaged,
		publish:  publishGenerated,
	})
}

func syncWithDependencies(ctx context.Context, options SyncOptions, dependencies syncDependencies) (SyncReport, error) {
	if options.RepositoryRoot == "" || options.Output == "" || options.CheckedAt == "" {
		return SyncReport{}, errors.New("guide sync requires repository root, output, and checked-at")
	}
	if dependencies.fetcher == nil || dependencies.acquire == nil || dependencies.generate == nil || dependencies.validate == nil || dependencies.publish == nil {
		return SyncReport{}, errors.New("guide sync dependencies are incomplete")
	}
	if err := ValidateSyncTarget(options.RepositoryRoot, options.Output, len(options.Only) > 0); err != nil {
		return SyncReport{}, fmt.Errorf("validate guide sync output: %w", err)
	}
	endpoints, err := dependencies.acquire(ctx, dependencies.fetcher, options.Only)
	if err != nil {
		return SyncReport{}, fmt.Errorf("acquire official guide: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(options.Output), 0o755); err != nil {
		return SyncReport{}, fmt.Errorf("prepare guide sync output parent: %w", err)
	}
	staging, err := os.MkdirTemp(filepath.Dir(options.Output), ".opendart-stage-")
	if err != nil {
		return SyncReport{}, fmt.Errorf("create guide sync staging directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(staging) }()

	generated, err := dependencies.generate(endpoints, GenerateOptions{OutputDir: staging, CheckedAt: options.CheckedAt})
	if err != nil {
		return SyncReport{}, fmt.Errorf("generate staged OpenAPI: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return SyncReport{}, fmt.Errorf("validate staged OpenAPI: %w", err)
	}
	complete := len(options.Only) == 0
	if err := dependencies.validate(staging, complete); err != nil {
		return SyncReport{}, err
	}
	if err := ctx.Err(); err != nil {
		return SyncReport{}, fmt.Errorf("validate staged OpenAPI: %w", err)
	}
	if options.ParityBaseline != "" {
		if dependencies.compare == nil {
			return SyncReport{}, errors.New("guide sync parity comparison is not configured")
		}
		baseline := options.ParityBaseline
		if !filepath.IsAbs(baseline) {
			baseline = filepath.Join(options.RepositoryRoot, baseline)
		}
		if err := dependencies.compare(staging, baseline); err != nil {
			return SyncReport{}, err
		}
	}
	if err := ctx.Err(); err != nil {
		return SyncReport{}, fmt.Errorf("publish staged OpenAPI: %w", err)
	}
	if err := dependencies.publish(staging, options.Output, options.RepositoryRoot); err != nil {
		return SyncReport{}, fmt.Errorf("publish staged OpenAPI: %w", err)
	}
	return SyncReport{
		Output:           options.Output,
		CheckedAt:        options.CheckedAt,
		LogicalEndpoints: len(endpoints),
		PhysicalPaths:    generated.PhysicalPaths,
		Schemas:          generated.SchemaFiles,
	}, nil
}
