# Make CLI artifact writes transactional and runtime-safe

## Outcome

Turn binary artifact handling into an explicit transaction that publishes only
the bytes written through the owned file handle, preserves the primary source
or limit failure when cleanup also fails, and never performs blocking
filesystem work on the current-thread async runtime.

## Current state

- `ArtifactTarget::stage` creates a `tempfile::NamedTempFile` in the destination
  directory. `stream_to_artifact` later publishes it through
  `persist_noclobber(&path)`.
- The tempfile crate documents path-based persistence as unsafe when another
  writer can replace the temporary path. In a shared writable parent, an
  attacker can unlink and recreate the staged name; publication can then move
  bytes that were not written through the retained handle while the report
  still carries the original byte count and metadata.
- `discard_error` replaces the primary error whenever `NamedTempFile::close`
  fails. A cleanup failure can therefore erase a source `Status`, transport
  failure, body-stream failure, `artifact_limit`, or earlier artifact I/O
  failure.
- Temporary-file creation, writes, flush, close, and no-clobber publication use
  synchronous `std::fs` and `std::io` calls from the CLI's current-thread Tokio
  runtime. A slow or stalled filesystem can prevent HTTP progress and timer
  polling, so configured deadlines are not observed while a write blocks.
- Process coverage now demonstrates both known contract failures: replacing a
  staged pathname can substitute attacker-controlled bytes, and a cleanup
  failure can replace an existing `artifact_limit` error. These acceptance
  tests remain red until the transaction implementation is corrected.
- The runtime-stall acceptance test is still pending. It needs the private
  worker boundary described below so the test blocks the actual production
  write path rather than a test-only imitation.
- The publication spike selected `cap-std` directory capabilities with a
  no-clobber hard-link commit. The prototype runs on macOS under the declared
  MSRV, survives staged-directory pathname replacement, preserves an existing
  destination, and compiles for Linux and Windows. Native Linux and Windows
  behavior remains an implementation-validation gate.
- `cap-std` 4.0.2 has no default features, declares an upstream MSRV below the
  workspace's Rust 1.85 floor, and keeps its dependency graph inside the CLI.
  That cost is justified because `tempfile` and other path-based atomic-write
  APIs retain the staged-name race, while direct `rustix` use does not provide
  the required Windows abstraction.

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

Add `cap-std` to the CLI package and introduce the private staged-transaction
worker boundary. Add the deterministic stalled-writer acceptance test against
that production boundary before moving creation, writes, flush, publication,
and cleanup off the async runtime.
