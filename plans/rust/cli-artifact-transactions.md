# Make CLI artifact writes transactional and runtime-safe

## Outcome

Turn binary artifact handling into an explicit transaction that publishes only
the bytes written through the owned file handle, preserves the primary source
or limit failure when cleanup also fails, and never performs blocking
filesystem work on the current-thread async runtime.

## Current state

- `ArtifactTransaction` now starts one blocking worker before network access.
  The worker alone owns the retained destination and staging directory
  capabilities, staged file, byte counter, limit, and no-clobber commit.
- Body chunks cross a bounded Tokio channel with capacity one. File creation,
  writes, flush, hard-link publication, and cleanup all execute on the blocking
  worker rather than the current-thread async runtime.
- Publication uses `cap-std` 4.0.2 and verifies that the private staged entry
  still identifies the retained file before creating the destination hard
  link. The adversarial pathname-replacement and rival-destination process
  tests now pass on macOS.
- Cleanup failure is optional top-level secondary evidence. Process coverage
  preserves artifact-limit errors, source status, and successful archive
  replies while attaching the documented cleanup object.
- The production-boundary stalled-writer test fills the one-chunk queue,
  observes a Tokio timer on the current-thread runtime, permanently revokes
  commit authority, returns without waiting for the stalled call, and confirms
  deferred staging cleanup after that call is released.
- The effective SDK total deadline, including its default when the CLI flag is
  omitted, now covers bounded-channel backpressure and flush completion. Any
  request or body timeout reports `cleanup_pending` without awaiting a detached
  worker's private-staging cleanup; that worker can no longer publish.
- Until the package dependency version advances, the CLI mirrors the current
  SDK default in `execution.rs`: its tarball must keep compiling against the
  already-published SDK at the same exact version. An explicit CLI override is
  still passed to both the SDK and artifact worker from one value.
- Private staging names use 128 bits of operating-system randomness. Pathname
  replacement safety begins once the private directory capability is acquired;
  the public contract requires a non-hostile parent during that short setup
  window.
- Commit and cancellation use an acknowledged atomic state handoff. A timeout
  wins only from the active state; after commit begins, the caller waits for
  publication and cannot report cancellation.
- Stable Clippy and the full compatibility artifact process suite pass. Native
  macOS checks pass on the Rust 1.85 floor, and the Rust 1.85 CLI cross-check
  passes for Windows. The Linux CLI cross-check is blocked locally by the
  native-TLS OpenSSL sysroot, while the isolated publication prototype still
  compiles for Linux. Native Linux and Windows process behavior remains a
  validation gate.
- The deadline, staging-setup trust boundary, and commit handoff policies found
  by review are now resolved in implementation and public documentation.
- Follow-up review also verified the SDK-default deadline path and pre-stream
  timeout cleanup path after their focused regressions were added; no further
  implementation finding remains for this slice.

## Transaction contract

- Keep these phases visible: validate destination, create owned staging state,
  stream bounded bytes, flush durable content as required by the existing
  contract, encode the final report, publish without clobber, then write the
  report to stdout.
- Publication must operate on the identity represented by the retained staging
  handle or a descriptor-relative private staging directory. It must not trust
  a later lookup of a public temporary pathname.
- The destination must remain absent before commit. Publication must be atomic
  and no-clobber on Linux, macOS, and Windows.
- The primary failure code and its source metadata are immutable once chosen.
  Cleanup failure is bounded secondary evidence, never a replacement.
- A timed-out or cancelled writer may never publish later. Cancellation must
  revoke commit authority before an error is returned.
- The async task may await bounded backpressure, but it may not execute file
  creation, write, flush, close, or publish syscalls directly.
- Keep the production design concrete. Add a narrow private writer interface
  only if it is needed to inject deterministic stalls and failures in tests;
  do not expose filesystem strategy through the public CLI contract.

## Implementation plan

### Specify adversarial acceptance tests first

- Add a test that pauses after staging, replaces the public staging pathname,
  resumes publication, and proves replacement bytes are never published or
  reported.
- Assert one of two acceptable outcomes: the original handle's exact bytes are
  atomically published, or publication fails safely with no destination.
- Add cleanup-failure fixtures for source status, transport/body failure,
  artifact limit, write failure, and publish failure. Assert the primary
  `error.code`, operation, and response metadata remain unchanged.
- Add a controllable writer that blocks after accepting a chunk. While blocked,
  prove the runtime can poll a timer and the response stream, cancellation
  removes commit authority, the channel remains bounded, and no final
  destination appears.
