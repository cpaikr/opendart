# Shorten CI and local verification feedback

## Outcome

Make verification fast enough to remain part of the normal development loop
without weakening release confidence. Go and Rust verification should run
independently, one stable aggregate check should report the complete result,
and expensive tests should pay only for the behavior they actually exercise.

## Current state

- PR #46 is merged. The goal now runs from the dedicated
  `goal/rust-ci-verification-feedback` integration branch based on the updated
  `rust` branch, with the local planning patch preserved.
- The `Verify` workflow serializes Go and Rust work in one Linux `verify` job.
  The macOS and Windows artifact jobs already run independently.
- PR #46 run `30153472800` took 19 minutes 17 seconds. The Go race suite used
  10 minutes, and the Rust work after repository verification used about
  7 minutes 41 seconds. Parallel Go and Rust jobs should therefore reduce the
  critical path to roughly 11–12 minutes before any tests are changed.
- On an 8-core Apple Silicon workstation on 2026-07-25, a forced fresh
  `go test -count=1 ./...` execution took 42 seconds while the same suite under
  the race detector took 14 minutes 27 seconds. Repository verification took
  6 seconds, and the warm Rust stable contract suite took 50 seconds. The
  blanket Go race command is therefore a local feedback problem as well as a
  CI problem.
- The most expensive race-instrumented Go packages are `internal/sdkgen`,
  `internal/sdkgen/model`, `internal/openapi`, `internal/verification`,
  `internal/guide`, and `internal/liveconformance`. Several generator and model
  mutation tests repeatedly rebuild the same canonical OpenAPI-derived state.
- The workflow also runs the repository verifier explicitly while
  `TestVerifyAcceptedRepository` exercises a full accepted-repository path
  inside the Go suite. Their assertions may overlap, but that overlap has not
  yet been mapped well enough to remove either gate safely.
- `internal/releaseguard` intentionally fixes the workflow's job and step
  shape, including `go test -race ./...`; the guard and its tests must change
  with the workflow rather than be bypassed.
- A read-only repository-settings check on 2026-07-25 found no required status
  checks or rulesets. A stable aggregate `verify` result is still the documented
  release contract and avoids churn if branch protection adopts it later.
- Go setup already enables the supported Go cache. Rust dependency fetching is
  locked and offline after fetch, but compiled Rust outputs are not cached.

Timing measurements are comparative evidence, not permanent budgets. Recheck
them on representative Rust-only, Go/tooling, and mixed changes after each
stage.

## Verification policy

### Every pull request

- Run `go vet ./...`, normal `go test ./...`, and the explicit repository
  verifier. Normal tests remain the comprehensive functional-correctness gate;
  a slow test is not removed merely because it is slow.
- Run race-instrumented tests for packages that own concurrency or exercise
  shared mutable state. Establish that list from a code-and-test audit,
  including HTTP server lifecycles, cancellation, shared fixtures, and global
  state; a search for goroutines alone is insufficient.
- Keep the current Rust stable, compatibility, MSRV, package-content, and clean
  source-install contracts. They protect distinct supported configurations and
  should not be dropped as part of the topology change.
- Keep native macOS and Windows artifact behavior as independent required jobs.
- Make a final Linux `verify` job depend on all required jobs, run even when a
  dependency fails or is skipped, and succeed only when every required result
  is successful. This preserves one durable result without masking failures.

The first workflow change must retain `go test -race ./...` unchanged. Splitting
topology and changing test coverage in one patch would make failures and timing
comparisons harder to interpret.

### Scheduled and explicit exhaustive verification

- After the race-ownership audit and test refactor, move the full
  `go test -race ./...` sweep to a scheduled and manually dispatchable gate.
  Keep targeted race packages on every pull request.
- Run the full race sweep before releases and when changing the race policy.
  Do not introduce path-based exemptions for concurrency-sensitive changes
  until package ownership and transitive dependencies are explicit.
- Treat a scheduled failure as a regression requiring repair, not as optional
  telemetry. Document who owns it and how a candidate fix can rerun it.

### Local development

- Provide a fast edit loop based on focused normal tests for the affected Go or
  Rust package.
- Provide one pre-push command that mirrors the required Linux pull-request
  contract: all normal Go tests, vet, repository verification, the audited
  targeted race suite, and the stable, compatibility, MSRV, packaging, and
  clean-install Rust gates.
- Provide an explicit exhaustive command that adds the full Go race sweep to
  the pre-push contract. Native macOS and Windows artifact checks remain
  CI-owned.
- Do not make the 14-minute local full-race run the default edit loop. Also do
  not use `testing.Short`, build tags, or command flags to hide avoidable
  fixture construction; flags should distinguish genuinely exhaustive or
  environment-specific coverage.

The implementation should use repository-owned commands or scripts so the
local tiers and CI invoke the same definitions instead of duplicating package
lists in prose and YAML.

## Implementation plan

### Split the workflow without changing coverage

