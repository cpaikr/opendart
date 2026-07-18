# Go Tooling Migration

## Objective

Replace the repository-owned Node.js tooling with one internal Go CLI while
preserving the released OpenAPI contract. The repository continues to publish a
specification and bundle, not an application or supported Go package. The
language decision is recorded in
[ADR 0001](../decisions/0001-go-repository-tooling.md).

## Current state

- Node.js remains authoritative for guide acquisition, generation, catalog
  checks, bundle creation and freshness, release-configuration checks, and the
  focused multi-company probe.
- The internal Go CLI and OpenAPI boundary are committed as an additive
  compatibility gate. Representative extraction, lint, JSON, XML, and ZIP
  fixtures plus the accepted multi-file OpenAPI 3.2 contract and Redocly bundle
  pass without OpenDART credentials.
- Pull-request and manual verification run the Node.js and Redocly baseline plus
  the Go compatibility gate. No repository command has cut over yet.
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
2. Port trusted guide acquisition, normalization, generation, staged
   validation, owned-output safeguards, and failure rollback. Compare a
   complete staged refresh with the accepted artifacts before switching
   synchronization.
3. Port catalog and reference checks, lint coverage, bundling, freshness, and
   workflow and release guards. Switch local documentation and credential-free
   CI after parity is demonstrated.
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

Port trusted guide acquisition and normalization into the internal Go CLI,
then add deterministic generation, staged validation, owned-output safeguards,
and rollback. Keep Node/Redocly authoritative until a complete staged refresh
matches the accepted artifacts semantically and the synchronization command
cuts over once.

## Progress log

- 2026-07-18: Completed ordered work 1. Selected the pinned libopenapi and
  libopenapi-validator components behind `internal/openapi` and x/net/html
  behind `internal/guide`; passed representative extraction, lint, JSON, XML,
  and ZIP fixtures; added the proven narrow adapters; and established
  deterministic rendering and bundling with zero semantic changes against the
  accepted Redocly bundle. The existing Node/Redocly gate remains authoritative
  while the additive Go compatibility gate runs in credential-free verification.
