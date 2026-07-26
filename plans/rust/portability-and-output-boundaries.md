# Harden Rust portability and output boundaries

## Outcome

Make the SDK's native-client feature boundary compile cleanly on WebAssembly
and make CLI discovery/error output deterministic across Unix and Windows path
and home-directory representations.

## Current state

- Every native client dependency and client-owned prepared-request field or
  helper now uses the same native-target boundary. The default feature is inert
  on WebAssembly while the prepared-request surface remains available.
- The Rust verification tier installs Rust 1.97.1's
  `wasm32-unknown-unknown` target, runs default and no-default warnings-denied
  Clippy, and rejects native transport/runtime packages in the target graph.
  Releaseguard owns the exact commands.
- Global emission encodes before borrowing stdout. A serialization failure
  becomes one static `output_encode` document; a write failure remains silent
  and nonzero without appending a second document. Operation-scoped paths keep
  their existing context.
- Home display uses an injectable, race-free resolver. Unix accepts only an
  absolute nonempty `HOME`; Windows prefers an absolute nonempty `USERPROFILE`
  and falls back to `HOMEDRIVE` plus `HOMEPATH`. Prefix comparison remains
  component-wise.
- Unit coverage exercises missing, empty, relative, equal, child,
  sibling-prefix, and non-UTF-8 paths. Linux process coverage exercises a
  non-UTF-8 executable path, while Windows CI runs native resolver tests and an
  installed discovery probe with `HOME` absent.
- The complete credential-free pre-push gate passes locally. Hosted Linux,
  macOS, and Windows runtime assertions pass, including the installed Windows
  discovery probe and Linux non-UTF-8 executable-path case.

## Design decisions

- Treat the HTTP client as native-only throughout the implementation, not only
  at the reqwest dependency declaration. Use one repeated cfg expression or a
  small private cfg module rather than partially compiled dead state.
- A default-feature WebAssembly build should expose the same
  transport-independent API as a no-client build and compile without client
  dependencies. Do not add a WebAssembly HTTP transport in this plan.
- Split output into an encode phase and a write phase. Encoding failure before
  any stdout bytes may emit the fixed global `output_encode` error document;
  write failure after output begins remains silent and exits nonzero.
- The fallback error must contain only statically safe, UTF-8 fields and must
  not attempt to serialize the value that failed.
- Resolve a platform-native home path through a small private function with an
  injectable environment lookup for tests. On Unix use `HOME`; on Windows use
  `USERPROFILE`, then the established native fallback if required.
- Compare paths component-wise using `Path`; do not normalize case, resolve
  symlinks, or invent shell quoting.

## Implementation plan

### Make the native client cfg uniform

- Inventory every `client-reqwest` module, export, type field, helper, impl, and
  dependency activation in `opendart`.
- Apply `all(feature = "client-reqwest", not(target_family = "wasm"))`
  consistently at the module/export boundary so native-only internals are not
  parsed into the WebAssembly crate.
- Move all client-only optional dependencies, not only reqwest, under the same
  target condition where Cargo supports it. Verify feature activation on
  WebAssembly does not pull bytes, futures, Tokio, reqwest, TLS, DNS, or proxy
  crates into the normal graph.
- Keep the public feature name and native default behavior unchanged.
- Document that `client-reqwest` is inert on WebAssembly and that callers use
  prepared requests with their own executor there.

### Add WebAssembly as a maintained compile gate

- Determine whether Rust 1.97.1 publishes the selected WebAssembly target in CI
  and clean local environments. Record the supported target/toolchain pair in
  `sdk/rust/README.md`.
- Run default-feature and no-default-feature `cargo check` or Clippy for
  `wasm32-unknown-unknown`. Use `-D warnings` for the default-feature case that
  exposed the gap.
- Add the commands to CI and repository verification policy without weakening
  native stable/MSRV or no-default dependency checks.
- Inspect the target dependency tree and add a regression assertion for the
  absence of native transport/runtime packages.

### Separate CLI encoding from stdout writes

- Refactor `output::encode` and the app emission path so serialization returns
  owned UTF-8 bytes before stdout is borrowed for writing.
- On serialization failure, encode a fixed global `ErrorEnvelope` with
  `output_encode` and no operation/response metadata, then attempt one stdout
  write.
- If encoding the fixed fallback violates an internal invariant, use a local
  `expect` whose message states why the static envelope must serialize; do not
  add recursive error handling.
- If either the original or fallback stdout write fails, return exit `1`
  without appending another document.
- Keep operation-scoped encoding failures in execution paths carrying their
  existing safe operation/metadata context.
- Add a Unix process or focused app test using a non-UTF-8 executable path and
  assert one valid `output_encode` JSON document, empty stderr, and exit `1`.

### Resolve home display portably

- Extract home discovery from `Home::new` into a private platform resolver that
  accepts an environment accessor. This avoids process-global environment
  mutation in parallel tests.
- On Unix, use a nonempty `HOME`. On Windows, prefer a nonempty `USERPROFILE`
  and evaluate `HOMEDRIVE` plus `HOMEPATH` only if needed for native behavior.
- Collapse only a true path prefix, preserve the remaining components and
  separator behavior, and leave unrelated paths unchanged.
- Add unit tests for missing, empty, relative, non-UTF-8, equal-to-home, direct
  child, sibling-prefix, and platform-native fallback cases.
- Add a Windows process-level discovery assertion in the existing native CI
  job with `HOME` absent and the native home variable present.
- Update the public contract if the exact Windows variable precedence was not
  previously specified.

## Validation

- Run both maintained WebAssembly feature configurations and dependency-tree
  checks.
- Run CLI discovery and output tests on Unix and Windows, including the
  non-UTF-8 test where the platform can represent it.
- Run pinned native stable and MSRV formatting, Clippy, tests, rustdoc, package,
  clean install, and repository verification gates.
- Confirm the ordinary native client and no-default SDK surfaces remain
  unchanged.

## Completion criteria

- Default and no-default WebAssembly builds are warning-free and contain no
  native HTTP client/runtime dependency path.
- A pre-write serialization failure produces exactly one stable JSON error;
  a stdout write failure never appends a replacement.
- Home display uses the native home path on Unix and Windows without global
  test-environment races.
- Native source-install behavior and the transport-independent core retain
  their existing contracts.

## Next action

Completed in PR #57 after Go, Rust, macOS, Windows, aggregate CI, and review
passed with no unresolved actionable feedback. The active Rust roadmap work is
now the API-contract and verification plan.
