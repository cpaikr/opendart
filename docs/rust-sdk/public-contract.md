# Rust SDK public contract

## Purpose

`opendart` is a protocol SDK. It gives ordinary callers a safe native client
and gives strict callers the same deterministic request plan before network
I/O. Both paths share operation identity, validation, serialization,
authorization, and response-envelope semantics.

Usage examples live in the
[crate README](../../sdk/rust/crates/opendart/README.md). Delivery state belongs
in the [Public Rust SDK plan](../../plans/rust/public-rust-sdk.md).

This is the current generated SDK contract. The accepted handwritten-only
replacement is defined by [ADR 0004](../decisions/0004-handwritten-rust-sdk-conformer.md)
and its linked plans. Do not read target names or target removal decisions into
this contract before the implementation cutover updates both together.

## Generated operation surface

The supported `operations` module contains one idiomatic request type for each
logical OpenDART operation. Required inputs are constructor arguments; optional
inputs use consuming builders over private fields. JSON and XML remain distinct
physical representations even when they share a logical request type.

Preparation enforces the constraints carried by the canonical schema:

- required and non-empty values;
- minimum and maximum string lengths;
- supported OpenDART formats;
- explicit allowed values;
- decimal-integer ranges;
- list cardinality; and
- declared parameter serialization, including comma-joined
  `style: form`, `explode: false` arrays.

Array builders inspect at most one element beyond the declared maximum, so an
oversized or infinite iterator fails without being exhausted. Narrative prose,
examples, and defaults do not become runtime validation unless the rule is
promoted into the canonical schema or an approved extension.

Every structured physical representation has its own generated success type.
These types are output-only, `#[non_exhaustive]` wire evidence: documented
fields are public for reading, additive fields remain accessible through the
generated additional-field interface, and external code cannot construct or
match the shape exhaustively. Unknown scalar kinds remain `SourceValue` until
the source contract supports a narrower type.

The generated operation names and behavior are supported public API. Generated
file and private module layout are not.

## Prepared requests and authorization

`PreparedRequest<T>` is immutable, performs no I/O, and contains no credential.
Its private response contract binds the chosen representation, XML root, and
generated success type `T`. `interpret_response` is the common pure seam for
the official client and caller-owned transports; it receives the authorization
key separately so reflected credential evidence can be removed. Narrow getters
expose:

- method and trusted relative path;
- deterministic non-secret query encoding;
- authentication requirement;
- physical and logical operation identity;
- expected response representations; and
- generator schema and projection identity.

Fixed ZIP operations return `PreparedBinaryRequest`; a binary plan cannot be
passed to structured execution.

Authorization is one explicit boundary:

```rust
let authorized = prepared.authorize(&api_key);
let raw = executor.execute_once(authorized).await?;
```

`ApiKey` and `AuthorizedRequest` are non-serializable and redact `Debug`.
`AuthorizedRequest` is non-cloneable and has no `Display`. Its
`with_exposed_relative_uri` method consumes the value and exposes the
credential-bearing URI only inside a caller-owned adapter callback. The
official client uses the same path; no generated endpoint appends a key
independently.

## Source evidence

The SDK separates representation evidence from application policy:

```rust
#[non_exhaustive]
pub enum SourceReply<T> {
    Success(T),
    Status(StatusEnvelope),
}

#[non_exhaustive]
pub struct SourceResponse<T> {
    pub metadata: ResponseMetadata,
    pub reply: T,
}
```

A recognized status-only envelope becomes `SourceReply::Status`, including
`000`, `013`, other documented codes, and unknown future strings. Generated
success payloads retain embedded status fields in their conservative wire
shape. `SourceStatus` is an open string wrapper; known constants and predicates
do not close the set.

Status `013` remains source evidence. The SDK does not declare it a successful
empty collection, retryable failure, or terminal collection result. It also
does not infer those policies from `Content-Type`. HTTP status is handled
separately: every non-2xx result is an explicit typed failure, optionally
carrying normalized bounded body evidence.

`StatusEnvelope` retains the exact source code, optional opaque message, and
complete normalized evidence. `SourceValue` preserves null, Boolean, number,
string, array, and object values plus unknown fields and scalar uncertainty.
Generated decoders return a path-aware `ResponseDecodeError` when a successful
envelope violates an established required shape.

## Bounded JSON and XML inspection

`WireInspector` classifies caller-supplied bytes without performing I/O. JSON
and XML share the same open status and `SourceValue` model.

XML inspection accepts bounded UTF-8 XML 1.0 only. A lexical preflight enforces
supported declaration values, processing-instruction targets, DTD rejection,
nesting depth, and a per-element attribute bound. A maintained safe-Rust parser
then validates the complete document before the event converter constructs
source values from the unchanged bytes. Malformed or truncated XML never
becomes authoritative status evidence.

An XML declaration, when present, must use version `1.0`; its encoding must be
`UTF-8`, and standalone must be `yes` or `no`. Custom entities, DTDs, and
external resolution are unavailable. Valid comments, processing instructions,
namespaces, mixed content, CDATA, and character references remain supported
within the configured bounds.

