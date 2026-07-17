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

Start on the current stable Go toolchain (Go 1.26 at decision time), with the
module and workflow pins as the durable source of truth. Use module path
`github.com/cpaikr/opendart`; all implementation packages remain under
`internal/`.

Do not introduce an application, web, dependency-injection, configuration, or
task framework. One binary with flat subcommands uses `flag.FlagSet`. Explicit
flags and the `OPENDART_API_KEY` environment variable are sufficient; there is
no configuration-file layer.

Prefer the standard library at operational boundaries:

- `net/http` with one reusable client, contexts, explicit timeouts, bounded
  bodies, and a small injectable request interface;
- `log/slog` to stderr, separate from deterministic machine reports;
- `encoding/json`, `encoding/xml`, and `archive/zip` with input, entry,
  expansion, and path-safety limits; and
- `testing`, `httptest`, and ordinary `testdata/` goldens. Use
  [`go-cmp`](https://pkg.go.dev/github.com/google/go-cmp/cmp) only where
  structural test diagnostics justify it.

Use [`goquery`](https://github.com/PuerkitoBio/goquery) for the guide's
selector-heavy HTML traversal and `golang.org/x/net/html/charset` before parsing
non-UTF-8 responses. A crawler framework is unnecessary because acquisition
policy belongs in repository code.

Use OpenAPI-native components where they prevent the repository from
reimplementing specification semantics:

- Adopt [`pb33f/libopenapi`](https://pb33f.io/libopenapi/) behind the
  `internal/spec` boundary for OpenAPI 3.2 loading, references, rendering,
  bundling, and semantic comparison, subject to the mandatory compatibility
  gate.
- Adopt
  [`libopenapi-validator`](https://github.com/pb33f/libopenapi-validator)
  behind a narrow validation boundary if the gate proves the required
  request/response, JSON, XML, and archive behavior.
- Compare all semantic changes for guide drift, not only changes classified as
  breaking. Do not add a second reporting wrapper around the same diff engine.
- Keep Redocly only as a migration oracle. The completed repository-owned CLI
  toolchain is Go-only: [`vacuum`](https://quobix.com/vacuum/) may replace
  compatible lint rules, and any missing repository-specific checks remain
  tested Go policy. Removal may not weaken validation.

The compatibility spike in the migration plan is a gate, not optional research.
It must prove the selected libraries preserve this repository's multi-file
OpenAPI 3.2 contract, extensions, XML metadata, bundle semantics, and required
JSON/XML/ZIP validation behavior before the port expands.

## Standards boundary

- OpenAPI contains facts supported by the official guide and drives physical
  operation enumeration, parameter serialization, routing, and structural
  response validation.
- Typed Go cases supply committed public inputs and stable semantic assertions
  that OpenAPI does not express. Start with typed constructors rather than a
  YAML assertion DSL; move only genuinely volatile data values to a strict data
  file if observed maintenance requires it.
- Defer Overlay and Arazzo. Neither belongs in the first implementation while
  OpenDART's XML, ZIP, discovery, and endpoint-specific assertions still
  require a parallel typed execution model. Reconsider either only if one
  current standards-driven path can replace, rather than duplicate, that model.

Test-only constraints never modify the released bundle.

## Reports and automation boundary

Commands emit versioned, schema-validated JSON reports. Logs go to stderr, and
process outcomes distinguish a clean run, an observed finding, and an execution
or configuration error.

Go code does not mutate GitHub issues. A separate minimally privileged Actions
job runs after each scheduled-observation producer even when that producer
fails. Under a strict byte ceiling, it independently validates the allowlisted
report, checks that the report outcome and producer conclusion agree under the
documented exit mapping, renders fixed Markdown from those fields, and uses
pinned
[`actions/github-script`](https://github.com/actions/github-script) for issue
creation, updates, and recovery comments. It never receives the OpenDART key,
pre-rendered unrestricted text, or response bodies and does not execute the
secret-bearing job's code. Pinned third-party Actions may use their bundled
runtime; they do not create a repository-owned Node.js toolchain dependency.

Report retrieval is non-fatal so absence or artifact-service failure reaches the
fixed fallback. If the report is unavailable, oversized, invalid, or inconsistent
with the producer conclusion, the notification job discards its contents and
synthesizes a fixed workflow-failure envelope. That envelope contains only a
version, an enumerated failure classification, the producer conclusion, commit
identity, and workflow run identity/link taken from trusted GitHub contexts. It
never incorporates producer logs, step output, exception messages, or other
arbitrary text. A platform failure that prevents the notification job itself
from running remains visible in GitHub Actions; repository automation cannot
reliably turn that failure into an issue.

Raw live responses are bounded in memory and discarded after validation.
Persisted reports contain only allowlisted identities, statuses, sizes, hashes,
schema locations, assertion IDs, and selected safe evidence.

## Compatibility and maintenance policy

OpenAPI loading, local references, document meaning, and semantic comparison
are hard requirements. A failure in one isolated renderer or validator
capability may be handled by a narrow adapter or replacement component, but
the migration never accepts reduced coverage and external library types do not
spread across domain packages.

The migration may make one reviewed formatting-only bundle cutover after
semantic equivalence is proven. That output becomes the new golden; CI never
updates goldens, and subsequent freshness checks are deterministic.

Dependabot groups dependency and pinned-action updates into approximately
monthly pull requests, with urgent security updates allowed sooner. Every
OpenAPI dependency update must pass the complete compatibility fixture suite.

## Consequences

- Node.js scripts and dependencies are removed after Go command parity and an
  artifact-equivalence review; there is no permanent dual implementation or
  Redocly-only runtime.
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
