# Repository architecture

## Purpose and boundary

This repository publishes a source-backed OpenAPI description of OpenDART. The
released bundle is the product. Repository tooling exists only to acquire,
generate, validate, test, and release that specification; it is not a public
application, SDK, or reusable package.

The official development guide is authoritative for documented behavior.
Authenticated API observations are evidence about the live service and must
remain distinguishable from guide-sourced facts.

## System shape

```text
OpenDART guide -> guide acquisition -> normalized catalog -> OpenAPI boundary
                                                           -> fragments + bundle

committed OpenAPI -------------------------> offline verification -> release gate
guide + committed OpenAPI -----------------> semantic drift ------> safe report
API + OpenAPI + typed cases --------------> live conformance ----> safe report
safe report or fixed workflow failure ----> Actions notifier ----> GitHub issue
```

The drift and conformance paths are independent. Neither may edit the
specification, create a pull request, or publish a release.

## Major components

- `openapi/openapi.yaml` is the canonical multi-file entry point. Path and
  schema fragments below `openapi/` are generated, reviewed artifacts.
- `openapi/generated/openapi.bundle.yaml` is the portable release interface.
- `scripts/` contains the current Node.js extraction, validation, and probe
  implementation. It remains authoritative until each command is replaced and
  verified by the Go migration.
- `.github/workflows/verify.yml` is the credential-free repository gate.
  Release automation consumes that gate before publishing the bundle.
- `docs/plans/` separates the Go migration, public-guide drift detection, and
  credentialed live conformance into reviewable work streams.

The target tooling is one repository-internal Go module with one CLI and a small
set of deep internal packages: guide acquisition and normalization, the sole
OpenAPI boundary, live execution, and sanitized reporting. Workflow YAML
orchestrates commands and permission boundaries; domain parsing, comparison,
validation, sanitization, and policy belong in tested Go code. The isolated
notifier owns only boundary validation and fixed issue rendering.

The accepted dependency boundary is recorded in
[ADR 0001](docs/decisions/0001-go-repository-tooling.md). In particular,
`libopenapi` types stay inside the specification package, typed Go cases own
live-only assertions, and the final repository-owned CLI toolchain is Go-only.

## Dominant flows

### Specification refresh

The acquisition layer fetches only trusted guide pages and converts them into
a normalized catalog. The specification layer renders a complete staging tree,
validates ownership, references, catalog invariants, and OpenAPI semantics, and
then atomically publishes the generated fragments. Bundling is derived from the
committed multi-file description. A mandatory compatibility gate protects this
flow from OpenAPI-library rendering, reference, or comparison regressions.

### Scheduled observations

The credential-free drift job regenerates a candidate with baseline snapshot
metadata and compares specification meaning. Page markup or formatting that
normalizes to the same OpenAPI contract is unchanged.

The credentialed conformance job executes every physical OpenAPI operation
using committed public inputs and narrowly scoped discovery where stable input
is impractical. OpenAPI owns enumeration, wire serialization, and structural
validation; typed cases own endpoint-specific meaning. The job produces an
allowlisted report and discards raw responses.

Separate notification jobs receive either validated reports or a fixed,
allowlisted workflow-failure envelope synthesized from trusted Actions metadata
when a producer cannot emit a valid report. They create or update the
independent drift and conformance issues. A recovered check comments once on its
existing issue but never closes it.

## Invariants

- OpenAPI 3.2 is canonical; compatibility artifacts must not weaken it.
- Generated OpenAPI files are changed through the generator, never by hand.
- Guide-sourced facts, empirical constraints, and executable test scenarios
  remain separate and traceable.
- A refresh publishes a complete validated catalog or leaves the prior catalog
  untouched.
- External OpenAPI library types do not cross the repository's specification
  boundary.
- Offline verification performs no OpenDART network requests and needs no API
  key.
- The API key is available only to trusted default-branch conformance code and
  never to issue-writing jobs.
- Reports and issues contain only allowlisted evidence. They never contain the
  key, authenticated URLs, or unrestricted response bodies.
- Missing, invalid, or conclusion-inconsistent observation reports produce a
  fixed workflow-failure notice; producer logs and arbitrary error text never
  become notification input.
- Raw live response bodies are never persisted as fixtures or artifacts.
- Scheduled automation reports evidence only. Specification changes always
  require a reviewed human-authored change.
- The completed migration has one Go implementation and no repository-owned
  Node.js CLI, package, or Redocly dependency. Pinned third-party Actions may
  use their bundled runtime.

## Start here

- [`README.md`](README.md) describes the artifacts and current commands.
- [`docs/decisions/0001-go-repository-tooling.md`](docs/decisions/0001-go-repository-tooling.md)
  records the language and tooling decision.
- [`docs/plans/go-tooling-migration.md`](docs/plans/go-tooling-migration.md)
  owns the Node-to-Go migration.
- [`docs/plans/guide-drift.md`](docs/plans/guide-drift.md)
  owns public-guide semantic drift detection.
- [`docs/plans/live-conformance.md`](docs/plans/live-conformance.md) owns
  weekly live endpoint conformance.