Literal CRLF and CR in text and CDATA normalize to LF as required by XML 1.0.
A numeric character reference such as `&#13;` remains the referenced CR.

The inspector never applies HTTP content decoding. A caller that owns transport
may preserve exact entity bytes before providing an explicitly decoded view.

## Optional native client

Outside WebAssembly, the default `client-reqwest` feature provides one
handwritten execution path over the same prepared requests:

- `Client::execute` buffers and classifies JSON or XML, then decodes the
  generated success type bound into `PreparedRequest<T>`.
- `Client::execute_raw` performs the same bounded request and envelope
  classification but returns the normalized `SourceValue` success payload.
- `Client::execute_binary` handles fixed ZIP operations with alternate XML
  status envelopes.

Only 2xx responses can return `SourceResponse`; non-2xx responses return
`ClientError::HttpStatus` with sanitized metadata and safely recognized body
evidence. The client returns `SourceResponse` so sanitized HTTP metadata remains available
beside the source reply and after supported failure paths. It performs at most
one OpenDART request and implements no retry, redirect follow, quota wait,
success-empty classification, or persistence behavior.

Binary execution returns `BinaryReply::Archive`,
`BinaryReply::Status`, or `BinaryReply::Unrecognized`. Archive and
unrecognized outcomes contain a fallible replaying `BodyStream`, so bytes read
during classification are not lost. Detailed HTTP and binary guarantees belong
in [Transport and safety](transport-and-safety.md).

On WebAssembly, `client-reqwest` is accepted but inert: native client types and
dependencies are absent while preparation, authorization, and wire inspection
remain available.

## Optional JSON serialization

With `serde-json`, generated response types, `SourceResponse`, `SourceReply`,
status envelopes, sanitized metadata, and `SourceValue` implement
`serde::Serialize`.

The serialized form is source-shaped:

- generated field names remain source names;
- absent optional fields are omitted while explicit source null remains null;
- additive fields flatten into their owning object;
- `SourceStatus` is its exact string;
- header values remain exact byte arrays; and
- `SourceReply` uses adjacent `kind` and `value` fields with `success` and
  `status` variants.

Credentials, authorized and prepared requests, clients, and body streams remain
non-serializable.

`SourceValue::number` accepts exactly one JSON number with no surrounding
whitespace:

```text
number = ["-"] integer [fraction] [exponent]
integer = "0" | nonzero-digit *digit
fraction = "." 1*digit
exponent = ("e" | "E") ["+" | "-"] 1*digit
digit = "0".."9"
nonzero-digit = "1".."9"
```

There is no leading `+`, leading zero before another integer digit, `NaN`, or
infinity spelling. The SDK imposes no magnitude, decimal, or exponent range.
Direct `serde_json` text encoding emits the retained lexeme exactly, preserving
details such as `-0`, exponent case and sign, and fractional trailing zeros.
Passing through `serde_json::Value`, a Rust numeric type, or an arbitrary Serde
format is outside this guarantee.

## Error boundary

Public errors are focused and sanitized:

- `PrepareError` reports invalid generated input without retaining its value.
- `AuthorizationError` reports invalid or missing credential structure without
  echoing the secret.
- `BodyLimitError`, `EnvelopeError`, and `WireInspectError` report bounded
  classification failure.
- `InvalidSourceNumberError` rejects a caller-created number before a
  `SourceValue` exists.
- `ResponseDecodeError` identifies the generated field path that violated an
  established source shape.
- Caller-owned response interpretation returns `ResponseInterpretError`; its
  `HttpStatus` variant retains only bounded, credential-safe evidence.
- Native `ClientBuildError`, `ClientError`, `TransportError`, and
  `BodyStreamError` retain only sanitized operation, failure, and metadata
  context.

No public error contains a raw `reqwest::Error`, authenticated URL, response
body, or retry recommendation. The transport-independent core has no transport
error type in its execution path.

## Policy and escape boundary

The crate owns source-protocol mechanics. It does not own collection
coordinates, request footprints, artifact persistence, retry scheduling,
quotas, closure, domain identifiers, successful-empty policy, or business
interpretation.

Callers needing explicit proxies, custom DNS or TLS, alternate decoding,
instrumentation, or unrestricted origins use prepared requests with their own
executor. The official client does not accept an arbitrary caller-built
`reqwest::Client` while claiming its safety guarantees. The low-level API also
does not require a public `Transport` trait.

An undocumented endpoint or serialization rule may begin as caller-local
experimental code. Reusable source behavior belongs in the canonical
specification or approved metadata and then in generated code.

## Compatibility policy

Review generated changes as Rust API and request-behavior changes, not merely
as OpenAPI diffs:

- Adding an operation or optional builder is normally additive.
- Adding a required input, changing serialization or operation meaning, or
  removing or renaming a public operation is breaking.
- Adding a recognized status constant is additive because `SourceStatus`
  remains open.
- Adding response fields is compatible because response structures are
  non-exhaustive and retain additive evidence.
- JSON and XML success types remain distinct even when their fields currently
  match.
- Replacing an opaque scalar with a narrower type requires evidence and
  compatibility review.
- Stable operation, request, response, error, and behavior contracts
  participate in SemVer; generated file layout does not.
