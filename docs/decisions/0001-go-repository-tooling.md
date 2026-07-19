# ADR 0001: Use Go for repository tooling

- Status: accepted
- Date: 2026-07-17

## Context

The current Node.js and Redocly toolchain owns guide acquisition, HTML parsing,
normalization, OpenAPI generation, filesystem publication, validation, bundling,
focused live requests, and reporting. Adding semantic guide-drift detection and
complete live conformance would enlarge that toolchain and duplicate more
OpenAPI behavior.

The repository's product is its OpenAPI specification and portable release
bundle. Its tooling is private repository infrastructure, so the language choice
should optimize OpenAPI fidelity, robustness, and long-term maintainability
rather than public package ergonomics or preservation of the current
implementation.

## Decision

Migrate repository-owned tooling to Go as one repository-internal module and one
CLI. Command entry points remain thin; parsing, normalization, specification
policy, validation, and live-test behavior live behind internal boundaries. The
module path is `github.com/cpaikr/opendart`, and no Go package is a supported
consumer API.

Use the standard library at operational boundaries unless a dependency removes
durable complexity the repository should not own. Do not introduce application,
web, dependency-injection, configuration, or task frameworks without concrete
need. Toolchain and dependency pins become the source of truth when the Go
module and workflow are committed.

The specification boundary owns OpenAPI loading, references, rendering,
bundling, validation, and comparison. External OpenAPI library types do not
cross that boundary. The compatibility gate selected
`github.com/pb33f/libopenapi` for OpenAPI 3.2 loading, modeling, rendering,
bundling, and change detection, with `github.com/pb33f/libopenapi-validator`
for document and representative JSON and XML response validation. Their types
remain confined to `internal/openapi`.

The extraction fixture selected `golang.org/x/net/html` for standards-based
HTML parsing. It remains confined to `internal/guide`; acquisition, guide
policy, and normalized repository types do not depend on parser nodes.

The repository boundary adds only the gaps proven by the gate: confined,
offline local-reference checks; comparison normalization for reference layout
and YAML scalar presentation; a representative response-description lint check
that document-schema validation does not enforce; and ZIP archive validation
because OpenAPI schema validation does not inspect archive contents.

Within OpenAPI, guide-supported facts remain distinguishable from explicitly
labeled empirical observations. Live-test scenarios remain separate typed
repository policy and never modify the released contract implicitly.

The future Go CLI does not mutate GitHub issues. Scheduled observation workflows
keep read-only production work separate from any narrowly privileged
notification step. Credential, report, and notification details belong to the
corresponding implementation plans.

## Compatibility gate

Before the port expands, the migration must prove that the selected components
preserve the repository's multi-file OpenAPI 3.2 meaning, local references,
extensions, XML metadata, bundle semantics, and required validation coverage.
The gate must also establish deterministic output and meaningful semantic
comparison. The selected components passed the repository's multi-file OpenAPI
3.2 and accepted-bundle fixtures. A future failure in a core requirement
requires replacement; an isolated capability gap may use a narrow adapter only
when coverage remains equal or stronger.

Node.js and Redocly remain authoritative for each command until its Go
replacement passes parity and cuts over. The migration may make one reviewed
formatting-only bundle cutover after semantic equivalence is demonstrated; that
output then becomes the new byte-for-byte freshness baseline. CI never updates
the baseline itself.

## Consequences

- Migration is incremental, but each command has one final implementation.
  Temporary cross-language comparison is not a permanent compatibility layer.
- Node.js scripts, package dependencies, and Redocly are removed after the Go
  replacements demonstrate equal or stronger coverage.
- GitHub Actions remains workflow YAML, while substantial parsing, policy,
  comparison, validation, and sanitization stay in tested repository code.
- Dependencies remain isolated behind internal boundaries and are upgraded only
  with the compatibility evidence appropriate to the behavior they own.
- The released OpenAPI contract and release process do not become a public Go
  tooling API.

## Alternatives considered

- Extending the current Node.js scripts would avoid a migration but retain the
  growing concentration of extraction, OpenAPI, filesystem, and live-test
  responsibilities.
- Python, TypeScript, Rust, and Elixir were considered. None offered enough
  repository-specific advantage to justify owning more OpenAPI 3.2 semantics or
  introducing a broader framework model.

## Related work

- [Repository architecture](../../ARCHITECTURE.md)
- [Go tooling migration](../../plans/go-tooling-migration.md)
- [Public-guide drift detection](../../tasks/guide-drift.md)
- [Credentialed live conformance](../../tasks/live-conformance.md)
