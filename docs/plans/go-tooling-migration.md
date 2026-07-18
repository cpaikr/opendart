# Go Tooling Migration

## Objective

Replace the repository-owned Node.js tooling with one internal Go CLI while
preserving the released OpenAPI contract. The repository continues to publish a
specification and bundle, not an application or supported Go package. The
language decision is recorded in
[ADR 0001](../decisions/0001-go-repository-tooling.md).

## Current state

- The Go CLI is authoritative for guide acquisition, normalization, generation,
  staged validation orchestration, guarded publication, rollback, catalog and
  reference checks, strict linting, deterministic bundling and freshness, and
  release/workflow guards. Canonical synchronization and offline verification
  have cut over once.
- Node.js remains authoritative only for the focused multi-company probe and
  its offline tests. The superseded Node and Redocly verification paths remain
  dormant until final removal and do not run in the current verification gate.
- Pull-request, manual, and release verification use the credential-free Go
  repository verifier. During cutover, the accepted multi-file OpenAPI 3.2
  contract and Go bundle also passed the former Node catalog and Redocly lint
  checks plus semantic parity checks without OpenDART credentials. Those
  non-authoritative checks are retained as migration evidence, not automation;
  the former Redocly byte-freshness check rejects the approved Go formatting.
- [Guide drift](guide-drift.md) and [live conformance](live-conformance.md) are
  committed follow-on work. They depend on the Go OpenAPI and reporting
  foundations but retain separate network, credential, and issue boundaries.

## Constraints

- Use module path `github.com/cpaikr/opendart`, one thin CLI, and internal
  packages with one boundary around third-party OpenAPI types.
- Cut over each command once. Cross-language comparison is temporary test
  scaffolding, not a supported compatibility layer.
- Before porting generation, prove that the selected OpenAPI components handle
  the repository's multi-file OpenAPI 3.2 documents, local references,
  `x-opendart` extensions, XML metadata, deterministic rendering and bundling,
  semantic comparison, and representative JSON, XML, and ZIP validation.
- Preserve or strengthen every current catalog, reference, lint, workflow, and
  release guard before removing its Node.js or Redocly implementation.
- Permit one reviewed formatting-only bundle cutover after semantic equivalence
  is demonstrated. That output becomes the byte-for-byte freshness baseline.
- Keep offline verification deterministic and free of OpenDART requests.
  Diagnostics must identify the affected operation, source, artifact, and phase
  without exposing credentials, authenticated URLs, or response bodies.

Both observation consumers use one shared automation contract: commands emit a
small versioned, allowlisted JSON report while diagnostics remain separate; a
minimally privileged notification job validates that report and otherwise uses
only a fixed failure envelope derived from trusted GitHub Actions metadata.
Producer logs and arbitrary error text are never notification input. Guide drift
and live conformance own independent deduplicated issues; recovery is recorded
once and automation never closes an issue.

## Ordered work

1. **Complete.** Add the Go module, CLI skeleton, compatibility harness, and
   representative fixtures. Record the chosen OpenAPI components and narrow
   adapters after the gate passes.
2. **Complete.** Port trusted guide acquisition, normalization, generation,
   staged validation, owned-output safeguards, and failure rollback. Compare a
   complete staged refresh with the accepted artifacts before switching
   synchronization.
3. **Complete.** Port catalog and reference checks, lint coverage, bundling,
   freshness, and workflow and release guards. Switch local documentation and
   credential-free CI after parity is demonstrated.
4. Add the shared report and HTTP safety boundaries needed by the drift and
   live plans. Replace the focused Node.js probe through the live-conformance
   work before removing it.
5. Remove repository-owned Node.js scripts and package metadata and retire
   Redocly after all current responsibilities have migrated. Add pinned Go and
   Actions dependency maintenance as part of the cutover.

## Acceptance criteria

- One internal Go CLI owns all substantial repository tooling.
- Generated artifacts retain the accepted OpenAPI meaning, provenance, and
  deterministic freshness behavior.
- Offline verification performs no OpenDART requests and needs no credential.
- Existing verification and release protections have equal or stronger
  coverage.
- No repository-owned Node.js CLI, package dependency, Redocly dependency,
  duplicate generator, or public Go API remains. Pinned third-party Actions may
  use their bundled runtimes.
- The drift and live plans reuse the OpenAPI and report foundations without
  sharing credentials or issue state.

## Next action

Begin ordered work 4 by adding the shared bounded report and HTTP safety
boundaries for drift and live conformance. This plan intentionally stops after
the completed verification cutover until that work is explicitly started.

## Progress log

- 2026-07-18: Completed ordered work 1. Selected the pinned libopenapi and
  libopenapi-validator components behind `internal/openapi` and x/net/html
  behind `internal/guide`; passed representative extraction, lint, JSON, XML,
  and ZIP fixtures; added the proven narrow adapters; and established
  deterministic rendering and bundling with zero semantic changes against the
  accepted Redocly bundle. The existing Node/Redocly gate remains authoritative
  while the additive Go compatibility gate runs in credential-free verification.
- 2026-07-18: Completed ordered work 2. Added exact-host credential-free guide
  acquisition, normalized source models, deterministic generation, staged
  catalog and Redocly validation, owned-output publication, and rollback. A
  complete live refresh matched the accepted OpenAPI source with zero semantic
  changes, and `sync:opendart` cut over once to the Go CLI.
- 2026-07-18: Completed ordered work 3. Ported catalog and confined-reference
  checks, strict linting, release/workflow guards, staged validation, composed
  bundling, exact freshness, and the offline repository verifier. The reviewed
  Go bundle has zero semantic changes from the accepted contract; the former
  Node catalog and Redocly lint checks passed as one-time cutover evidence,
  while the obsolete Redocly byte check is intentionally superseded. Local and
  credential-free CI commands now use only the Go verifier. Ordered work 4 and
  5 remain unstarted.
