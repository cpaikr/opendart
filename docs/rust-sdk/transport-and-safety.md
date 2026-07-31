# Rust SDK transport and safety

## Purpose

This document defines the guarantees of the optional native `reqwest` client:
one source interaction, fixed transport policy, bounded interpretation,
credential-safe diagnostics, and exact binary entity bytes. The
transport-independent prepared-request API remains available without the
client.

Delivery state belongs in the
[Public Rust SDK task](../../tasks/rust/public-rust-sdk.md).

## Availability and construction

`client-reqwest` is a default feature backed by target-specific native
dependencies. The client module is compiled only outside the WebAssembly target
family. WebAssembly and no-default-features consumers use `PreparedRequest` or
`PreparedBinaryRequest` with a caller-owned executor.

The crate manifest selects `reqwest` with default features disabled and only
the native TLS and streaming capabilities required by the SDK. The manifest
and lockfile own the exact dependency version.

Every official client is constructed by the single private factory behind
`Client::builder`. Endpoint code cannot create another `reqwest::Client` or
bypass the factory.

## Fixed transport contract

The official client always:

- disables reqwest retries with `reqwest::retry::never()`;
- disables redirects;
- ignores ambient and system proxy configuration;
- selects native TLS with a TLS 1.2 minimum;
- selects the non-Hickory resolver even when Cargo features are unified;
- disables gzip, Brotli, Zstandard, and deflate response decoding;
- disables automatic `Referer` generation;
- accepts only the production HTTPS origin;
- stores no cookies or implicit authentication state; and
- applies connect, per-read, and total request/body deadlines.

These are guarantees, not builder defaults. Public configuration cannot enable
retries, redirects, proxies, alternate DNS or TLS backends, automatic content
decoding, arbitrary origins, or an unbounded mode.

Cargo feature unification is part of this boundary. A separate compatibility
package deliberately enables both TLS backends, Hickory DNS, HTTP/2, streaming,
and every supported compression feature on the same `reqwest` dependency. Its
tests prove that the official factory still uses system resolution, emits the
required native TLS client handshake, and preserves the fixed runtime policy.

## Supported configuration

`ClientBuilder` exposes only:

- a nonzero connection timeout;
- a nonzero per-read timeout;
- a nonzero total request and body deadline;
- a nonzero inclusive JSON, XML, and alternate-envelope byte limit; and
- a visible-ASCII application suffix for the SDK user agent.

Invalid or unrepresentable values fail during `build`, before network access.
Representation selection belongs to the generated `prepare_json`,
`prepare_xml`, or binary preparation method. Streaming storage and workflow
budgets remain caller-owned.

Applications needing an explicit proxy, custom connector or resolver, different
TLS or decoding policy, transport instrumentation, or a nonproduction origin
use prepared requests with their own executor. That executor does not inherit
the official client's guarantees.

## One-interaction contract

One high-level execution performs at most one OpenDART HTTP request. The client
does not repeat a request after:

- connection, TLS, or protocol failure;
- a retryable HTTP/2 event;
- a redirect response;
- timeout or incomplete body;
- HTTP or OpenDART source status; or
- malformed or unrecognized response syntax.

DNS lookups and failed connection establishment are transport activity, not
additional source requests. Compatibility tests count observed HTTP request
streams. A caller may make another explicit invocation, but the SDK neither
automates nor labels that policy decision.

## Entity bytes and response metadata

The client never calls automatic text, JSON, or content-decoding helpers before
bounded SDK inspection. Concatenating successful binary `BodyChunk` values
yields the entity bytes delivered by the HTTP stack in order, without
decompression or character conversion. HTTP framing is outside this contract.

`ResponseMetadata` retains the HTTP status and version plus a conservative
allowlist of delivery and representation headers. Header values fail closed if
they contain the credential or an authenticated query, or if multi-stage
percent decoding is malformed, ambiguous, or unsafe. Redirect targets, cookies,
and extension headers never cross the public metadata boundary. Metadata
remains available when structured decoding fails or when a returned binary
stream later fails.

Structured JSON and XML responses are buffered only to the configured envelope
limit. Binary responses remain fallible streams: timeout, incomplete delivery,
or a body-read failure produces a sanitized terminal `BodyStreamError`, not
clean end-of-stream.

Strict collectors that need more HTTP evidence use the prepared-request seam
and own raw header capture, exact-byte persistence, decoding, and artifact
policy.

## ZIP and XML discrimination

Some operations normally return ZIP but may return a source status as XML.
`Client::execute_binary` distinguishes:

- `BinaryReply::Archive` after a supported positive ZIP signature;
- `BinaryReply::Status` after complete bounded XML-envelope validation; and
- `BinaryReply::Unrecognized` for every other body.

Only a normal local-file header (`PK\x03\x04`) or the empty-archive
end-of-central-directory signature (`PK\x05\x06`) is positive archive evidence.
Split or spanned markers, ZIP64-only prefixes, self-extracting preambles,
truncated signatures, and other binary prefixes remain unrecognized. This is
representation classification, not archive validation or extraction.

Classification retains every consumed byte. Archive and unrecognized results
return a replaying stream containing the inspected prefix followed by the
untouched remainder. A plausible XML candidate is buffered only to the envelope
limit; truncated, malformed, ambiguous, or oversized candidates become
unrecognized replay streams. `Content-Type` is evidence and never the sole
discriminator.

## Credential and diagnostic boundary

OpenDART authorization is a query credential, so the authorized URI is secret.
The SDK enforces these boundaries:

- `ApiKey` owns secret memory, redacts `Debug`, and implements neither
  `Display` nor serialization.
- Prepared requests contain no credential.
- `AuthorizedRequest` is non-cloneable and exposes its URI only by being
  consumed inside an explicit adapter callback.
- The public client uses that same authorization path and adds exactly one
  `crtfc_key`.
- Public errors retain only sanitized operation identity, failure class, and
  safe response metadata. They never expose a raw `reqwest::Error` or URL.
- No SDK error claims retryability or request-send certainty that the transport
  cannot establish.

The test suite uses sentinel credentials and rejects literal, form-encoded, or
percent-encoded disclosure through errors, metadata, debug output, and
repository-owned logs.

## Verification evidence

Loopback tests exercise the fixed contract rather than inspecting configuration
alone. They cover retryable protocol events, redirects, ambient proxy variables,
feature-unified compression, TLS and DNS selection, deadlines, incomplete
bodies, metadata sanitization, authorization, and ZIP/XML replay.

The canonical command is the Rust verification mode documented in the
[workspace README](../../sdk/rust/README.md). It is credential-free and makes
no live OpenDART request.
