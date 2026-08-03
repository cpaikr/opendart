# Rust SDK public contract

## Purpose

`opendart` is a handwritten protocol SDK. It gives ordinary callers a safe
native client and strict callers the same deterministic request plan before
network I/O. Both paths share reviewed operation identity, validation,
serialization, authorization, and response semantics.

Usage examples live in the
[crate README](../../sdk/rust/crates/opendart/README.md). Delivery state belongs
in the [Public Rust SDK plan](../../plans/rust/public-rust-sdk.md).

## Handwritten operation surface

The `operations` module is divided into semantic families. Each logical
operation has one reviewed input type; each physical representation has one
reviewed preparation method and response contract. Rust names are product
language recorded in `sdk/rust/interface`, not mechanical transformations of
OpenDART or generator identifiers. Exact physical and logical OpenAPI
identities remain available as protocol evidence.

Required source-shaped inputs are constructor arguments. Optional inputs use
consuming `with_*` builders. Private validated values centralize only recurring
invariants proven identical across operations: company-code, compact date,
business-year, and report-code rules. They do not create a second public input
API. Operation preparation still owns operation-local decimal ranges and other
operation-specific or cross-field validation.

Preparation enforces source-backed requiredness, lengths, formats, allowed
values, decimal ranges, list cardinality, cross-field rules, and declared query
serialization. Narrative prose, examples, and defaults do not become runtime
validation unless promoted into the canonical contract or reviewed metadata.
Bounded list inputs inspect at most one item beyond their maximum.

Representation is selected by a distinct `prepare_json`, `prepare_xml`, or
`prepare_archive` method, never a generic runtime selector. JSON and XML remain
distinct physical types even when their documented fields coincide. ZIP
operations retain a separate streaming lifecycle.

## Prepared requests and authorization

`PreparedRequest<T>` is immutable, performs no I/O, and contains no credential.
Its private response contract binds the chosen representation, expected XML
root, and opaque handwritten wrapper `T`. `PreparedBinaryRequest` is separate;
a binary plan cannot be passed to structured execution.

The public evidence includes:

- method and trusted relative path;
- deterministic non-secret query encoding;
- authentication requirement;
- exact physical and logical operation identity; and
- expected response representation.

`interpret_response` is the pure seam shared by the official client and
caller-owned transports. It receives the active key separately so reflected
credential evidence can be removed.

Authorization is explicit:

```rust
let authorized = prepared.authorize(&api_key);
authorized.with_exposed_relative_uri(|relative_uri| {
    executor.execute_once(relative_uri)
})?;
```

`ApiKey` and `AuthorizedRequest` are non-serializable and redact `Debug`.
`AuthorizedRequest` is non-cloneable and has no `Display`; its callback consumes
the value and confines the credential-bearing URI to the caller-owned adapter.
The official client uses the same path.

## Source-complete response wrappers

Every structured physical representation has an opaque handwritten wrapper.
Successful construction validates all established required structure eagerly
and retains the complete normalized `SourceValue`, including unknown fields,
exact source-number lexemes, and future provider values. Typed accessors and
borrowed item views navigate that evidence; `source()` and `field()` preserve
additive source access rather than creating a second response model.

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
`000`, `013`, documented errors, and unknown future strings. `SourceStatus` is
an open string wrapper. Status `013` remains evidence; the SDK does not declare
it successful empty data, retryable, or terminal collection state.

`StatusEnvelope` retains exact source code, optional opaque message, and
complete normalized evidence. A path-aware `ResponseDecodeError` identifies
established structure that a successful envelope violated without retaining an
unsafe body or credential.

## Bounded JSON and XML inspection

`WireInspector` classifies bounded caller-supplied bytes without I/O. JSON and
XML share the open status and normalized source-evidence model.

XML inspection accepts complete bounded UTF-8 XML 1.0. A lexical preflight
enforces declarations, processing-instruction targets, DTD rejection, nesting
depth, and per-element attribute bounds before a maintained safe-Rust parser
validates the document. Custom entities and external resolution are unavailable.
Valid comments, namespaces, mixed content, CDATA, and character references are
preserved within the documented bounds.

