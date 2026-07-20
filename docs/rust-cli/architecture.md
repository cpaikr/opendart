# OpenDART CLI target architecture

This is the approved target shape for the planned public Rust CLI. The current
repository architecture is documented in [the root architecture](../../ARCHITECTURE.md),
and implementation status lives only in the
[CLI plan](../../plans/rust/public-opendart-cli.md).

## Purpose and boundaries

The CLI gives shell agents one non-interactive call path for every logical
operation already exposed by the Rust SDK. It owns command parsing, process
output, credentials sourced from the environment, application exit policy, and
binary artifact writes.

The CLI does not own endpoint facts, response wire types, HTTP mechanics,
source-status meaning, retry, pagination, archive extraction, persistence, or
collection policy. Those remain in the canonical contract, SDK, or caller.

## System relationship

```text
canonical OpenAPI 3.2
    -> private OpenAPI projection
    -> normalized SDK model
         |-> SDK Rust renderer -> opendart generated types
         `-> CLI Rust renderer -> opendart-cli generated commands

agent invocation
    -> handwritten CLI runtime
    -> generated command parse and typed SDK input construction
    -> SDK prepare_* and Client execution
    -> typed SourceResponse
    -> CLI outcome envelope
         |-> compact JSON on stdout
         `-> exact binary stream to an explicit artifact path
```

The normalized model is the deep module at the generation seam: emitters learn
one validated logical/physical operation model rather than OpenAPI parser types.
The public SDK is the runtime seam: the CLI does not reproduce request encoding,
authorization, response classification, or typed decoding.

## Modules

### Private generation

`internal/openapi` continues to confine third-party OpenAPI types.
`internal/sdkgen/model` owns normalized operation, parameter, representation,
response-shape, identity, and presentation metadata. Product-specific
projections ensure a CLI-only description change does not rewrite or release
the SDK.

The Rust generation orchestrator renders and verifies two independently owned
trees from one model:

- the existing SDK generated subtree; and
- `sdk/rust/crates/opendart-cli/src/generated`, containing the command catalog,
  discovery records, typed input construction, and dispatch arms.

Both trees are staged and validated before either is replaced. An optional
presentation overlay may change descriptions or examples only. It is keyed by
logical ID and fails closed on unknown IDs, duplicate keys, type facts, or
operation inventory changes.

### SDK JSON serialization

The optional SDK `serde-json` feature serializes generated structured responses
and the shared `SourceResponse`, `SourceReply`, status, metadata, and
source-value types for direct `serde_json` text encoding. Stable adjacent tags
distinguish open reply variants. Additive response fields flatten into their
source object in deterministic order. Absent optional fields remain absent,
source null remains null, and raw sanitized header values remain byte arrays
rather than undergoing a lossy text conversion.

A source number is validated when its `SourceValue` is constructed and emitted
from its retained lexeme through `serde_json::value::RawValue`. It never passes
through `serde_json::Value`, a fixed-width integer, or a floating-point value.
This is an SDK-owned JSON contract, not a promise that arbitrary Serde formats
share JSON's arbitrary-precision number model.

Credential-bearing and executable values remain outside this interface.
`ApiKey`, authorized requests, prepared requests, body streams, and clients do
not become serializable merely because the feature is enabled.

### Handwritten CLI runtime

The published package is a binary product, not a public Rust library. A thin
`main` delegates to internal modules with these responsibilities:

- build and strictly parse the generated `clap` command tree;
- expose keyless home and operation-discovery output;
- construct and prepare the selected typed SDK request before credential access;
- resolve `OPENDART_API_KEY` only after invocation validation and preparation;
- apply only explicitly supplied SDK client-setting overrides;
- execute generated structured or binary dispatch;
- encode one stable JSON outcome; and
- perform same-directory atomic no-clobber artifact writes.

The command builder uses `clap`'s builder interface because the command tree is
generated. Parsing uses non-exiting methods so dependency diagnostics can be
translated into the CLI's structured error envelope.

### Generated CLI dispatch

