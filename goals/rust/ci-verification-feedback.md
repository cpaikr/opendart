# Goal: Shorten CI and local verification feedback

Status: active
Planning scope: rust

## Original contract

Goal contract
- Outcome: Make CI and local verification materially faster without weakening functional, generated-artifact, platform, compatibility, or concurrency coverage.
- Goal state: goals/rust/ci-verification-feedback.md
- Included results and sources (semantic results define scope; paths supply detail):
  - Parallel Go and Rust verification behind one failure-aware, stable aggregate `verify` result, with workflow-policy guards and release documentation kept truthful — plans/rust/ci-verification-feedback.md, .github/workflows/verify.yml, internal/releaseguard/check.go, RELEASING.md
  - Structurally faster Go tests by removing repeated canonical model and artifact construction while preserving command-level and end-to-end coverage — plans/rust/ci-verification-feedback.md, internal/sdkgen, internal/sdkgen/model, internal/verification
  - An audited verification portfolio: normal Go tests and targeted race coverage on every PR, a scheduled/manual full-race sweep, and repository-owned fast, pre-push, and exhaustive local entrypoints — plans/rust/ci-verification-feedback.md, README.md, sdk/rust/README.md
  - Measured post-change CI and local performance, plus an explicit evidence-backed decision on bounded Rust caching and conditional execution; implement caching only if measurements justify it — plans/rust/ci-verification-feedback.md
- Complete when: All included results are implemented; representative CI measurements demonstrate a materially shorter critical path; local tiers are documented and runnable; full functional and audited race coverage remains available; repository validation and review pass; and the selected delivery lifecycle is finished with truthful goal and planning state.
- Excluded: CLI artifact-transaction work and later roadmap items; credentialed live conformance; path-based exemptions before dependency ownership is machine-checkable; speculative caching; unrelated product behavior or broad release-system refactors.
- Authority: Only included results and the smallest bounded work necessary for their completion; record other work without executing it; expansion requires explicit user instruction.
- Resume: Invoke $progress in goal mode to initialize this exact contract before initial work and recover it before every resume, automatic continuation, compaction recovery, or handoff; fail closed if recovery fails.
- Delivery: PR delivery

## Authorized amendments

_None._

## Execution status

### Completed included results

- Parallel Go and Rust verification now runs behind one failure-aware, stable
  aggregate `verify` result. PR #47 merged as `fdabb18` with workflow-policy
  guards, truthful documentation, and unchanged verification commands.

### Current in-scope result

Structurally faster Go tests by removing repeated canonical model and artifact
construction while preserving command-level and end-to-end coverage.

### Next in-scope action

Push the bounded review follow-up, wait for PR #48's final checks, resolve its
review thread, and merge before changing the test portfolio.

### Evidence and blockers

- PR #46 merged as `750ba04`; the goal integration branch is based on that
  updated `rust` state.
- The local planning commit `90c0963` is preserved as equivalent rebased commit
  `2a88770` on the integration branch.
- Candidate: first workflow topology PR. Classification: included. Contract
  basis: the first included result and explicit first-PR delivery constraint.
  Action: proceed without changing test coverage.
- The work branch splits Go and Rust verification without changing any
  verification command and adds a `verify` fan-in over all four required work
  jobs. Releaseguard tests execute successful, failed, cancelled, and skipped
  dependency results.
- Local validation passed `go vet ./...`, `go test ./...`,
  `go test -race ./...`, targeted releaseguard and verification tests, and the
  repository verifier. The full race run completed in 5 minutes 13 seconds
  with the current local build cache.
- PR #47's initial run `30156608755` passed in 10 minutes 37 seconds. Its Go
  job set the 10-minute-25-second critical path while Rust finished in 8
  minutes 3 seconds, versus the 19-minute-17-second PR #46 baseline. The
  topology change therefore shortened representative verification by about
  46% without changing coverage.
- The required review found one protected-environment policy gap; the guard and
  mutation tests now reject environment-gated Go, Rust, native, and aggregate
  verification jobs. No decision-required findings remain.
- CodeRabbit's remaining finding was valid: executed aggregate-script tests
  now use a bounded command context and distinguish timeouts from expected
  fail-closed exits. Targeted releaseguard and verification tests and vet pass
  with the follow-up.
- Candidate: canonical model and rendered-artifact fixture refactor.
  Classification: included. Contract basis: the second included result.
  Action: proceed without changing functional or race coverage.
- Reconnaissance found 23 full canonical surface loads in
  `internal/sdkgen/model` and about 32 full render pipelines in
  `internal/sdkgen`. The refactor loads and serializes the canonical model once
  for isolated clones and renders one immutable artifact fixture for mutation
  and filesystem cases; one complete public generation/freshness path remains.
- Forced-fresh package time fell from 14.2 to 1.7 seconds for
  `internal/sdkgen/model` and from 36.3 to 4.6 seconds for `internal/sdkgen`.
  The unchanged full race suite fell from 5 minutes 13 seconds to 2 minutes 2
  seconds with the workstation's current build cache.
- `TestVerifyAcceptedRepository` remains because it is the only package test
  that exercises all real verification dependencies; the explicit command is
  operational end-to-end coverage rather than a substitute for that regression
  path.
- Vet, all normal tests, the repository verifier, repeated shuffled fixture
  tests, focused race tests, and the full race suite pass.
- Independent review found no delivery or correctness gaps. CodeRabbit's one
  valid test-strength finding is addressed by requiring matching security and
  response-schema fields and asserting both remain isolated across clones.
