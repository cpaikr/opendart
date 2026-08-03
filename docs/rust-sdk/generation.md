# Rust SDK generation

## Purpose

The private Go generator turns the canonical OpenAPI 3.2 contract into complete,
deterministic Rust request and conservative response code. Consumers compile
the checked-in Rust projection and never need the generator or a repository
checkout.

Delivery state belongs in the
[Public Rust SDK plan](../../plans/rust/public-rust-sdk.md).

This document describes the current SDK and CLI generation pipeline. Under
[ADR 0004](../decisions/0004-handwritten-rust-sdk-conformer.md), SDK wire-code
generation is transitional and will be deleted with the handwritten cutover;
only bounded CLI interface and private dispatch adapter projections may remain.
Until then, these commands and freshness rules remain the implemented
repository contract.

## Inputs and source of truth

Generation loads `openapi/openapi.yaml` and its confined references through
`internal/openapi`. The portable bundle may be compared for provenance, but it
is not a second operation inventory.

The repository-owned SDK surface admits only source facts that affect safe
request construction, response routing, public documentation, or generated
wire shapes:

- path, method, parameters, security, responses, and media types;
- explicit schemas and supported constraints;
- `operationId` and `x-opendart.logicalOperationId`;
- stable group, API, and naming inputs; and
- source evidence required to serialize or classify a representation.

Descriptions and examples can supply generated documentation. They do not
become validation, success policy, domain interpretation, or collection
behavior.

Third-party OpenAPI types remain confined to `internal/openapi`.
`internal/sdkgen/model` owns the deterministic, language-neutral projection.
The model is a private in-memory Go API, not a public or serialized SDK
manifest.

The reviewed manifests in `sdk/rust/interface` are separate product-language
inputs for the handwritten SDK and CLI. They bind OpenAPI identities and
parameter concepts to approved Rust and CLI names without copying paths,
schemas, statuses, media types, or other wire facts. The current SDK projection
does not consume them; the Rust conformance gate checks them against OpenAPI,
while the CLI interface projection consumes only reviewed CLI vocabulary and
the private dispatch projection binds canonical concepts to current generated
SDK symbols.

## Logical and physical operations

Every canonical path and method with one `operationId` is one physical callable
operation. JSON and XML paths with distinct operation IDs remain distinct even
when they share a logical identity. An alternate media type is an outcome of
the same physical operation; an XML source-error response on a ZIP operation
does not create another callable endpoint.

Physical operations with the same logical identity share a public request type
only when their parameter contracts are equivalent. Generation fails if a
shared type would hide a difference in requiredness, serialization,
constraints, authentication, or response routing.

Rust names are deterministic and collision-checked after case conversion and
reserved-word escaping. Generated identity metadata preserves the public Rust
name, physical OpenAPI operation ID, and logical operation ID so compatibility
review can distinguish a source rename from a semantic change.

## Generated request contract

The generator emits constructors, consuming optional builders, representation
preparation methods, deterministic query serialization, authentication
requirements, and operation/projection identity.

Preparation validates the closed set of constraints carried by the canonical
schema:

- required and non-empty inputs;
- string minimum and maximum lengths;
- supported OpenDART formats;
- explicit allowed values;
- decimal-integer minimum and maximum bounds;
- list minimum and maximum cardinality; and
- declared parameter style and explode behavior.

These checks apply to scalar values and array elements as appropriate. A
bounded array consumes at most one element beyond its maximum so oversized or
infinite iterators fail without being exhausted. The normalized model rejects
array bounds that cannot leave room for that sentinel on supported targets.

Unsupported representation choices are absent methods rather than generic
runtime options. Constraints found only in narrative prose are not generated;
promote a reusable rule into the canonical schema or an approved
`x-opendart` extension first.

## Generated response contract

Each structured physical representation receives its own public
`#[non_exhaustive]` wire type. Generated code preserves:

- canonical property names and supported descriptions;
- source-established object and array structure;
- explicit requiredness and scalar types;
- unknown fields where source evolution is possible; and
- opaque `SourceValue` values where the source does not establish a narrower
  type.

Generated fields are public for reading. Additive fields remain accessible
through the generated additional-field interface without permitting exhaustive
external construction or matching. JSON and XML decoders are private and are
bound into the corresponding `PreparedRequest<T>`.

The canonical `OpenDartStatus` schema maps to the open `SourceStatus` wrapper.
Known values are conveniences; unknown future strings remain representable.
Shared status-envelope parsing and ZIP/XML prefix classification stay
handwritten because they cross operation boundaries and must preserve source
uncertainty and consumed bytes.

## Command and transaction

From the repository root, regenerate all Rust product projections with:

```sh
go run ./cmd/opendart-tool generate-sdk \
  --language rust \
  --root openapi/openapi.yaml \
  --interface sdk/rust/interface \
  --output sdk/rust/crates/opendart/src/generated \
  --cli-output sdk/rust/crates/opendart-cli/src/generated
```

The command:

1. Loads and validates the complete canonical document.
2. Builds one semantic model and separately checksummed SDK, CLI-interface, and
   CLI-dispatch projections.
3. Renders all three owned trees into staging directories.
4. Validates their complete content and ownership markers.
5. Replaces no accepted tree until all staged projections are valid.
6. Rolls the set back if accepted-tree replacement fails partway through.

The generator replaces only marked owned subtrees. It never invokes Cargo and
does not hide validation behind Make, Just, npm, shell, or another wrapper.

## Checked-in output and freshness

Generated Rust is reviewed source. Every generated file carries a do-not-edit
marker, generator schema, and deterministic owning-projection checksum. Output
contains no local paths, timestamps, credentials, or environment-dependent
ordering.

The generated SDK module is intentionally declared with `#[rustfmt::skip]`.
Generator freshness owns its compact canonical formatting; `cargo fmt --check`
covers handwritten Rust. Cargo checks, Clippy, tests, and rustdoc still compile
and validate generated code.

`opendart-tool verify` renders all three projections in memory and compares
them byte for byte with the committed trees. Verification is offline and never
rewrites the working tree. The public CLI projection is deliberately free of
Rust input, field, preparation-method, and response-wrapper symbols.

The projection checksum includes only normalized inputs that affect generated
Rust behavior, API, or emitted documentation. Full source provenance is
release-selected handwritten metadata, so an unrelated specification change
does not create projection churn or an SDK release.

Consumer builds must not use `build.rs`, a specification-parsing proc macro, a
Git submodule, a network download, or an environment path back to the canonical
contract.

## Failure policy

Generation fails closed on:

- an unsupported contract construct that affects a public request or response;
- unresolved or non-confined references;
- missing, duplicate, or incompatible operation identities;
- missing, duplicate, stale, ambiguous, or orphan CLI interface mappings;
- Rust name collisions;
- unsupported parameter serialization or constraint evidence;
- ambiguous authentication placement;
- an unrepresentable response shape;
- incomplete canonical-to-generated operation coverage; or
- nondeterministic or unowned output.

There is no fallback generic untyped endpoint. Add narrow model and renderer
support with fixtures, or leave the complete-coverage gate failing until the
contract can be represented safely.

## Future emitters

A later language emitter may reuse the normalized model and language-neutral
request vectors, but it owns idiomatic builders, runtime integration, errors,
package metadata, and release policy. Extract a shared serialized fixture
format only when a second emitter proves the need; generated Rust is never an
input to another language.