Generated code owns the exhaustive operation match. Each structured arm creates
the public SDK operation type and calls its representation-specific preparation
method. A handwritten generic executor accepts the resulting
`PreparedRequest<T>` where `T` is serializable, calls `Client::execute`, and
emits the typed response. Binary arms use `PreparedBinaryRequest` and the shared
artifact module.

This keeps endpoint breadth behind a small handwritten interface. No dynamic
parameter map, endpoint-specific adapter, or erased response registry becomes a
second public model.

## Runtime flows

### Structured call

1. Parse and validate the operation, flags, representation, and client-override
   syntax without network access.
2. Construct the generated SDK input and prepare the selected physical request,
   surfacing SDK-backed input violations as usage failures.
3. Read and validate `OPENDART_API_KEY`; apply client overrides and build the SDK
   client, which validates their representable range.
4. Execute once through `Client::execute`.
5. Pair operation identity with the complete typed response and encode it once.
6. Exit from the reply class without changing its contents.

### Binary call

1. Parse the invocation, construct the generated SDK input, prepare the binary
   request, and require a destination that does not exist.
2. Read and validate `OPENDART_API_KEY`, build the SDK client, and create a
   temporary file in the destination directory before network access.
3. Execute once through `Client::execute_binary` and preserve response metadata.
4. For `Archive` or `Unrecognized`, count and stream every body chunk into the
   temporary file within the selected finite artifact budget, then publish it
   without clobbering the destination.
5. Emit an artifact reply with the SDK classification, path, and byte count.
6. Remove the temporary file on a source `Status` or any read or write failure;
   no such path publishes a destination.

## Target code map

- `internal/sdkgen/model` — shared normalized input and collision validation.
- `internal/sdkgen/rust` — orchestration and product-specific Rust renderers.
- `sdk/rust/crates/opendart` — typed request, response, serialization, and HTTP
  interfaces consumed by the CLI.
- `sdk/rust/crates/opendart-cli/src/generated` — one generator-owned command and
  dispatch subtree.
- `sdk/rust/crates/opendart-cli/src` — handwritten parsing, execution, output,
  credential, and artifact modules.
- `sdk/rust/crates/opendart-cli/tests` — process-level contract, loopback, and
  opt-in live smoke tests.

These paths describe the target starting points, not current implementation.

## Invariants

- The canonical OpenAPI contract remains the only endpoint inventory.
- Every logical SDK operation resolves through both its SDK-derived CLI name and
  exact logical ID, with generation-time collision checks.
- Generated dispatch calls public SDK preparation and typed execution; it does
  not reproduce request or response mechanics.
- Discovery is credential-free and has no network side effect.
- Home and operation-detail discovery expose structured argument vectors,
  credentials, global controls, representation selection, artifact requirements,
  and additive response-shape behavior. An agent can reach SDK preparation
  without parsing prose or invoking human help.
- Every binary invocation has a finite CLI-owned default byte budget and may
  override it with a positive `--artifact-limit-bytes` value. The default is
  512 MiB (`536870912` bytes); implementation cannot treat omission as
  unbounded permission.
- Argument parsing, typed input construction, and SDK request preparation finish
  before credentials are read. SDK client-setting validation follows key binding
  because the current builder owns the key, but still precedes network access.
- One CLI call performs at most one SDK HTTP attempt.
- A completed structured stdout write contains one document and no progress
  text. Safe stderr output is opt-in diagnostics only.
- CLI-owned output and errors never expose an API key, authorized URL, raw
  dependency error, unsafe header, or unbounded source body. Typed source
  payloads and exact binary artifacts remain unredacted source evidence.
- Structured source responses remain complete. The CLI adds no silent filtering
  or successful-empty interpretation.
- A binary destination is explicit, exact, atomic, and never overwritten.
- The packaged source distribution is verified natively on Linux, macOS, and
  Windows. Prebuilt target and platform-version policy is a separate boundary.
- Generated SDK and CLI outputs are committed, deterministic, and verified
  offline; consumer builds run neither Go nor OpenAPI parsing.

## Related decisions and contracts

- [ADR 0003](../decisions/0003-agent-first-opendart-cli.md)
- [Public CLI contract](public-contract.md)
- [Rust SDK public contract](../rust-sdk/public-contract.md)
- [Rust SDK generation](../rust-sdk/generation.md)
