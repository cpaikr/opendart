# Credentialed live conformance progress

This current-state handoff tracks implementation of
[Credentialed live conformance](live-conformance.md). Update it in place; do
not append session transcripts.

## Current slice

- Integration branch: `sub`; ordered work 5 is merged.
- Completed scope: the manual protected producer, isolated notifier, and
  offline enforcement of credential, artifact, permission, deduplication,
  recovery, and sanitization boundaries.
- Stop before environment or credential configuration, workflow dispatch, any
  real OpenDART request, weekly scheduling, or guide drift.

## Decisions

- Deliver three sequential, non-stacked PRs matching ordered work 3, 4, and 5.
  Merge each into `sub` before creating its successor branch.
- Derive the physical GET matrix and representation split from the canonical
  OpenAPI document. An XML API-error response on a ZIP route is an alternate
  response, not another physical operation.
- Extend `internal/openapi` only through repository-owned types. Keep pinned
  libopenapi types private and preserve the Rust worktree's ownership boundary.
- Rust currently has no changes in the shared seams. After this slice lands,
  Rust must rebase the landed `internal/openapi` projection before its ordered
  work 3; later live changes must extend rather than fork that projection.
- Require all preflight validation to complete before the credential is read or
  any request is attempted. The derived ceiling includes primary attempts and
  declared discovery maxima; the runner performs one attempt per request.
- Keep reports versioned and allowlisted. Raw bodies, authenticated URLs,
  credentials, arbitrary response headers, and arbitrary error text are never
  report fields or notifier inputs.
- Prefer stable committed historical inputs. If the inventory proves a receipt
  number cannot remain durable, allow only an explicit, budgeted, reusable
  discovery dependency with a distinct failure classification.

## Completed

- Read the roadmap, task, completed Go migration plans, repository architecture,
  current Go seams, and the Rust worktree boundary.
- Published `sub` from the current local `dev` state and created the first
  feature branch from it.
- Established the canonical inventory counts and representation split without
  changing generated OpenAPI artifacts.
- Merged ordered work 3 through PR #18 into `sub` with individual commits
  preserved and all review threads resolved, then based this slice on that
  integration result.
- Added the deterministic repository-owned physical-operation projection,
  trusted server and query-auth metadata, parameter serialization metadata, and
  offline OpenAPI request validation. Canonical tests prove the complete
  representation split and distinguish ZIP success from its XML error outcome.
- Added the fail-closed runner foundation: exact coverage and trust preflight,
  derived hard request ceiling, one-attempt execution, bounded JSON/XML/ZIP
  adapters, typed assertion registry, fixed sanitized failures, versioned
  allowlisted report, and strict bounded report decoding for the future
  notifier.
- Focused race-enabled validation passes for `internal/openapi`,
  `internal/liveconformance`, and `internal/liveprobe`; representative fake HTTP
  coverage exercises JSON, XML, ZIP, pacing, body bounds, no retries, alternate
  ZIP error routing, and report sanitization.
- Completed the implementation review gate. Applied operation-level OpenAPI
  parameter precedence, nil-definition hardening, normalized-path rejection,
  identical producer/notifier allowlists, unsupported-media sanitization, and
  failed-report round-trip coverage. Kept the single sequential execution path,
  bounded representation adapters, and narrow repository-owned projection.
- Addressed all Codex and CodeRabbit findings on the runner PR, including
  complete success evidence, canonical failure identities, bounded semantic ZIP
  evidence, deterministic XML paths, body lifecycle handling, and identical
  producer/notifier size limits. The independent follow-up review is clean.
- Added deterministic reviewed cases for every physical operation. Structured
  JSON/XML pairs share one logical definition and semantic policy; the archive
  operations remain explicit. Stable Samsung, reporting-period, taxonomy, and
  receipt coordinates are committed in typed code rather than OpenAPI metadata.
- Added one fixed rare-disclosure discovery batch whose declared historical
  partitions, detail types, pages, aliases, and maximum are preflighted. Results
  are reused across paired primary cases; empty partitions and unused pages do
  not spend the ceiling, open pagination fails closed, and unresolved or invalid
  coordinates use distinct allowlisted discovery failures.
- Wired `live-conformance --preflight-only` and the full manual command to the
  reviewed registry. Repository verification now runs the credential-free
  coverage, request, budget, pagination, and report-identity gate.
- Matched committed archive evidence without weakening structured routing:
  positive ZIP signatures may normalize the observed download media to the
  canonical ZIP representation, and bounded archive XML supports reviewed
  EUC-KR/CP949 labels or bytes. XML API-error envelopes remain alternate ZIP
  responses and cannot pass a primary case.
- Completed the ordered-work-4 review corrections: discovered assertions bind
  the exact resolved corporation and receipt within one record, DS006
  assertions follow its nested `group/list` structure in JSON and XML, index
  assertions use fields their schemas expose, corrected filings cannot supply
  initial-receipt coordinates, and short discovery partitions require unique
  contiguous declarations plus consistent response pagination.
- Baseline validation passed: `go vet ./...`, `go test -race ./...`,
  `go run ./cmd/opendart-tool verify --repository-root .`, and
  `git diff --check`.
- Merged ordered work 4 through PR #21 into `sub` with individual commits
  preserved and all review threads resolved, then based this slice on that
  integration result.
- Added the manual-only trusted-`main` producer with read-only repository
  access, protected-environment declaration, a credential-free preflight, API
  key exposure only at the request step, and a fixed sanitized report artifact.
- Added a separate default-branch `workflow_run` notifier with minimal artifact,
  repository, and issue permissions. It has no protected environment or
  OpenDART credential and checks out the exact trusted producer revision.
- Added strict bounded report consumption, fixed workflow-failure fallback,
  one-issue deduplication, update-in-place failures, one-time recovery
  recording, and issue-state preservation. Tests use fake stores and local HTTP
  servers only; no GitHub issue write or OpenDART request was made.
- Extended offline repository verification to fail closed on workflow trigger,
  ref, environment, permission, action pin, checkout, artifact, credential, and
  notifier-input drift.
- Completed the required independent security review. Its fixed-fallback
  finding expanded the allowlisted producer conclusions to every terminal
  Actions result, including timeout, startup, stale, neutral, and
  action-required outcomes; each now produces the fixed notification rather
  than exiting before issue handling.
- Addressed external notifier review with a fail-closed multiple-issue test,
  safe HTTP transport cloning, explicit ignored-close handling, and bounded
  exact-title GitHub issue search. One newest-issues page is merged with search
  results to cover indexing delay without requiring repository label setup;
  unrelated issue bodies are discarded before candidate validation and the
  bounded response size accommodates the selected GitHub page size.
- Merged ordered work 5 through PR #22 into `sub` with individual commits
  preserved, all review threads resolved, and the final Verify run passing.
- Ordered-work-5 validation passes: `go vet ./...`, `go test -race ./...`,
  `go run ./cmd/opendart-tool verify --repository-root .`, and
  `git diff --check`.

## Blockers

None. The protected GitHub environment and credential are intentionally absent,
not implementation blockers.

## Next action

None within this task. Stop before protected environment or credential setup,
the first workflow dispatch or live OpenDART request, weekly scheduling,
ordered work 6, or guide-drift implementation.
