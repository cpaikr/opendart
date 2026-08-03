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

- Reviewed product contract and Rust-native conformance gate. Implementation,
  repository validation, independent review, feedback resolution, and PR
  delivery completed in [PR #74](https://github.com/cpaikr/opendart/pull/74).

### Current in-scope result

CLI-owned grammar, discovery, and isolated private dispatch. The result is
active from merge commit `d071546`; implementation and independent review are
complete, while clean-tree qualification and PR delivery remain pending.

### Next in-scope action

Commit the reviewed CLI projection slice, run its complete clean-tree offline
Rust qualification, then create, review, and merge its PR before starting the
handwritten conformer result.

### Evidence and blockers

- Result-one delivery: PR #74 merged into `rust` as `d071546` after all required
  checks passed and all 20 review conversations were resolved.
- Boundary check: the current result is directly included by the original contract; public SDK publication and adoption remain outside this goal.
- First-result validation: full Go verification, the focused seven-case offline
  Rust gate, strict Clippy, diff hygiene, and independent implementation and
  system reviews passed with no actionable findings.
- Result-two implementation: 85 reviewed commands and 167 physical dispatch
  cases are generated into independently checksummed interface and private
  dispatch trees; public discovery exposes source concepts and coarse output
  shape without private Rust symbols.
- Result-two review: generator and Rust CLI/package reviewers found no remaining
  actionable issues after the manifest input directory was made explicit and
  missing/orphan mapping diagnostics were tightened.
- Result-two validation: repository verification, all Go tests, Rust formatting,
  strict Clippy, CLI unit and discovery tests, compatibility loopbacks, and the
  dirty-tree package inventory pass. The full Rust gate reached packaging and
  requires the planned clean commit because Cargo rejects dirty packages.
