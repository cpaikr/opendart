package guide

import (
	"context"
	"net/url"
	"sync"
)

// AbsoluteDriftRequestLimit bounds the complete credential-free guide read,
// including group inventory pages and detail pages.
const AbsoluteDriftRequestLimit = 200

// RequestBudget reports the inventory-derived ceiling and the number of guide
// pages whose acquisition was attempted.
type RequestBudget struct {
	Ceiling int `json:"ceiling"`
	Used    int `json:"used"`
}

// DriftAcquisition is the validated current guide inventory and its request
// evidence. A failed acquisition still returns the budget consumed so far.
type DriftAcquisition struct {
	Endpoints     []Endpoint
	RequestBudget RequestBudget
}

// AcquireDrift reads the current public guide once per page without applying
// committed inventory cardinalities.
func AcquireDrift(ctx context.Context) (DriftAcquisition, error) {
	fetcher, err := newDriftHTTPFetcher()
	if err != nil {
		return DriftAcquisition{}, err
	}
	return acquireDriftWithFetcher(ctx, fetcher, AbsoluteDriftRequestLimit)
}

func acquireDriftWithFetcher(ctx context.Context, fetcher Fetcher, absoluteLimit int) (result DriftAcquisition, err error) {
	if fetcher == nil {
		return result, sourceError("OpenDART guide fetcher is required", nil, nil)
	}
	if absoluteLimit < len(Groups) {
		return result, sourceError("OpenDART guide absolute request ceiling is invalid", map[string]any{
			"ceiling": absoluteLimit, "minimum": len(Groups),
		}, nil)
	}
	budget := newRequestBudget(absoluteLimit)
	defer func() { result.RequestBudget = budget.snapshot() }()
	boundedFetcher := budgetedFetcher{fetcher: fetcher, budget: budget}

	inventory, err := acquireInventory(ctx, boundedFetcher, currentInventory)
	if err != nil {
		return result, err
	}
	derivedCeiling := len(Groups) + len(inventory)
	if derivedCeiling > absoluteLimit {
		return result, sourceError("OpenDART guide inventory exceeds the absolute request ceiling", map[string]any{
			"ceiling": absoluteLimit, "required": derivedCeiling,
		}, nil)
	}
	if err := budget.setCeiling(derivedCeiling); err != nil {
		return result, err
	}
	result.Endpoints, err = acquireEndpoints(ctx, boundedFetcher, inventory)
	return result, err
}

type requestBudget struct {
	mu      sync.Mutex
	ceiling int
	used    int
}

func newRequestBudget(ceiling int) *requestBudget {
	return &requestBudget{ceiling: ceiling}
}

func (budget *requestBudget) reserve() error {
	budget.mu.Lock()
	defer budget.mu.Unlock()
	if budget.used >= budget.ceiling {
		return sourceError("OpenDART guide request budget exhausted", map[string]any{
			"ceiling": budget.ceiling, "used": budget.used,
		}, nil)
	}
	budget.used++
	return nil
}

func (budget *requestBudget) setCeiling(ceiling int) error {
	budget.mu.Lock()
	defer budget.mu.Unlock()
	if ceiling < budget.used {
		return sourceError("OpenDART guide request budget is below current usage", map[string]any{
			"ceiling": ceiling, "used": budget.used,
		}, nil)
	}
	budget.ceiling = ceiling
	return nil
}

func (budget *requestBudget) snapshot() RequestBudget {
	budget.mu.Lock()
	defer budget.mu.Unlock()
	return RequestBudget{Ceiling: budget.ceiling, Used: budget.used}
}

type budgetedFetcher struct {
	fetcher Fetcher
	budget  *requestBudget
}

func (fetcher budgetedFetcher) Fetch(ctx context.Context, sourceURL *url.URL) ([]byte, error) {
	if err := fetcher.budget.reserve(); err != nil {
		return nil, err
	}
	return fetcher.fetcher.Fetch(ctx, sourceURL)
}
