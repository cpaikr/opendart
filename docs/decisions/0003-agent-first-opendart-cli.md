# ADR 0003: Add an agent-first OpenDART CLI

- Status: accepted
- Date: 2026-07-20

## Context

The public Rust SDK provides complete typed request preparation and conservative
response decoding, but shell users still have to write Rust to call an
operation. The intended CLI users are primarily LLM agents, which need
deterministic discovery, structured output, strict input validation, and useful
exit codes.

A separately modeled CLI would duplicate the endpoint inventory and response
types. A generic raw endpoint runner would avoid that duplication but bypass
the SDK's logical operation types and typed response decoders. Extending the
private Go `opendart-tool` would mix a maintainer command with a public product
that has different credential, output, compatibility, and release obligations.

## Decision

Add a separate public Cargo package named `opendart-cli` with an `opendart`
binary. It joins the existing Rust workspace and depends on the public
`opendart` crate; it does not extend `cmd/opendart-tool` or expose a supported
Rust library interface.

Generate one CLI command for every logical SDK operation from the same private
normalized model that generates the SDK. A command uses operation-specific
flags, constructs the public SDK input type, selects the applicable representation,
prepares the request, and calls the typed SDK client. Logical IDs remain stable
aliases for SDK-derived kebab-case command names. Generated discovery exposes
the same identities, parameters, constraints, representations, and response
shapes without parsing OpenAPI at runtime.

Add an optional `serde-json` feature to the SDK. It supplies a deliberate
JSON serialization contract for generated response types and shared
source-response types while leaving credentials, authorized requests, prepared
requests, and clients non-serializable. Unknown fields remain available, and
`SourceValue` uses natural JSON values. The SDK owns the accepted JSON-number
grammar and rejects invalid public construction before a `SourceValue` exists.
A valid source number is emitted from its exact retained lexeme through
`serde_json::value::RawValue`, without an integer or floating-point range
conversion. The CLI serializes the typed SDK response directly rather than
creating endpoint-specific mirror types.

Use a small CLI-owned envelope to add operation identity and a stable outcome
discriminant around the complete typed SDK response. Emit compact JSON as the
only structured stdout format. Do not add a format selector or make the CLI own
a second response notation. JSON keeps the process contract aligned with the
SDK's source model and existing arbitrary-precision parser boundary.

Follow the agent-facing parts of [AXI](https://axi.md/): non-interactive strict
arguments, machine-readable discovery, structured errors on stdout, and exit
codes `0`, `1`, and `2` for success, execution failure, and usage failure.
Deliberately depart from AXI's TOON, field-selection, and truncation guidance.
The CLI is a protocol mirror: source fidelity, SDK-shaped types, and standard
JSON tooling take precedence over an unproven token optimization. Callers
control source pagination and may filter JSON downstream when they need less
data. A status-only SDK reply, including `000` or `013`, remains intact but
exits `1`; the CLI does not promote source status into successful-empty policy.

Do not install ambient session hooks in the initial CLI. It has no
directory-scoped live state to preload, and an implicit authenticated request
would violate explicit operation selection. The keyless structured home and
discovery commands provide the appropriate on-demand agent orientation.

ZIP operations require an explicit destination. Archive and unrecognized
streams are preserved exactly through an atomic no-clobber write, while stdout
contains metadata and an artifact result. Every binary call has a finite
CLI-owned default byte budget of 512 MiB (`536870912` bytes) and may override
it with a positive `--artifact-limit-bytes` value. The CLI does not extract
archives.

The CLI and SDK have independent SemVer and release components. Implement and
review the SDK serialization consumer before the first SDK publication, publish
and verify `opendart` first, and only then publish `opendart-cli`. Initial CLI
distribution is through crates.io with source-install support for Linux, macOS,
and Windows; prebuilt GitHub release artifacts are a separate hardening task.
The CLI exact-pins the reviewed SDK version. Every SDK version-bump change
updates the same-workspace CLI path-and-version pin and lockfile before merge,
without making the CLI release-eligible by itself; CLI packaging waits until
that SDK version is published and verified.

## Consequences

- Endpoint additions and compatible request changes flow through one normalized
  model and fail verification if either generated Rust product is stale.
- SDK serialization becomes a supported optional public interface and requires
  compatibility tests; automatic `Serialize` derivation is insufficient for
  source wrappers and enum discriminants.
- CLI presentation text may evolve independently, but it cannot add operations
  or change typed request and response facts.
- The only response adaptation is CLI-owned process evidence, such as operation
  identity, errors, and a file reference replacing a consumed binary stream.
- JSON number spelling, grammar, and invalid-value behavior are SDK-owned
  compatibility commitments rather than properties selected by a CLI encoder.
- Agents gain universal JSON parsers and shell tooling but do not receive
  TOON's workload-dependent token reductions from the CLI itself.
- The CLI exposes SDK timeout and envelope limits but adds no retry, proxy,
  alternate origin, pagination, or domain policy.
- Registry publication remains separately authorized and recoverable. Neither
  this decision nor implementation grants a workflow crates.io credentials.

## Alternatives considered

- A raw physical-endpoint command would preserve source bytes but bypass typed
  SDK responses and make logical operations less discoverable.
- CLI-owned response schemas would decouple releases at the cost of permanent
  conversion and drift.
- TOON-only or dual-format output could reduce tokens for sufficiently uniform
  tables, but would add a second compatibility surface and could not naturally
  preserve the SDK's arbitrary-precision source numbers through the selected
  Rust encoder. Its benefit over compact JSON is not established for the real
  OpenDART response corpus.
- Lockstep SDK and CLI versions would communicate coupling while forcing
  unrelated releases and obscuring their distinct compatibility surfaces.

## Related work

- [Target CLI architecture](../rust-cli/architecture.md)
- [Public CLI contract](../rust-cli/public-contract.md)
- [Public CLI implementation plan](../../plans/rust/public-opendart-cli.md)
- [ADR 0002: Add a first-party Rust SDK](0002-public-rust-sdk.md)
