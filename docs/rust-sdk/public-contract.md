# Rust SDK Public Contract

Planning source: [Public Rust SDK](../../tasks/rust/public-rust-sdk.md).

## Purpose

Expose a small reusable protocol interface that supports both ordinary SDK
callers and projects that require complete control over HTTP execution. The
advanced interface stops after deterministic request construction and before
network I/O.

## Interface layers

### Generated operation input

Each logical OpenDART operation has an idiomatic request type. Required inputs
are constructor arguments; optional inputs use consuming builders over private
fields. Narrow getters expose request state when useful. JSON and XML remain
distinguishable physical representations while sharing logical identity.

Generated response structures are output-only wire evidence. Mark them
`#[non_exhaustive]` and expose documented fields for reading without permitting
external exhaustive construction or pattern matching. This keeps later
source-supported response fields additive.

Illustrative, non-final naming:

```rust
let operation = AuditorOpinion::new(corp_code, business_year, report_code);
let prepared = operation.prepare(Representation::Json)?;
```

Generation enforces only constraints supported by the canonical contract.
Narrative length, enum, date, range, and default text remains documentation
unless stronger evidence has promoted it into the specification. The documented
multi-company cardinality and comma-separated serialization are enforceable
because they are explicit contract facts.

### Prepared request

`PreparedRequest` is immutable, performs no I/O, and contains no credential. It
privately holds:

- HTTP method;
- trusted relative path;
- deterministically encoded non-secret query parameters;
- authentication requirement;
- physical OpenAPI and stable logical operation identities;
- expected response representations; and
- generator schema and SDK-projection identity needed for safe diagnostics.

Expose narrow getters rather than public fields so generated layout and URL
library choices do not become permanent API. `Debug` may show sanitized
operation and target metadata but never reserve a slot where a credential can
later appear.

### Authorization

Authorization adds `crtfc_key` only at the private execution boundary:

```rust
let authorized = prepared.authorize(&api_key);
let raw = executor.execute_once(authorized).await?;
```

`ApiKey` and `AuthorizedRequest` must be non-serializable and redact `Debug`.
`AuthorizedRequest` should be non-`Clone` unless a concrete caller proves that
cloning is required. Do not implement `Display`. Any method exposing the actual
credential-bearing URI is explicit, narrowly scoped to adapter code, and
documented as secret exposure.

The public high-level client consumes the same authorization path internally.
No endpoint method appends a key independently.

### Wire inspection

Wire inspection accepts response metadata and caller-supplied bytes or a
bounded reader. It produces source evidence, not collection policy:

```rust
pub enum SourceReply<T> {
    Success(T),
    Status(StatusEnvelope),
}

#[non_exhaustive]
pub struct SourceResponse<T> {
    pub metadata: ResponseMetadata,
    pub reply: T,
}

pub struct StatusEnvelope {
    pub code: SourceStatus,
    pub message: Option<SourceValue>,
}
```

This sketch describes evidence variants, not Rust error conversion. The
low-level inspector returns every recognized source code through
`SourceReply::Status`, including `000`, `013`, other documented codes, and
unknown future strings when they occur in a status-only envelope. A generated
success payload retains any embedded status fields in its conservative wire
shape. The inspector uses `Err` only for inability to read or safely classify
the representation.

The ergonomic client likewise returns a shape equivalent to
`Result<SourceResponse<SourceReply<T>>, ClientError>`; it does not turn `013`,
another known source code, or an unknown source code into a Rust error.
`ResponseMetadata` is a repository-owned, non-exhaustive value that preserves
HTTP status, version, and response headers after removing credential-bearing
values. Header names and safe values remain available for caller-owned policy;
raw authenticated redirect locations and the API key in literal or encoded
form never cross the public boundary. A future opt-in helper may offer a
conventional interpretation, but it must return the original response and
cannot define collection-specific successful-empty or retry policy.

`SourceStatus` preserves unknown future strings. Known values may be associated
constants or predicates, not a closed enum. Status `013` remains source status
evidence; the SDK does not label it a successful empty collection.

JSON/XML envelope parsing and ZIP/XML error recognition are handwritten shared
runtime behavior. Parsers operate only on bounded inputs, do not resolve XML
external entities, and enforce practical nesting and expansion limits.
Generated success shapes preserve unknown fields and use an opaque
`SourceValue` where scalar type is not established. They are wire types, not
domain types.

The inspector never automatically decodes HTTP content encodings. A strict
caller may first preserve the exact entity bytes and then provide an explicitly
decoded interpretation view.

ZIP success bodies remain opaque streamed bytes. The initial crate does not
extract archives; any later archive helper requires explicit entry, expanded-
size, nesting, and path-safety limits.

For a callable operation whose normal result is ZIP but whose alternate source
error is XML, the ergonomic client performs bounded prefix inspection and
returns a shape equivalent to:

```rust
pub enum BinaryReply<S> {
    Archive(S),
    Status(StatusEnvelope),
    Unrecognized(S),
}
```

The ergonomic return shape is
`Result<SourceResponse<BinaryReply<BodyStream>>, ClientError>`.

`S` is a replaying stream. Any bytes consumed while distinguishing ZIP from
XML are prepended to the remaining body, so `Archive` and `Unrecognized`
yield every successfully read entity byte in order. The concrete public stream
is fallible, with an item shape equivalent to `Result<Bytes, BodyStreamError>`;
a read failure, configured read timeout, configured total deadline, or
incomplete body is a terminal sanitized stream error rather than end-of-stream.
The outer `SourceResponse` retains response metadata after such a failure.
Caller-owned byte, storage, and workflow budgets remain outside the HTTP
client. `Content-Type` is evidence, never the sole discriminator. XML is
buffered only up to the configured envelope limit. A truncated, ambiguous,
malformed, or oversized candidate becomes `Unrecognized` with all consumed
bytes replayed rather than an error that silently discards the body. Strict
callers may bypass discrimination entirely, persist the raw stream, and inspect
a second view.

