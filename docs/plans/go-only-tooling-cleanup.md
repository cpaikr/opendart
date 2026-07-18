# Go-Only Tooling Cleanup

## Objective

Finish the repository-tooling migration by moving the focused multi-company
probe to the internal Go CLI and removing repository-owned Node.js, npm, and
Redocly surfaces. Preserve the released OpenAPI contract and the probe's
credential, request, assertion, and sanitized-report behavior.

This plan completes the remaining work in the
[Go tooling migration](go-tooling-migration.md). It does not implement public
guide-drift automation or the general
[live-conformance runner](live-conformance.md).

## Current state

- The Go CLI owns guide synchronization, catalog and reference validation,
  strict linting, deterministic bundling and freshness, offline verification,
  release/workflow guards, and an additive focused multi-company probe.
- The Go probe preserves the existing ten-request sequence, OpenAPI-backed
  response checks, endpoint assertions, pacing, one-attempt HTTP policy, and
  allowlisted report, with bounded response bodies and offline HTTP coverage.
- `scripts/probe-multi-company.mjs` remains the operational probe only until
  the second delivery slice cuts current entry points over to Go.
- `package.json` is both the probe dependency manifest and a command alias
  layer over the Go CLI. The remaining Node and Redocly scripts are dormant
  migration scaffolding.
- Credential-free CI still installs Node.js and npm dependencies only to run
  the focused probe tests.

## Decisions

- Use direct `go run ./cmd/opendart-tool ...` commands for local documentation
  and CI. Do not replace npm scripts with Just, Make, shell wrappers, or another
  task runner.
- Add `probe-multi-company` to the existing CLI. Read `OPENDART_API_KEY` only
  from the process environment; never accept it as an argument.
- Put the current probe behavior behind one narrow internal module. Keep its
  interface specific to the implemented probe rather than predicting the
  eventual general live-conformance runner.
- Keep production HTTP behavior, request pacing, and their test adapters
  internal to that module. Extract a shared drift/live seam only when another
  implemented consumer creates real variation.
- Preserve the existing versioned, allowlisted observation report. Parse and
  validate bounded response bodies, then discard them.
- Treat the shared drift/live automation contract as future cross-plan policy,
  not as an implemented module or a prerequisite for this cleanup.
- Keep `opendart-tool verify` responsible for committed repository artifacts.
  CI separately owns `go vet`, race-enabled tests, and the verifier invocation.
- Keep Release Please as pinned workflow automation. Its configuration,
  manifest, changelog, tag, and release assets remain independent of
  `package.json`.
- Preserve historical Node and Redocly context in the accepted ADR and progress
  history. Remove only current operational guidance and obsolete scaffolding.

## Target command surface

```sh
go run ./cmd/opendart-tool sync --checked-at YYYY-MM-DD
go run ./cmd/opendart-tool catalog --root openapi/openapi.yaml
go run ./cmd/opendart-tool lint --root openapi/openapi.yaml
go run ./cmd/opendart-tool bundle \
  --root openapi/openapi.yaml \
  --output openapi/generated/openapi.bundle.yaml
go run ./cmd/opendart-tool verify --repository-root .
go run ./cmd/opendart-tool probe-multi-company
```

Repository validation remains explicit:

```sh
go vet ./...
go test -race ./...
go run ./cmd/opendart-tool verify --repository-root .
```

## Ordered work

### 1. **Complete.** Port the focused probe with temporary parity

- Add the Go CLI command and a focused internal module with a small
  `Run`-style interface returning the sanitized report.
- Preserve canonical comma-separated requests, repeated-key controls,
  single-company baselines, JSON/XML identity extraction, meaningful success
  assertions, and report conclusions.
- Constrain the external request path to the exact OpenDART API origin, reject
  redirects, run sequentially without retries, preserve the current
  inter-request pacing, and enforce timeout and response size limits. Keep
  authenticated URLs, the key, and unrestricted bodies out of reports, logs,
  diagnostics, errors, and artifacts; the key necessarily exists in the
  outbound query at the private request seam.
