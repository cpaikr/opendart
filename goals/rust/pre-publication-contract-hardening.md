# Goal: Finish Rust pre-publication contract hardening

Status: active
Planning scope: rust

## Original contract

Goal contract
- Outcome: Finish the remaining pre-publication correctness, portability, and verification hardening for the Rust SDK and CLI so only separately authorized registry publication and adoption work remains.
- Goal state: goals/rust/pre-publication-contract-hardening.md
- Included results and sources (semantic results define scope; paths supply detail):
  - Complete transactional artifact failure coverage so transport/body, write, and publication failures preserve their primary result when cleanup also fails — plans/rust/cli-artifact-transactions.md
  - Mechanically prove every generated CLI operation, alias, and representation dispatch; provide deepest-valid-context help; and safely accept supported hyphen-leading values — plans/rust/cli-command-contracts.md
  - Make the native SDK client boundary warning-free and dependency-clean on WebAssembly, and make CLI encoding and home-path output deterministic across Unix and Windows — plans/rust/portability-and-output-boundaries.md
  - Bound canonically limited generated iterators, document meaningful errors for every public fallible Rust API, and align the advertised offline verification gate with CI — plans/rust/api-contracts-and-verification.md
- Complete when: All four included results satisfy their documented completion criteria; generated artifacts, fixtures, public documentation, and planning state are truthful and fresh; applicable credential-free stable, MSRV, feature-graph, WebAssembly, native-platform, package, compatibility, and repository gates pass; required review has no unresolved actionable findings; and the selected Delivery lifecycle is finished.
- Excluded: Public CLI Work 9; Rust SDK crates.io publication or collector adoption; registry ownership checks; CLI publication or release finalization; prebuilt archives, installers, signing, and package-manager work; credentialed live conformance; non-Rust planning; and unrelated product or release-system changes.
- Authority: Only included results and the smallest bounded work necessary for their completion; record other work without executing it; expansion requires explicit user instruction.
- Resume: Invoke $progress in goal mode to initialize this exact contract before initial work and recover it before every resume, automatic continuation, compaction recovery, or handoff; fail closed if recovery fails.
- Delivery: PR delivery

## Authorized amendments

_None._

## Execution status

### Completed included results

_None. The artifact result remains active until its follow-up delivery
lifecycle is complete._

### Current in-scope result

- The transactional artifact implementation merged in PR #54. Its deferred
  cleanup-failure matrix now covers transport/body delivery, staged writes,
  and publication while proving the complete primary document is unchanged
  when cleanup evidence is attached. Deliver the follow-up through native CI,
  review, and merge.

### Next in-scope action

- Commit the reviewed matrix, rerun the clean package and install gates, then
  open the follow-up PR and finish its review and merge before starting CLI
  command-contract work.

### Evidence and blockers

- PR #54 merged as `06dc2f6` after Linux, macOS, Windows, aggregate CI, and
  review passed; its body explicitly deferred this matrix.
- Focused `opendart_compat` binary loopback coverage and warnings-denied Clippy
  pass with the matrix implementation.
- The independent implementation review found no code, cross-platform,
  security, fidelity, scope, or complexity issue. Its sole finding was to
  refresh the artifact plan's stale next action.
- The full pre-push gate passed through Go, repository freshness, stable Rust,
  compatibility, dependency-graph, rustdoc, and MSRV checks; Cargo packaging
  then correctly stopped because the reviewed files were not yet committed.
- The requested `$progress` skill is unavailable in this session. Recovery was
  performed from the original attached contract, repository plans, current
  implementation and tests, git history, and merged PR evidence.
