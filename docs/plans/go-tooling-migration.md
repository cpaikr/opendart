# Go Tooling Migration Plan

## Objective

Replace the large repository-owned Node.js scripts with one repository-internal
Go module and CLI while preserving the generated OpenAPI contract semantically.
The migration may land one reviewed formatting-only bundle cutover; that output
then becomes the deterministic golden. The repository continues to publish
specifications, not an application or Go package.

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

Use module path `github.com/cpaikr/opendart` and one CLI, provisionally
`opendart-spec`, with task-oriented commands such as `sync`, `verify`, `drift`,
and `live`. The exact flags are designed with the first implementation slice;
command names are not a public compatibility surface.

Use `flag.FlagSet` for subcommands and the standard library for networking,
logging, encoders, archive handling, filesystem primitives, and tests. Use
`goquery` plus `golang.org/x/net/html/charset` for guide pages. Do not add an
application, CLI, configuration, crawler, HTTP, retry, logging, test-suite, or
task framework without evidence that the standard-library design has become
the source of durable complexity.

One reusable `net/http.Client` enforces contexts, explicit timeouts, bounded
bodies, trusted origins, and a small injectable request interface. Requests are
sequential initially. There is no hidden retry or generic rate-limiting layer;
introduce measured concurrency or rate control only through a reviewed policy.

`log/slog` writes diagnostics to stderr. Commands write a versioned
`report.json` separately. Process outcomes distinguish clean completion, an
observed finding, and execution or configuration failure. Configuration uses
explicit flags and `OPENDART_API_KEY` only; do not add Viper or a
configuration-file schema.

Keep the initial package map small:

- `cmd/opendart-spec` owns process-level arguments, exit codes, and dependency
  wiring. It contains no extraction or validation rules.
- `internal/guide` owns trusted URL acquisition, HTML interpretation, source
  diagnostics, and normalization into one source catalog.
- `internal/spec` owns OpenAPI generation, loading, references, validation,
  bundling, semantic comparison, and atomic publication. It is the only package
  that imports `libopenapi`; library types do not escape its package API.
- `internal/live` owns physical-operation enumeration, committed test cases,
  representation decoders, content assertions, request budgets, and typed
  conformance findings.
- `internal/report` owns versioned report schemas and allowlist sanitization. It
  has no GitHub mutation or Markdown-rendering client.

Prefer deeper packages over a package for every processing step. Introduce a
new seam only when it owns a durable policy or isolates an external system.

## Library compatibility gate

Before porting the generator, build a committed compatibility harness against
the current specification and minimal representative fixtures. It must prove:

- multi-file OpenAPI 3.2 loading and local `$ref` resolution;
- preservation of `x-opendart`, XML metadata including `nodeType`, path order,
  schema names, and all source descriptions through render and bundle;
- a no-op semantic comparison reports unchanged while representative
  additions, removals, parameter changes, schema changes, and operation changes
  identify the affected contract;
- request and response validation for representative JSON success, XML
  success, and XML API-error responses;
- bounded ZIP inspection for a valid archive, contained XML, malformed data,
  unsafe entry paths, excessive entries, and oversized decompression;
- deterministic rendering suitable for a reviewed golden and stable freshness
  gate;
- goquery extraction parity for representative guide tables, nested response
  rows, contradictions, and source metadata; and
- Vacuum mapping for each current Redocly rule, including the
  repository-specific Go checks needed for equal or stronger coverage.

Core loading, reference, meaning, and semantic-comparison failures disqualify
the selected OpenAPI engine. An isolated renderer or validator gap may use a
narrow adapter or replacement component if coverage remains equal or stronger.
Do not spread library-specific types through domain packages.

The initial dependency candidates are:

- `pb33f/libopenapi` for loading, rendering, bundling, and semantic comparison;
- `pb33f/libopenapi-validator` for structural request and response validation
  if the representative wire fixtures pass;
- `goquery` and `x/net/html/charset` for guide HTML; and
- `go-cmp` only as a test dependency when standard comparisons are insufficient.