- Reuse the OpenAPI response-validation boundary where the declared
  representation supports it; retain endpoint-specific assertions as typed
  probe policy rather than specification metadata.
- Port the offline behavior, pacing, and sanitization tests. Keep the Node probe
  only long enough to compare the two implementations during this slice.

### 2. Cut local and CI entry points over to Go

- Replace npm commands in current documentation and workflows with direct Go
  commands.
- Remove Node setup, npm caching, and dependency installation from the Verify
  workflow.
- Update the Go release guard and its tests to require the canonical Go
  verification command and reject Node/npm reintroduction in credential-free
  verification.
- Keep the release workflow's permission, ordering, action-pinning, recovery,
  and immutable-asset invariants unchanged.

### 3. Remove the superseded toolchain

- Delete `package.json`, `package-lock.json`, all repository-owned JavaScript
  tooling and tests, and `openapi/redocly.yaml`.
- Remove the empty `scripts/` directory and Node-specific ignore entries.
- Remove stale `scripts` release exclusions and update their exact guard
  expectations.
- Remove the temporary Go `compatibility` CLI command and compatibility-only
  report after retaining its lasting document, comparison, lint, response, and
  archive coverage under current interfaces. Remove migration-only fixtures
  only when equivalent current coverage exists. Do not remove OpenAPI
  `nodeType` XML metadata; it is unrelated to Node.js.

### 4. Close the migration state

- Update current architecture, usage, release, migration, and live-conformance
  documentation to describe the Go-only operational state.
- Mark this cleanup and the repository-tooling migration complete only after
  the Node dependency graph and duplicate implementations are gone.
- Leave guide drift, the general live operation matrix, credentialed workflow,
  issue notification, and production promotion for their dedicated plans.

## Next action

Land the additive probe slice into `dev`. Then start the second slice from that
merged state, cut current documentation and CI directly to Go, and remove the
superseded Node/npm/Redocly and temporary compatibility surfaces.

## Acceptance criteria

- One internal Go CLI owns every repository-operated command, including the
  focused credentialed probe.
- The probe preserves its request cases, semantic assertions, credential
  isolation, sequential pacing, no-retry policy, and allowlisted report without
  persisting raw bodies.
- Pull-request, manual, and release verification require only the Go toolchain
  and perform no OpenDART request or credential lookup.
- No tracked Node package manifest, Node dependency lock, JavaScript tooling,
  Redocly configuration, or duplicate generator/validator remains.
- Direct Go commands are the only documented local and CI interface; no
  replacement task runner is added.
- The committed bundle remains byte-for-byte fresh and has no semantic contract
  change from this tooling-only work.
- Operational Node/npm/Redocly references are removed. Intentional historical
  references remain clearly historical.
- The focused Go probe does not claim completion of the general
  live-conformance or guide-drift plans.

## Validation

```sh
go mod tidy
test -z "$(gofmt -l cmd internal)"
go vet ./...
go test -race ./...
go run ./cmd/opendart-tool verify --repository-root .
git diff --check
```

Before deleting the Node implementation, run its offline tests once alongside
the Go replacement. A supervised credentialed comparison is useful when an
authorized key is available, but absence of an external credential must not
block the offline cutover.

## Delivery and stopping boundary

Use two cohesive review slices: first the additive Go probe with parity
evidence, then the Go-only command/CI cutover and deletion. Each slice must pass
the repository review and validation requirements before the next begins.

Stop when the repository-owned toolchain is Go-only and this plan is complete.
Do not continue into scheduled guide drift, the general live-conformance
matrix, credential or notifier configuration, or promotion from `dev` to
`main`.

## Progress log

- 2026-07-18: Completed the additive focused-probe slice. Added the Go command
  and narrow internal module, preserved the fixed request matrix, assertions,
  pacing, one-attempt policy, and sanitized report, added bounded-body and
  OpenAPI response checks, and passed both the original Node 22.12.0 tests and
  the Go race-enabled suite. No credential was available for an optional live
  comparison; the cutover and deletion slice remains next.
