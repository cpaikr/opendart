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

- Transactional artifact cleanup-failure coverage is complete. PR #55 merged
  the body-delivery, staged-write, and publication matrix after native CI and
  review passed; each case preserves the full primary result while attaching
  only bounded cleanup evidence.

### Current in-scope result

- Close the CLI command and dispatch contract gaps: mechanically prove every
  generated canonical operation, logical-ID alias, and representation; return
  help for the deepest valid command prefix; and accept supported
  hyphen-leading values without consuming real flags.

### Next in-scope action

- Add the exhaustive generated keyless dispatch evidence and the failing
  nested-help and hyphen-leading argv boundaries, then implement the shared
  command-context and normalization paths that make them pass.

### Evidence and blockers

- PR #54 merged as `06dc2f6` after Linux, macOS, Windows, aggregate CI, and
  review passed; its body explicitly deferred this matrix.
- Focused `opendart_compat` binary loopback coverage and warnings-denied Clippy
  pass with the matrix implementation.
- The independent implementation review found no code, cross-platform,
  security, fidelity, scope, or complexity issue. Its sole finding was to
  refresh the artifact plan's stale next action.
- Commit `ba3fe36` passes the complete credential-free pre-push gate: Go and
  repository verification, stable and MSRV Rust, structured and binary
  compatibility suites, transport-independent and reqwest compatibility
  graphs, rustdoc, package contents and verification, and clean CLI install.
- PR #55 merged as `b6439e5` after Linux, macOS, Windows, aggregate CI, and
  review passed. The valid review finding was fixed by routing deterministic
  failures through the production cleanup handlers; the final review had no
  unresolved actionable findings.
- The requested `$progress` skill is unavailable in this session. Recovery was
  performed from the original attached contract, repository plans, current
  implementation and tests, git history, and merged PR evidence.
