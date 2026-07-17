# Go Tooling Migration Plan

## Objective

Replace the large repository-owned Node.js scripts with one repository-internal
Go module and CLI while preserving the generated OpenAPI contract byte-for-byte
where formatting is part of the reviewed artifact and semantically everywhere
else. The repository continues to publish specifications, not an application or
Go package.

The accepted language decision and its rationale are recorded in
[`../decisions/0001-go-repository-tooling.md`](../decisions/0001-go-repository-tooling.md).

## Current state

- Node.js currently owns guide acquisition, generation, catalog validation,
  bundle freshness, and the focused multi-company probe.
- Credential-free GitHub verification runs the Node.js toolchain.
- Go has been selected, but no Go module or migrated command is committed.
- Public-guide drift detection and complete live endpoint conformance are not
  implemented.

## Target shape

Use one CLI, provisionally `opendart-spec`, with task-oriented commands such as
`sync`, `verify`, `drift`, and `live`. The exact flags are designed with the
first implementation slice; command names are not a public compatibility
surface.

Use standard-library CLI dispatch, networking, encoders, archive handling,
filesystem primitives, and tests. Use `golang.org/x/net/html` for guide pages.
Do not choose an application framework; add a CLI library only if the concrete
interface outgrows the standard library during implementation.

Keep the initial package map small:

- `cmd/opendart-spec` owns process-level arguments, exit codes, and dependency
  wiring. It contains no extraction or validation rules.
- `internal/guide` owns trusted URL acquisition, HTML interpretation, source
  diagnostics, and normalization into one source catalog.
- `internal/spec` owns OpenAPI generation, loading, references, validation,
  bundling, semantic comparison, and atomic publication.
- `internal/live` owns physical-operation enumeration, committed test cases,
  representation decoders, content assertions, request budgets, and sanitized
  reports.

Prefer deeper packages over a package for every processing step. Introduce a
new seam only when it owns a durable policy or isolates an external system.

## Library compatibility gate

Before porting the generator, build a disposable but committed test harness
against the current specification and representative fixtures. It must prove:

- multi-file OpenAPI 3.2 loading and local `$ref` resolution;
- preservation of `x-opendart`, XML metadata including `nodeType`, path order,
  schema names, and all source descriptions through render and bundle;
- a no-op semantic comparison reports unchanged while representative parameter,
  schema, and operation mutations identify the affected contract;
- Overlay application does not alter the canonical source files or release
  bundle;
- the selected Arazzo support can represent and execute the required inputs and
  JSON/XML success criteria, or clearly identifies the small typed case layer
  still required;
- request and response validation for representative JSON and XML operations;
- bounded ZIP inspection for a valid archive, an XML API error, malformed data,
  and oversized decompression attempts; and
- deterministic output suitable for the existing freshness gate.

Pin dependencies only after this gate passes. If a library fails a requirement,
replace or wrap that capability deliberately; do not spread library-specific
types through the domain packages.

## Migration slices

1. Add the Go module, CLI skeleton, compatibility harness, and golden fixtures.
   Record the selected libraries and any narrow adapters after the gate passes.
2. Port trusted acquisition, source normalization, generation, staged
   validation, and atomic publication. Compare a complete refresh with the
   accepted artifacts before switching `sync`.
3. Port catalog verification, OpenAPI validation, bundling, bundle freshness,
   and the existing workflow/release-configuration guards. Keep Redocly as an
   independent oracle during this slice.
4. Port the focused probe primitives into the general live runner, then build
   the complete physical-operation case inventory described in the live plan.
5. Switch GitHub verification and documented commands to Go, update Release
   Please's repository-only path exclusions, and remove the Node.js scripts,
   `package.json`, lockfile, and Redocly only after replacement checks
   demonstrate equal or stronger coverage.

Each command cuts over once. Temporary cross-language comparison is test
scaffolding, not a supported compatibility layer.

## Testing strategy

- Pure unit tests cover normalization, policies, serialization, comparison,
  budgets, sanitization, and representation-specific assertions.
- Recorded minimal HTML and API fixtures cover source and wire edge cases
  without network access or credentials.
- Golden tests compare generated fragments and the bundle against reviewed
  artifacts with useful semantic diagnostics on failure.
- Filesystem integration tests use temporary trees to prove complete
  publication, rollback on failure, ownership checks, and bundle freshness.
- A complete live guide refresh is a deliberate integration check, never part
  of pull-request verification.
- Credentialed endpoint tests remain in their separate protected workflow.

## Acceptance criteria

- One Go CLI owns all substantial repository tooling.
- Generated artifacts retain the accepted OpenAPI meaning and provenance.
- Offline verification is deterministic and performs no OpenDART requests.
- Errors identify the operation, source page or artifact, processing phase, and
  safe cause without exposing response bodies or credentials.
- No permanent Node.js runtime, duplicated generator, or public Go API remains.
- The drift and live plans can reuse the same OpenAPI and reporting foundations
  without sharing credentials or issue state.

## Next action

Implement the library compatibility gate and CLI skeleton only. Do not port the
generator until the current specification, semantic diff, Overlay/Arazzo needs,
and JSON/XML/ZIP fixtures pass that gate.
