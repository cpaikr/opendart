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

_None._

### Current in-scope result

Parallel Go and Rust verification behind one failure-aware, stable aggregate
`verify` result, with matching workflow-policy guards and truthful release
documentation.

### Next in-scope action

Implement the first workflow-only PR while retaining `go test -race ./...`,
then measure its critical path before changing test coverage.

### Evidence and blockers

- PR #46 merged as `750ba04`; the goal integration branch is based on that
  updated `rust` state.
- The local planning commit `90c0963` is preserved as equivalent rebased commit
  `2a88770` on the integration branch.
- Candidate: first workflow topology PR. Classification: included. Contract
  basis: the first included result and explicit first-PR delivery constraint.
  Action: proceed without changing test coverage.
