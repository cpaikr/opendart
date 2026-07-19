# Credentialed live conformance progress

This current-state handoff tracks implementation of
[Credentialed live conformance](live-conformance.md). Update it in place; do
not append session transcripts.

## Current slice

- Branch: `feat/live-conformance-runner`, based on the published `sub`
  integration branch.
- Scope: ordered work 3 only—operation enumeration, request validation, the
  general runner, request budget, representation adapters, typed assertions,
  sanitized report, and offline HTTP tests.
- Stop before the complete primary-case inventory, workflow/notifier work,
  environment or credential configuration, any real OpenDART request, weekly
  scheduling, or guide drift.

## Decisions

- Deliver three sequential, non-stacked PRs matching ordered work 3, 4, and 5.
  Merge each into `sub` before creating its successor branch.
- Treat the canonical matrix as 167 physical GET operations: 82 JSON, 82 XML,
  and 3 ZIP success representations. An XML API-error response on a ZIP route
  is an alternate response, not another physical operation.
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
- Baseline validation passed: `go vet ./...`, `go test -race ./...`,
  `go run ./cmd/opendart-tool verify --repository-root .`, and
  `git diff --check`.

## Blockers

None.

## Next action

Address the runner-foundation PR review, merge it to `sub`, then begin ordered
work 4 on a fresh branch from the updated integration branch.