- Keep real filesystem process tests for same-directory behavior, symlinks,
  existing destinations, and platform no-clobber semantics.

### Select an identity-stable publication primitive

- Use `cap-std` directory capabilities for the opened destination parent and a
  private staging directory. Publish with `Dir::hard_link`, which creates the
  destination without replacing an existing entry, then remove the staging
  link and directory.
- Create the private staging directory with owner-only access on Unix. On
  Windows, keep directory handles open and deny write/delete sharing on the
  staged file through publication. Validate those native guarantees in the
  platform process suites.
- Treat missing filesystem hard-link support as a safe publish failure with no
  destination. Do not fall back to path-based persistence.
- Preserve the retained parent identity if its pathname changes. Document that
  callers must keep ancestors of the destination parent stable; no publication
  primitive can keep a caller-visible path stable after an adversary replaces
  its ancestor.
- Keep all repository code safe. Platform-specific descriptor and handle work
  remains encapsulated by the maintained dependency.

### Preserve primary errors and attach cleanup context

- Add one optional top-level `cleanup` field to every binary error or response
  document, including successful archive publication. This location is the
  confirmed public contract; consumers must not need to inspect different
  nested paths based on the primary outcome.
- Use a small enum for cleanup stage/reason. Do not store raw OS messages,
  paths other than the already-sanitized caller spelling, or dependency debug
  output.
- Refactor cleanup helpers to return cleanup context. Construct the primary
  error at the failure site, attempt cleanup, then attach the secondary result.
- Apply the same rule to source-status handling: if discarding the unused
  staging file fails, retain the status reply as the primary outcome and add
  cleanup evidence in the documented representation.
- Update JSON fixtures and the public error contract as an additive schema
  change. Check downstream discovery/error examples for exact snapshots.

### Move filesystem work behind a bounded worker

- Give one dedicated blocking worker ownership of the staging handle,
  byte count, limit, and publication capability.
- Feed it body chunks through a bounded channel. Keep the capacity small and
  explicit so memory use is bounded and backpressure is visible.
- Enforce the inclusive byte limit before enqueue or before write with one
  authoritative counter. Avoid maintaining independent counters that can
  disagree with the final report.
- Return typed worker events for limit, write, flush, publish, and cleanup
  outcomes. Do not collapse them into strings.
- Add a cancellation/commit permit shared across the async owner and worker.
  Revoke it when the body stream fails or a deadline/cancellation path wins,
  and require a final check immediately before publication.
- Define worker shutdown behavior for an in-flight blocking syscall. The
  command may report a timeout only after it can guarantee no later publish;
  if immediate cancellation is impossible on a supported platform, document
  and test the bounded shutdown policy explicitly.
- Keep report encoding before the commit point and stdout writing after it, as
  required by the current artifact contract.
  Pre-encode both the ordinary report and the sole cleanup-bearing variant,
  then select between those buffers after commit cleanup finishes.

### Integrate without broadening the CLI

- Keep `ArtifactTarget` responsible for caller input and destination spelling.
  Give a distinct private type ownership of the staged transaction and make
  commit consuming so it cannot run twice.
- Keep `Archive`, `Status`, and `Unrecognized` branch behavior visible in
  `execute`; do not hide it behind a generic artifact framework.
- Preserve exact-byte streaming, inclusive limits, metadata sanitization,
  source-status exit behavior, unrecognized artifact reporting, and
  post-publication stdout-failure semantics.
- Update package dependencies and lockfile only after the cross-platform spike
  proves the selected filesystem API.

## Validation

- Run focused artifact unit tests and the full `binary_loopback` process suite
  with `opendart_compat`.
- Run native artifact tests on Linux, macOS, and Windows, including staged-name
  replacement and no-clobber publication.
- Run timeout tests under the same current-thread runtime used by the binary
  and assert timer progress while the writer is stalled.
- Run stable and MSRV Clippy/tests, rustdoc, package inventories, clean
  source-install checks, and repository verification.
- Confirm the no-default SDK graph remains transport-independent; CLI worker
  dependencies must not leak into `opendart`.

## Completion criteria

- Replacing or unlinking any public staging path cannot substitute published
  bytes.
- Every primary failure retains its original stable code and metadata even if
  cleanup fails.
- The runtime continues polling network and timer work while filesystem calls
  are blocked.
- Cancellation or timeout cannot be followed by a late artifact publication.
- All supported platforms preserve no-clobber, exact-byte, cleanup, and output
  contracts.

## Next action

Complete the cleanup-failure matrix for transport/body, write, and publication
failures, then run the artifact process suite natively on Linux and Windows.