Overlay, Arazzo, `openapi-changes`, and general CLI/configuration/HTTP/test
frameworks are not part of the first implementation. Vacuum remains a
compatibility-gated lint tool rather than a second OpenAPI correctness oracle.

## Migration slices

1. Add the Go module, CLI skeleton, compatibility harness, and golden fixtures.
   Record the selected libraries and any narrow adapters after the gate passes.
2. Port trusted acquisition, source normalization, generation, and atomic
   publication behind tests. Compare staged generated fragments with the
   accepted artifacts, but keep the production `sync` command on Node while its
   validators and bundler are still authoritative.
3. Port catalog verification, OpenAPI validation, bundling, bundle freshness,
   and the existing workflow/release-configuration guards. Keep Redocly during
   comparison, then replace it with Vacuum plus tested Go checks only after
   equal or stronger coverage is demonstrated. Compare one complete refresh,
   switch `sync` once, and, if bundle formatting differs, review the semantic
   no-op bundle cutover and establish the replacement golden at that point.
4. Port the focused probe primitives into the general live runner, then build
   the complete physical-operation case inventory described in the live plan.
5. Switch GitHub verification and documented commands to Go, add the grouped
   Dependabot updates described below, update Release Please's repository-only
   path exclusions, and remove the Node.js scripts, `package.json`, lockfile,
   and Redocly after replacement checks demonstrate equal or stronger coverage.

Each command cuts over once. Temporary cross-language comparison is test
scaffolding, not a supported compatibility layer.

## Testing strategy

- Pure unit tests cover normalization, policies, serialization, comparison,
  budgets, sanitization, and representation-specific assertions.
- Recorded minimal HTML and API fixtures cover source and wire edge cases
  without network access or credentials.
- Plain `testdata/*.golden` files compare generated fragments and the bundle
  against reviewed artifacts with useful semantic diagnostics. A local update
  flag is explicit; CI never rewrites goldens.
- Filesystem integration tests use temporary trees to prove complete
  publication, rollback on failure, ownership checks, and bundle freshness.
- `httptest` covers request serialization and HTTP boundaries. Use
  `testing/synctest` where virtual time materially simplifies timeout behavior.
- CI checks `gofmt`, `go vet ./...`, `go mod verify`, and
  `go test -race ./...`. Do not add a lint aggregator initially; add a
  separately pinned vulnerability or static-analysis tool only when it
  contributes a check not already owned by the compiler, vet, or tests.
- A complete live guide refresh is a deliberate integration check, never part
  of pull-request verification.
- Credentialed endpoint tests remain in their separate protected workflow.
- Live response bodies are never snapshots or test fixtures.
- Workflow fixtures cover valid reports, missing, oversized, and invalid
  reports, producer conclusion mismatches, and the fixed workflow-failure
  envelope. Notification rendering never consumes producer logs or arbitrary
  error text.

## Dependency maintenance

Pin the Go toolchain, module graph, lint tool, and Actions by their native lock
or immutable reference. Dependabot opens grouped dependency and Actions pull
requests approximately monthly; urgent security updates may arrive sooner.
OpenAPI library updates must run the complete compatibility harness, not only
ordinary unit tests.

## Acceptance criteria

- One Go CLI owns all substantial repository tooling.
- Generated artifacts retain the accepted OpenAPI meaning and provenance.
- After the reviewed formatting cutover, repeated generation is deterministic
  and byte-for-byte freshness is enforced against the new golden.
- Offline verification is deterministic and performs no OpenDART requests.
- Errors identify the operation, source page or artifact, processing phase, and
  safe cause without exposing response bodies or credentials.
- No repository-owned Node.js CLI or package dependency, Redocly dependency,
  duplicated generator, or public Go API remains. Pinned third-party Actions
  may use their bundled runtime.
- The drift and live plans can reuse the same OpenAPI and reporting foundations
  without sharing credentials or issue state.

## Next action

Implement the library compatibility gate and CLI skeleton only. Do not port the
generator until the current specification, semantic diff, guide-extraction,
lint-parity, and JSON/XML/ZIP fixtures pass that gate.
