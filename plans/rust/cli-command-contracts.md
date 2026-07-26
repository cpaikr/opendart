# Close CLI command and dispatch contract gaps

## Outcome

Make every generated operation/representation mapping mechanically verified
and make invalid CLI invocations return help for the deepest valid command
context. Preserve the documented ability to pass hyphen-leading output and
query values without consuming real flags as data.

## Current state

- `sdk/rust/crates/opendart-cli/tests/discovery.rs` exercises generated
  dispatch through a representative company operation. It does not prove every
  operation, logical-ID alias, and representation reaches the expected typed
  SDK method and physical identity.
- Generated dispatch is exhaustive at compile time, but many arms return the
  same Rust types. A same-typed arm swap or incorrect representation mapping
  can compile and escape the representative test.
- An independent exhaustive review probe succeeded for the current generated
  tree. The finding is a missing durable regression gate, not a known current
  mapping mismatch.
- `usage_help` falls back to root help when no known operation or
  `operations list` context is detected. Bare `opendart call`, an unknown call
  operation, and bare `opendart operations describe` therefore recommend root
  commands instead of the relevant nested syntax and choices.
- Clap interprets a separate value beginning with `-` as another option.
  `--output -artifact.zip` fails even though the public contract accepts every
  nonempty Unicode path except exactly `-`; `--output=-artifact.zip` works.
- Enabling `allow_hyphen_values` blindly would consume `--help`, `--version`, or
  known operation flags as values. Query filters have the same boundary.

## Design decisions

- Use generated canonical metadata as the oracle for expected logical and
  physical identities, but exercise the public parse-and-prepare dispatch path.
  A test that compares one generated table with itself is not sufficient.
- Generate or derive one minimal valid invocation for every logical operation
  and each advertised representation. Validate both the canonical name and
  exact logical-ID alias.
- Resolve help context by walking the command tree to the deepest valid prefix
  in the supplied argv. A missing or invalid next token must not erase the
  valid parent context.
- Keep help output bounded to repository-owned command, flag, and operation
  names. Never echo arbitrary user values or dependency-formatted diagnostics.
- Preserve separate-token hyphen-leading values. Before Clap parsing, rewrite
  only known value-taking option/value pairs to `--option=<value>` when the
  value begins with `-` and is not a recognized flag, `--help`, or
  `--version`.
- Keep preprocessing narrow and explicit. Do not build a second general CLI
  parser or change ordinary non-hyphen values.

## Implementation plan

### Build exhaustive generated dispatch evidence

- Extend the normalized CLI projection with the minimum valid argv values
  needed to prepare each operation. Reuse canonical examples or constraint
  facts; do not hand-maintain a second operation inventory in Rust tests.
- For every catalog operation and representation, parse the generated
  invocation, run request construction up to the credential boundary, and
  assert logical identity, physical identity, representation, path, and
  operation context.
- Repeat through the logical-ID alias and assert it yields the same prepared
  request evidence as the canonical name.
- Include ZIP operations and structured multi-representation operations
  explicitly. Ensure generated response type alone is never used as the
  expected oracle.
- Add generator validation that every operation/representation has one test
  invocation and that stale, missing, or extra fixture output fails repository
  verification.
- Keep the tests keyless and network-free. Failure must occur before any
  `OPENDART_API_KEY` access.

### Make nested help prefix-aware

- Extract a private command-prefix resolver that walks
  `generated::command::command()` using UTF-8 command tokens until the first
  missing or invalid child.
- Return a small context enum for root, operations, operations-list,
  operations-describe, call, and known operation. Keep operation-specific flag
  lookup separate.
- Derive valid next commands or flags from the resolved command node and the
  generated catalog. Preserve deterministic order.
- Update `invocation_reason`, `valid_subcommands`, and `usage_help` to share the
  same resolved context rather than inferring context independently.
- Add process tests for bare `call`, unknown call operation, bare
  `operations`, bare `operations describe`, unknown operation description,
  operation-level missing flags, and non-UTF-8 command tokens where supported.
- Assert safe exact help entries and verify arbitrary invalid values never
  appear in stdout or stderr.

### Accept hyphen-leading data without swallowing options

- Inventory the CLI options whose values legitimately may begin with `-`.
  Start with artifact output and free-text query; include generated string
  parameters only if their canonical contract allows such values.
- Build the recognized-option set from the same command/catalog metadata used
  by parsing. Include global execution flags, operation flags, representation
  flags, help, and version.
- Add a pre-parse argv normalization function over `OsString` that:
  - preserves argv zero and all nonmatching tokens;
  - recognizes exact value-taking option tokens in the current command context;
  - joins a following hyphen-leading token only when it is not a recognized
    option;
  - never converts exactly `-` for `--output`;
  - preserves non-UTF-8 values for Clap to classify consistently.
- Keep the post-parse `ArtifactTarget` validation for empty and exact `-`; the
  normalization layer should not duplicate semantic validation.
- Add boundary tests for separate and equals forms, `--help`, `--version`,
  every known following flag, negative-looking query text, exact `-`, missing
  values, and `--` termination.
- Update the public contract only if generated string parameters require a
  narrower rule than output and query.

### Close process and generator gates

- Run the discovery-only consumer against every generated invocation and keep
  the output deterministic.
- Add representative process tests to stable JSON fixtures, while leaving the
  exhaustive matrix assertion-based to avoid a volatile monolithic snapshot.
- Regenerate both SDK and CLI owned trees and prove CLI-only fixture changes do
  not alter the SDK projection checksum.

## Validation

- Run generated freshness and the full keyless discovery/process suite.
- Run `opendart_compat` structured and binary loopbacks for representative
  structured and ZIP invocations after argv normalization.
- Run pinned stable and MSRV formatting, Clippy, tests, rustdoc, package, clean
  install, and repository verification gates.
- Confirm invalid invocations never read a credential or make a network call.

## Completion criteria

- Every generated operation, alias, and representation proves its exact
  prepared SDK identity through a keyless test.
- Nested invocation errors provide help from the deepest valid command prefix.
- Separate-token hyphen-leading output and query values work, while real flags,
  help, and version remain options.
- The command contract stays generated, deterministic, credential-free, and
  safe to expose to agents.

## Next action

Add the exhaustive keyless dispatch matrix and the failing nested-help and
hyphen-leading argv boundary tests before refactoring command context.
