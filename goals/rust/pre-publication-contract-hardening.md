# Goal: Finish Rust pre-publication contract hardening

Status: complete
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
- Portability and output-boundary hardening is complete. PR #57 merged the
  WebAssembly-native client boundary, single-write output encoding, and native
  Unix/Windows home discovery after hosted platform CI and review passed.
- API contract and verification hardening is complete. PR #58 merged bounded
  generated iterators, operation-specific public error documentation, and the
  exact offline compatibility commands after the complete credential-free
  pre-push gate, hosted CI, and review passed.

### Completion state

- Every included result satisfies its plan's completion criteria. The final
  audit found no failed required checks or unresolved review threads across
  PRs #55 through #58, and generated artifacts, public documentation, and
  planning state are current.
- No in-scope action remains. Registry publication, collector adoption, and
  the other explicitly excluded release work remain separately authorized.

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
- Commits `39638f5` and `8a12da8` make the native client boundary uniformly
  target-gated, add maintained WebAssembly feature/graph verification, and make
  CLI encoding and home discovery deterministic across native platforms. The
  complete credential-free pre-push gate and Windows-target test compilation
  pass locally.
- PR #57 merged as `7df7111` after Go, Rust, macOS, Windows, aggregate CI, and
  review passed. The hosted Windows probe exposed a PowerShell automatic-
  variable collision; commit `9881ea3` fixed it, rerun CI passed, and the sole
  review thread was answered and resolved.
- Commit `8f73f52` bounds generated string-array collection at maximum plus one,
  rejects an unrepresentable sentinel, documents retained invalid state, and
  proves finite, oversized, infinite, exact-maximum, and representation-specific
  behavior for every currently bounded generated parameter.
- Commit `3e6c339` generates operation-specific error contracts for every
  public preparation method, documents the handwritten fallible API inventory,
  enforces `clippy::missing_errors_doc`, and synchronizes the README's two
  credential-free compatibility commands. Focused generator, release-guard,
  public-contract, all-feature Clippy, strict rustdoc, and both compatibility
  suites pass; generated sources are fresh.
- Independent review found array-element prose ambiguity, a future sentinel
  that could exceed `usize` on supported `wasm32`, a test-only public fallible
  API in the feature-unification crate, and two handwritten error-contract
  omissions. Commits `b0197a2` and `fa624e6` resolve every finding and add the
  portable boundary regression.
- The complete credential-free pre-push gate passes on `fa624e6`: Go vet,
  normal and race tests, repository policy and generated freshness, stable and
  MSRV Rust contracts, structured and binary compatibility, strict rustdoc,
  transport-independent and WebAssembly dependency checks, reqwest feature
  unification, package verification, and clean CLI installation.
- PR #58 merged as `405f5ab` after Go, Rust, macOS, Windows, aggregate CI, and
  review passed. Review-driven fixes made the iterator sentinel portable to
  `wasm32`, removed a test-only public fallible API, and completed the precise
  handwritten error contracts.
- The final audit confirmed that PRs #55 through #58 are merged, have no failed
  required checks, and have no unresolved review threads.
- The requested `$progress` skill is unavailable in this session. Recovery was
  performed from the original attached contract, repository plans, current
  implementation and tests, git history, and merged PR evidence.