- Replace the monolithic Linux job with independent `go` and `rust` jobs.
  Give each job only its own checkout, toolchain setup, and commands.
- Add a lightweight `verify` fan-in over Go, Rust, macOS artifacts, and Windows
  artifacts. Use an unconditional dependency evaluation and explicitly reject
  failed, cancelled, or unexpectedly skipped required jobs.
- Update `internal/releaseguard` and its fixtures to validate responsibilities
  per job and the fan-in contract instead of the old serial step list.
- Update release and contributor documentation to name the aggregate result
  and explain the underlying required jobs.
- Compare the new critical path with the PR #46 baseline while retaining the
  full Go race suite. Do not add caching or test-selection changes to this
  first measurement.

### Remove repeated canonical construction

- Inventory the assertions and setup costs in the slow Go packages before
  changing their boundaries.
- In `internal/sdkgen/model`, inspect the canonical surface once and give each
  mutation case a deep clone. Nested slices, maps, and pointers must not leak
  mutations between tests.
- In `internal/sdkgen`, render one immutable canonical artifact tree for
  mutation scenarios, then copy it into a fresh temporary directory for each
  case. Keep bespoke-input tests independent.
- Preserve a small number of true end-to-end tests that parse the source
  contract, construct the model, render artifacts, and verify repository
  freshness together. Test narrower marker, mutation, and filesystem behavior
  through narrower seams.
- Map `TestVerifyAcceptedRepository` against the explicit repository verifier.
  Remove duplicate work only when each unique failure class has another
  durable assertion and the command-level acceptance path remains covered.
- Benchmark normal and race-instrumented package timings after the refactor.
  Prefer structural savings that improve both CI and local runs.

### Define the race-owned package set

- Audit production and test concurrency package by package. Record why each
  always-raced package owns a concurrency invariant and which tests exercise
  it.
- Confirm that the targeted command actually runs the concurrent behavior;
  race instrumentation without the relevant execution path is not coverage.
- Keep the full race sweep on every pull request until the audited targeted
  suite and scheduled/manual full sweep exist together.
- Centralize the package set and its rationale in one repository-owned
  entrypoint consumed by CI and local commands. Verify that CI invokes that
  entrypoint rather than duplicating the list, and require the audit to be
  revisited when production or test concurrency changes.

### Establish fast, pre-push, and exhaustive entrypoints

- Add documented repository commands for the three local tiers and use those
  same commands from GitHub Actions where practical.
- Make the pre-push tier representative of required pull-request checks, not a
  second independently maintained approximation.
- Keep credentialed live conformance outside ordinary offline verification and
  require the existing environment wrapper whenever it is run.
- Explain expected use and approximate measured scale without promising fixed
  runtimes across machines.

### Evaluate caching and conditional execution last

- Re-measure cold and warm Rust jobs after parallelization and test refactoring.
- If compilation remains material, evaluate a bounded Cargo build cache or
  compiler cache keyed by operating system, toolchain, lockfiles, and relevant
  feature inputs. Do not cache credentials or blindly upload an unbounded
  workspace `target` tree.
- Account for fork pull requests being able to read eligible caches; cached
  content must never be trusted as a security boundary.
- Consider path-aware exhaustive jobs only after dependency ownership is
  machine-checkable. Normal Go tests and repository verification should still
  run for Rust changes because Go tooling owns generated Rust artifacts and
  workflow policy.

## Validation

- Exercise success, failure, cancellation, and unexpected-skip cases for the
  aggregate job; a failed dependency must never produce a green `verify`.
- Run `internal/releaseguard` tests and the repository verifier against the new
  workflow structure.
- Compare several successful runs across Rust-only, Go/tooling, and mixed
  changes. Record per-job and critical-path timings rather than only total
  workflow duration.
- Confirm the initial topology change preserves every existing command and
  supported Rust platform/configuration.
- After the portfolio change, show that all normal Go tests still run on every
  pull request, targeted race tests are enforced there, and the full race suite
  is runnable both on schedule and on demand.
- Run each documented local tier from a clean checkout without credentials.

## Completion criteria

- Go and Rust verification run in parallel behind one truthful aggregate
  `verify` result.
- Normal pull-request verification is materially faster while retaining
  functional, generated-artifact, platform, compatibility, and audited race
  coverage.
- Repeated canonical model and artifact construction no longer dominates the
  affected Go tests.
- Developers have one fast loop, one faithful pre-push gate, and one explicit
  exhaustive gate; the full race suite is not implicit in every local edit.
- Scheduled and manual full-race failures have an owned response path.
- Any Rust build cache is bounded, reproducible from reviewed inputs, and
  treated only as an optimization.

## Next action

Implement only the first workflow topology slice: parallel `go` and `rust`
jobs, a failure-aware aggregate `verify` job, matching `internal/releaseguard`
coverage, and documentation updates. Retain the full Go race command so the
new timing establishes a comparable baseline before changing the test
portfolio.
