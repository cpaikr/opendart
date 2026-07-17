# ADR 0001: Use Go for repository tooling

- Status: accepted
- Date: 2026-07-17

## Context

The current Node.js scripts combine source acquisition, HTML parsing,
normalization, OpenAPI generation, filesystem publication, validation, live
requests, and reporting. Planned semantic drift detection and complete live
endpoint conformance would make those scripts larger and would duplicate more
OpenAPI behavior.

The repository's product is its OpenAPI specification and release bundle. The
tooling is private repository infrastructure, so language selection should
optimize OpenAPI fidelity, robustness, and maintainability rather than public
package ergonomics or preservation of the current implementation.

## Decision

Implement repository-owned tooling in Go as one repository-internal module and
one CLI. Keep command orchestration thin and place behavior in `internal/`
packages with clear ownership of guide acquisition, normalized catalog
generation, OpenAPI operations, and live conformance.

Do not introduce an application or web framework. Start with Go's standard
library for CLI dispatch, HTTP, JSON, XML, ZIP, hashing, filesystem work, and
tests, plus `golang.org/x/net/html` for guide parsing. Add a CLI framework only
if the implemented command surface demonstrates a concrete need.

Use OpenAPI-native components where they prevent the repository from
reimplementing OpenAPI semantics:

- Evaluate [`pb33f/libopenapi`](https://pb33f.io/libopenapi/) for OpenAPI 3.2
  loading, references, rendering, bundling, semantic change detection, and
  Overlay/Arazzo models.
- Evaluate
  [`libopenapi-validator`](https://github.com/pb33f/libopenapi-validator) for
  request and response validation.
- Keep standard-library HTTP, JSON, XML, ZIP, hashing, and process primitives
  at the representation and security boundaries.
- Keep Redocly as an independent transitional validator until Go validation
  and bundling demonstrate parity. It does not influence the implementation
  language. Evaluate [`vacuum`](https://quobix.com/vacuum/) separately if
  replacing Redocly removes durable Node.js ownership without weakening checks.

The compatibility spike in the migration plan is a gate, not optional research.
It must prove the selected libraries preserve this repository's multi-file
OpenAPI 3.2 contract, extensions, XML metadata, bundle semantics, and required
JSON/XML/ZIP validation behavior before the port expands.

## Standards boundary

- OpenAPI contains facts supported by the official guide.
- An Overlay may hold reviewed empirical constraints used only by live tests.
- Arazzo is the preferred representation for committed inputs and success
  criteria when the current Go stack supports the required version and XML
  expressions faithfully.
- Repository-owned typed cases are allowed only for behavior that the standards
  cannot express or the selected implementation cannot execute correctly.

Test-only constraints never modify the released bundle.

## Consequences

- Node.js scripts and dependencies are removed after Go command parity and an
  artifact-equivalence review; there is no permanent dual implementation.
- GitHub Actions remains YAML, but substantial parsing, policy, comparison, and
  sanitization logic does not live in workflow shell fragments.
- Go packages under `internal/` are not a supported consumer API.
- Library versions are pinned and upgrades are verified against repository
  fixtures because the relevant OpenAPI 3.2 ecosystem is still evolving.

## Alternatives considered

- Python has strong schema-driven and generative API testing, but provides a
  less cohesive native core for this repository's combined OpenAPI 3.2
  loading, rendering, bundling, diffing, and generation needs.
- TypeScript with Effect would provide strong orchestration, but Effect v4 and
  several relevant modules are still pre-stable and do not add OpenAPI 3.2
  leverage.
- Rust and Elixir have attractive runtime or type-system properties, but their
  current OpenAPI libraries do not provide the same complete 3.2-oriented
  foundation. Choosing either would make the repository own more OpenAPI
  semantics.

## Related work

- [Go tooling migration](../plans/go-tooling-migration.md)
- [Public-guide drift detection](../plans/guide-drift.md)
- [Credentialed live conformance](../plans/live-conformance.md)
