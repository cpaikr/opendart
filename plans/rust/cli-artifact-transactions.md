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
- Existing process tests cover ordinary cleanup, destination races, size
  limits, exact bytes, source status, and broken stdout, but do not replace the
  staged pathname, force cleanup failure while another error is active, or
  stall a writer while observing the runtime.

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

- Spike safe, maintained descriptor-relative or handle-relative filesystem
  APIs on every supported source-install platform.
- Evaluate retained parent-directory identity, private staging-directory
  permissions, source-file identity, atomic no-clobber publication, symlink and
  directory replacement behavior, cleanup, MSRV, and crate feature cost.
- Prefer a safe library abstraction over repository-owned `unsafe`. The
  workspace forbids unsafe Rust, and this plan must not weaken that lint.
- If portable identity-stable publication cannot satisfy the existing
  arbitrary-parent contract, stop for an explicit contract decision. The only
  fallback is to restrict output to a caller-affirmed trusted directory and
  document that limitation; silently accepting attacker-writable parents is
  not acceptable.
- Record the chosen primitive and threat boundary in
  `docs/rust-cli/architecture.md` and `docs/rust-cli/public-contract.md`.

### Preserve primary errors and attach cleanup context

- Extend the stable CLI error envelope with an optional structured cleanup
  field or equivalent bounded secondary context.
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

Write the adversarial staged-name replacement, cleanup-secondary, and stalled
writer tests, then run the cross-platform publication-primitive spike before
refactoring `StagedArtifact`.
