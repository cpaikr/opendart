# Goal: Deliver the offline-qualified handwritten Rust SDK conformer

Status: active
Planning scope: ROADMAP_RUST.md

## Original contract

Goal contract
- Outcome: Deliver one offline-qualified, handwritten-only Rust OpenDART SDK conformer with a decoupled CLI-owned presentation interface.
- Goal state: goals/rust/handwritten-sdk-cutover.md
- Included results and sources (semantic results define scope; paths supply detail):
  - Reviewed product contract and Rust-native conformance gate — plans/rust/handwritten-sdk-contract-and-conformance.md
  - CLI-owned grammar, discovery, and isolated private dispatch — plans/rust/cli-presentation-projection.md
  - Handwritten SDK implementation, CLI retargeting, generated-SDK deletion, documentation reconciliation, and offline package qualification — plans/rust/handwritten-sdk-conformer.md
- Complete when: Every included result achieves its cited outcome and applicable completion criteria within its named semantic boundary; repository-required validation and review pass; planning is truthful; Delivery finishes.
- Excluded: Public SDK publication and adoption described by plans/rust/public-rust-sdk.md.
- Authority: Execute only included results and necessary supporting work; resolve remaining decisions within that closed outcome using best judgment; record anything else and ask before scope expansion or external authority.
- Resume: Initialize this contract with $progress goal mode before work; recover it before every resume, continuation, compaction, or handoff; stop if recovery fails.
- Delivery: PR delivery — use $progress's PR lifecycle and the fewest sequential reviewable PRs; finish each through $create-pr and $address-pr-feedback before starting the next, including the final implementation slice.

## Authorized amendments

_None._

## Execution status

### Completed included results

_None._

### Current in-scope result

Reviewed product contract and Rust-native conformance gate.

### Next in-scope action

Review `plans/rust/handwritten-sdk-contract-and-conformance.md` against repository evidence, resolve its remaining in-scope decisions, and implement its conformance gate.

### Evidence and blockers

- Delivery boundary: use the established non-production `rust` integration branch and preflight its direct metadata push before creating the first work branch.
- Boundary check: the current result is directly included by the original contract; public SDK publication and adoption remain outside this goal.
