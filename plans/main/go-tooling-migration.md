# Go Tooling Migration

## Outcome

Replace the repository-owned Node.js tooling with one internal Go CLI while
preserving the released OpenAPI contract. The repository continues to publish a
specification and bundle, not an application or supported Go package. The
language decision is recorded in
[ADR 0001](../../docs/decisions/0001-go-repository-tooling.md).

## Current state

- The Go CLI is authoritative for guide acquisition, normalization, generation,
  staged validation orchestration, guarded publication, rollback, catalog and
  reference checks, strict linting, deterministic bundling and freshness, and
  release/workflow guards. Canonical synchronization and offline verification
  have cut over once. The focused multi-company probe is implemented there with
  offline HTTP coverage and direct operational entry points.
- Pull-request, manual, and release verification use the credential-free Go
  toolchain directly. The accepted multi-file OpenAPI 3.2 contract and Go bundle
  passed the former Node and Redocly checks as one-time cutover evidence; those
  implementations and temporary compatibility scaffolding are now removed.
- [Guide drift](../../tasks/main/guide-drift.md) now has drift-safe
  acquisition, a bounded semantic report, a manual trusted-main workflow, an
  isolated notifier, and credential-free policy gates. Its supervised run and
  scheduling remain explicitly deferred.
  [Live conformance](../../tasks/main/live-conformance.md) has its general
  runner, reviewed case inventory, manual protected workflow, isolated
  notifier, and credential-free repository gates. Protected environment setup,
  credential configuration, supervised execution, and scheduling also remain
  explicitly deferred.
- The [Go-only cleanup](go-only-tooling-cleanup.md) and this migration are
  complete. Remaining supervised drift and live-conformance work remains
  intentionally separate and does not reopen the tooling migration.

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
- Use direct Go commands as the final local and CI interface. Do not replace npm
  scripts with another task runner.

Drift and live observation consumers must use one shared automation
contract: commands emit a small versioned, allowlisted JSON report while
diagnostics remain separate; a minimally privileged notification job validates
that report and otherwise uses only a fixed failure envelope derived from
trusted GitHub Actions metadata. Producer logs and arbitrary error text are
never notification input. Guide drift and live conformance own independent
deduplicated issues; recovery is recorded once and automation never closes an
issue. This is settled cross-plan policy, not an implemented shared module or a
tooling-migration completion prerequisite. The live-conformance and guide-drift
implementations now apply that contract independently without sharing
credentials or issue state.

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
4. **Complete.** The [Go-only cleanup](go-only-tooling-cleanup.md) added the
   focused probe behind a narrow Go interface, cut local and CI entry points
   over to direct Go commands, and removed repository-owned Node.js, npm,
   Redocly, and temporary compatibility surfaces.
5. **Complete.** The general live-conformance runner and credential-free guide
   drift command apply the versioned report contract through independent
   isolated notifiers without adding a public Go API.

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
- Direct Go commands are the sole documented command surface; the repository
  does not require a replacement task runner.
- The drift and live plans retain one shared automation contract without
  sharing credentials or issue state.

## Next action

None — complete.

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
  credential-free CI commands now use only the Go verifier.
- 2026-07-18: Completed ordered work 4 through the Go-only cleanup. Ported the
  focused probe, cut documentation and CI directly to Go, removed the Node/npm/
  Redocly dependency graph and duplicate implementations, and retired temporary
  compatibility adapters while retaining lasting Go coverage. Ordered work 5
  remained a follow-on owned by the drift and live-conformance plans.
- 2026-07-19: Completed ordered work 5 through the credentialed
  live-conformance implementation. Added the bounded report and HTTP seams,
  credential-free preflight, manual protected producer, and isolated notifier;
  environment setup, credentials, supervised execution, and scheduling remain
  outside the completed migration.
- 2026-07-20: Applied the same bounded-report and isolated-notifier contract to
  credential-free guide drift with manual trusted-main automation and offline
  policy checks. Supervised execution and scheduling remain in the drift task.
