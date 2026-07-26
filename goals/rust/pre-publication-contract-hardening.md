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
- CLI command and dispatch contract hardening is complete. PR #56 merged the
  exhaustive generated dispatch matrix, deepest-prefix help, and narrow
  hyphen-value normalization after native CI and review passed.

### Current in-scope result

- Make the native SDK client boundary warning-free and dependency-clean on
  WebAssembly, and make CLI encoding and home-path output deterministic across
  Unix and Windows.

### Next in-scope action

- Add the failing default-feature WebAssembly gate, Unix non-UTF-8 home-output
  test, and platform-native home resolver tests before changing cfg or emission
  code.

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
- Commits `16df19e`, `f9a154c`, and `200341c` add exhaustive generated dispatch
  evidence, shared deepest-prefix help, and narrow hyphen-value normalization.
  The full credential-free pre-push gate passes with the new discovery matrix,
  compatibility suites, stable/MSRV checks, package verification, and clean
  installation.
- PR #56 merged as `14eb163` after Go, Rust, macOS, Windows, aggregate CI, and
  review passed. Its valid test-isolation finding was fixed, the rejected
  source-group expansion was withdrawn, and no review thread remained open.
- The requested `$progress` skill is unavailable in this session. Recovery was
  performed from the original attached contract, repository plans, current
  implementation and tests, git history, and merged PR evidence.