Classify a body as `Archive` only when its first four entity bytes are a normal
ZIP local-file header (`PK\x03\x04`) or the end-of-central-directory signature
for an empty archive (`PK\x05\x06`). Split/spanned markers, self-extracting
preambles, ZIP64-only prefixes, truncated signatures, and other binary bodies
remain `Unrecognized` in the initial crate. This is positive representation
evidence, not full archive validation; the SDK still does not extract entries.

### Ergonomic client

The optional client executes the same generated operation inputs used by the
advanced seam. It owns bounded buffering for JSON/XML and a streaming interface
for ZIP or other binary responses without duplicating endpoint methods.

```rust
let client = opendart::Client::builder(api_key)
    .connect_timeout(connect_timeout)
    .read_timeout(read_timeout)
    .total_timeout(total_timeout)
    .build()?;

let prepared = operation.prepare(Representation::Json)?;
let reply = client.execute(&prepared).await?;
```

Callers select JSON or XML explicitly during preparation. Fixed ZIP operations
use `Client::execute_binary` and do not pretend to have a structured
alternative. Operation-specific request types remain the discoverable,
generated API; client execution remains one stable handwritten path.

## Invariants

- Preparing, inspecting, and reading operation metadata perform no network I/O.
- One high-level call invokes its HTTP adapter at most once. It performs no
  automatic retry or redirect follow.
- Request authorization is centralized and never included in safe diagnostics.
- HTTP status and `Content-Type` alone do not determine OpenDART success.
- Unknown source statuses, response fields, and permitted scalar forms remain
  representable.
- No public error carries a raw `reqwest::Error`, because it may retain a URL.
- No error is marked retryable. Retry policy requires application context.
- Generated wire values never become application domain identifiers, dates,
  amounts, or enums without a separate evidence-backed contract change.
- The low-level API does not expose or require a public `Transport` trait.

## Errors

Use focused, non-exhaustive error categories with sanitized context:

- `PrepareError`: missing input, invalid shape, explicit cardinality, invalid
  trusted-target construction, or unsupported representation.
- `AuthorizationError`: structurally invalid or missing credential without
  echoing its value.
- `TransportError`: sanitized connection, timeout, TLS, body-read, or protocol
  failure. It does not claim retryability or send certainty not actually known.
  Errors occurring after response headers are received retain sanitized
  `ResponseMetadata`.
- `BodyLimitError`: configured buffered-body limit exceeded while preserving
  response metadata.
- `EnvelopeError`: malformed or unrecognized source-envelope syntax.
- `DecodeError`: malformed declared success representation or unsupported wire
  value.

The transport-independent core has no `TransportError`. The optional client
maps private `reqwest` errors into the public sanitized taxonomy.

## Supported customization and escape path

The official client exposes configuration that preserves its invariants:

- connect, read, and total timeouts;
- buffered response limits, while streaming budgets and sink behavior remain
  caller-owned;
- user-agent suffix or application identity;
- representation selection; and
- narrowly reviewed TLS trust configuration if a real consumer requires it.

It does not expose switches to enable retries, redirects, ambient proxies,
automatic content decoding, unbounded buffering, or unrestricted origins.

Do not accept an arbitrary caller-built `reqwest::Client` through the API that
claims safe-default guarantees. Callers needing explicit proxies, custom DNS,
special connectors, different TLS policy, or transport instrumentation use
`PreparedRequest` and their own executor. This is the supported escape path,
not a fork and not an all-or-nothing SDK decision.

If a caller needs an undocumented endpoint or serialization rule, it may keep a
local experimental request beside the SDK. Once evidence establishes reusable
source behavior, update the canonical specification or `x-opendart` metadata
and regenerate. Permanent source-contract differences do not belong in a
consumer-only fork.

## Compatibility policy

Treat these as public Rust API compatibility changes:

- Adding an operation or an optional request builder over private state is
  normally additive.
- Adding a required request input, changing serialization, removing or
  renaming an operation, or changing operation meaning is breaking.
- Adding a recognized status constant is additive because `SourceStatus`
  remains open.
- Adding response fields remains compatible because generated structures are
  non-exhaustive, retain unknown fields, and cannot be constructed or matched
  exhaustively outside the crate.
- Replacing an opaque scalar with a narrower type requires evidence and a
  compatibility review; it is not automatically a fix.
- Generated module layout is private where possible. Stable operation and
  request names are public and participate in SemVer.

## Acceptance criteria

- An ordinary caller can make an API call without understanding transport
  internals.
- A strict caller can prepare and authorize the same request, then execute it
  without using SDK transport code.
- Both paths share operation identity, validation, serialization, and
  credential placement.
- Public types preserve unknown statuses, fields, and scalar uncertainty.
- Fixtures prove that `000`, `013`, another documented source error, and an
  unknown code remain source evidence through both low-level and ergonomic
  paths rather than being collapsed into application policy.
- Ergonomic replies preserve sanitized HTTP status, version, and headers, and
  mid-stream failures remain observable without discarding that metadata.
- ZIP discrimination either returns a recognized source status or a stream
  that replays every consumed entity byte.
- No application retry, persistence, collection, or domain-policy type appears
  in the crate.
- Transport customization that would weaken guarantees is routed through the
  prepared-request seam and caller-owned executor.