Literal CRLF and CR in text and CDATA normalize to LF under XML 1.0; numeric
character references retain the referenced character. The inspector never
applies HTTP content decoding.

## Optional native client

Outside WebAssembly, default feature `client-reqwest` provides one concrete
handwritten execution path:

- `Client::execute` performs bounded JSON or XML interpretation and returns the
  wrapper bound into `PreparedRequest<T>`.
- `Client::execute_binary` handles ZIP operations and alternate XML status
  envelopes.

There is no parallel raw structured execution API. Every successful wrapper
already exposes complete normalized source evidence.

Only 2xx responses can return `SourceResponse`; non-2xx responses return
`ClientError::HttpStatus` with sanitized metadata and safely recognized bounded
evidence. The client performs at most one OpenDART request and owns no retry,
redirect follow, quota wait, successful-empty classification, or persistence.

Binary execution returns `BinaryReply::Archive`, `BinaryReply::Status`, or
`BinaryReply::Unrecognized`. Archive and unrecognized outcomes contain a
fallible replaying `BodyStream`, so classification loses no consumed prefix
bytes. Detailed guarantees belong in
[Transport and safety](transport-and-safety.md).

On WebAssembly the native client types and dependencies are absent while
preparation, authorization, source wrappers, and wire inspection remain
available.

## Optional JSON serialization

With `serde-json`, structured wrappers, `SourceResponse`, `SourceReply`, status
envelopes, sanitized metadata, and `SourceValue` implement `serde::Serialize`.
Wrapper serialization delegates directly to retained `SourceValue`; typed
accessor fields are never serialized as a second model.

The serialized form is source-shaped: absent optional source fields remain
absent, explicit null remains null, additive fields remain in their source
object, statuses retain exact strings, and sanitized header values remain exact
byte arrays. Credentials, requests, clients, and streams remain
non-serializable.

`SourceValue::number` accepts exactly one JSON number with no surrounding
whitespace and imposes no magnitude, decimal, or exponent range. Direct JSON
encoding emits the retained lexeme exactly, including `-0`, exponent spelling,
and fractional trailing zeros. Passing through `serde_json::Value`, a Rust
numeric type, or another Serde data model is outside that guarantee.

## Error boundary

Public errors are focused and sanitized:

- private validation and `PrepareError` categories identify the affected
  concept without retaining rejected values;
- authorization errors never echo the credential;
- body-limit, envelope, inspection, and decode errors identify bounded failure
  classes and safe paths;
- caller-owned interpretation exposes `ResponseInterpretError` with only
  credential-safe evidence; and
- native client and body-stream errors retain only sanitized operation,
  failure, and metadata context.

No public error contains a raw dependency error, authenticated URL, unbounded
response body, or retry recommendation.

## Policy and compatibility boundary

The crate owns source-protocol mechanics. It does not own collection
coordinates, request footprints, persistence, retry scheduling, quota policy,
closure, domain identifiers, successful-empty policy, or business
interpretation. Callers needing proxies, custom DNS or TLS, alternate decoding,
instrumentation, or other origins use prepared requests and own those policies;
the SDK does not publish a transport trait or accept an arbitrary client while
claiming its native safety guarantees.

Review handwritten changes as Rust API and request-behavior changes:

- adding an operation or optional builder is normally additive;
- adding a required input, changing serialization, or renaming/removing a
  public operation is breaking;
- status constants are additive because `SourceStatus` remains open;
- additive response fields remain compatible through opaque source-backed
  wrappers;
- JSON and XML wrappers remain distinct; and
- narrowing an uncertain scalar requires new evidence and compatibility review.

OpenAPI is the only complete wire authority. Reusable source behavior belongs
there or in approved metadata before the handwritten conformer implements it;
there is no SDK generator, compatibility selector, or fallback untyped endpoint.
