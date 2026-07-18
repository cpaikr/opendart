# Rust SDK Transport and Safety

## Purpose

Define the optional `reqwest` client's fixed safety behavior, supported
configuration, exact-byte boundary, credential handling, and integration tests.
The transport-independent request core remains usable without this client.

## Dependency baseline

At plan creation, `reqwest` 0.13.4 is the preferred compatibility-gate
candidate. It exposes explicit retry policy, defaults to retrying selected
low-level protocol NACKs, defaults to following redirects, and declares Rust
1.85.0. `reqwest` 0.12.28 has the same relevant retry and redirect controls and
a lower declared MSRV, but should be selected only if the repository adopts a
lower MSRV deliberately.

Before committing the dependency:

- confirm the selected line against its versioned official documentation and
  source;
- record the crate MSRV and test it in CI;
- inspect its default and optional Cargo features;
- run the protocol-NACK, redirect, proxy, and compression compatibility
  fixtures; and
- pin the repository lockfile while keeping an appropriate published SemVer
  requirement.

No Cargo `retry` feature is required or available. Retry behavior is part of
the client and must be disabled through `ClientBuilder`.

References:

- [`reqwest` 0.13.4 retry module](https://docs.rs/reqwest/0.13.4/reqwest/retry/)
- [`reqwest` 0.13.4 `ClientBuilder`](https://docs.rs/reqwest/0.13.4/reqwest/struct.ClientBuilder.html)
- [`reqwest` 0.13.4 feature list](https://docs.rs/crate/reqwest/0.13.4/features)
- [`reqwest` 0.12.28 `ClientBuilder`](https://docs.rs/reqwest/0.12.28/reqwest/struct.ClientBuilder.html)

## Cargo features

Begin the compatibility gate with a minimal explicit dependency equivalent to:

```toml
[dependencies.reqwest]
version = "0.13.4"
optional = true
default-features = false
features = ["rustls", "stream"]
```

Retain `stream` only if the public binary-body interface requires
`bytes_stream`; `Response::chunk` may allow a smaller internal implementation.
Do not enable `json` or `query` merely for convenience: request serialization
and bounded source decoding belong to repository-owned code. Do not enable
compression, cookies, system proxy, blocking, multipart, or additional HTTP
versions without a demonstrated operation requirement.

`default-features = false` is necessary but insufficient as a runtime
guarantee because Cargo features unify across a dependency graph. The client
factory must explicitly disable proxy and decoding behavior even when another
crate enables those `reqwest` features, and it must pin the intended TLS backend
and DNS resolver. A dedicated integration-test package or dev-dependency must
deliberately enable HTTP/2, `native-tls`, `hickory-dns`, and every compression
feature on the same `reqwest` version to exercise that unification.

## Mandatory client factory

Construct every official convenience client through one private function. No
endpoint may call `reqwest::Client::new()` or maintain a second builder path.

The target configuration is equivalent to:

```rust
let client = reqwest::Client::builder()
    .retry(reqwest::retry::never())
    .redirect(reqwest::redirect::Policy::none())
    .no_proxy()
    .tls_backend_rustls()
    .no_hickory_dns()
    .no_gzip()
    .no_brotli()
    .no_zstd()
    .no_deflate()
    .referer(false)
    .https_only(true)
    .connect_timeout(config.connect_timeout)
    .read_timeout(config.read_timeout)
    .timeout(config.total_timeout)
    .user_agent(config.user_agent)
    .build()?;
```

Exact method availability is verified against the selected dependency before
implementation. The explicit TLS, DNS, proxy, and no-compression selections are
intentional guards against Cargo feature unification changing runtime behavior.

## Fixed invariants versus configuration

### Fixed, not user-enableable

- Retry policy is always `reqwest::retry::never()`.
- Redirect policy is always `Policy::none()`.
- Ambient system and environment proxy discovery is disabled.
- The TLS backend is always Rustls.
- Cargo feature unification cannot switch the default resolver to Hickory DNS.
- Gzip, Brotli, Zstandard, and deflate auto-decoding are disabled.
- Automatic `Referer` generation is disabled.
- The production client accepts only HTTPS OpenDART targets.
- Cookie storage and implicit authentication state are absent.
- Response bodies are never passed through `text()` or `json()` before an
  explicit bounded interpretation step.
- The SDK performs no application retry, backoff, quota wait, or status-based
  repeat request.

These are contract guarantees, not convenience defaults. A builder must not
offer methods that turn them on.

### Configurable while preserving invariants

- Connect timeout.
- Per-read timeout.
- Total request/body deadline.
- Maximum buffered JSON/XML or error-envelope bytes.
- Maximum diagnostic preview, always sanitized.
- User-agent application suffix after validation.
- Representation choice where the operation supports it.
- Streaming consumer budget or sink behavior outside the HTTP client.

Provide safe nonzero defaults for ordinary callers. Validate contradictory or
zero values before network access. Do not hide an unbounded mode behind
`Option::None` in the safe-default client.

An explicit proxy, custom connector, custom DNS, unrestricted origin, caller-
built `reqwest::Client`, or different decoding policy belongs to a caller-owned
executor using `PreparedRequest`. That path does not inherit the convenience
client's transport guarantees. Add a typed, explicitly configured proxy option
to the official client only after a real consumer proves it is needed and the
credential-routing and one-interaction tests define its contract; never add an
"use environment proxies" switch.

## One-interaction contract

One invocation of a high-level operation performs at most one source HTTP
request. Redirect targets and protocol retries would be additional source
requests and are therefore forbidden. DNS resolution and TCP/TLS connection
attempts that fail before request headers are sent are transport activity, not
additional OpenDART requests; tests count observed request headers or streams,
not socket connections.

The SDK does not retry after:

- connection or TLS failure;
- HTTP/2 or HTTP/3 protocol NACK;
- redirect response;
- timeout;
- incomplete body;
- HTTP status;
- OpenDART source status; or
- malformed representation.

A caller may schedule another operation invocation as its own new attempt. The
SDK neither automates nor labels that decision.

## Exact entity bytes

The low-level response path preserves:

- HTTP status and version;
- response headers, including `Content-Encoding` and `Content-Length` when
  supplied;
- undecoded entity-body chunks in observed order; and
- an explicit incomplete-body error if streaming fails.

The ergonomic path exposes a sanitized metadata view with the same status and
version plus all response headers whose values do not contain the credential or
an authenticated URL. Unsafe values are removed rather than partially rendered.
Metadata remains available when buffered decoding fails or a returned binary
stream later terminates with an error.

HTTP framing bytes are outside the entity-body contract. Concatenating all
successful body chunks yields the entity bytes delivered by the HTTP stack
without content decoding or character conversion.

The ergonomic JSON/XML path may buffer up to its configured limit and then
inspect a derived byte view. Binary responses remain streaming. Strict callers
can persist exact bytes before choosing any decompression or parsing step.

For ZIP-success/XML-error operations, classification reads a bounded prefix
and, only for a plausible XML envelope, buffers up to the configured XML limit.
Every byte read during classification is retained. A ZIP or unrecognized
result exposes a replay stream consisting of the retained prefix followed by
the untouched remainder. `Content-Type` may guide diagnostics but cannot by
itself select success. Ambiguous, truncated, malformed, and oversized XML
candidates return an unrecognized replay stream rather than losing bytes or
being labeled archive success.

## Credentials and diagnostics

OpenDART uses a query credential, so raw URLs are secret-bearing after
authorization. Enforce all of the following:

- `ApiKey` redacts `Debug`, has no `Display` or serialization, and clears owned
  memory where the selected secret type can provide that guarantee.
- `PreparedRequest` contains no key.
- `AuthorizedRequest` does not expose a safe printable URL and is consumed by
  the adapter.
- Public errors strip any URL retained by `reqwest::Error` and add only the
  sanitized operation identity and failure class.
- Logging occurs before authorization or uses an allowlisted structure. Never
  log a request builder, raw URI, response body, or third-party error debug
  representation without sanitization.
- Tests use a sentinel key and scan errors, snapshots, logs, and debug output
  for both the literal key and its percent-encoded form.

## Integration tests

### Redirect

Run two local listeners. The first records one request and returns a redirect
to the second. Assert that one client call records exactly one request at the
first listener, zero at the target, and returns the un-followed response or its
documented sanitized classification.

Use local TLS with a test-only trust root, or a private test-origin seam that
cannot be constructed through the packaged public API. Do not add a public
arbitrary-base-URL option merely to simplify tests, because an authenticated
request to an untrusted origin would disclose the query credential.

### Protocol retry

Use a deterministic local HTTP/2 fixture and a test package that enables
`reqwest`'s HTTP/2 feature on the unified dependency. Record request streams and
return a retryable protocol NACK such as `REFUSED_STREAM`. Assert exactly one
observed request stream and one terminal client result. This test must fail
against an otherwise equivalent client using `reqwest` defaults, proving that
the fixture exercises retry behavior rather than only inspecting configuration.

If the selected `reqwest` version changes its default classifier, update the
fixture to a documented safe-to-retry protocol event before upgrading.

### Ambient proxy

Run the test in an isolated subprocess with HTTP and HTTPS proxy environment
variables pointing to a counting listener. Call a local trusted test target and
assert zero proxy observations and exactly one target request. Do not mutate
process-global proxy variables in parallel test threads.

### Content decoding

Use a test package or dev-dependency that enables gzip, Brotli, Zstandard, and
deflate on the unified `reqwest` dependency; enabling only the `opendart`
crate's features is not sufficient. Return known compressed payloads for every
codec. Assert that the observed bytes remain compressed and that encoding
headers are retained. This proves the explicit `no_*` calls survive Cargo
feature unification.

### TLS and DNS feature unification

Use a test package or dev-dependency that enables `native-tls` and
`hickory-dns` on the unified `reqwest` dependency. Assert through observable
local TLS and name-resolution fixtures that the official client still uses
Rustls and the non-Hickory resolver. Feature inspection alone is insufficient.

### Timeout and incomplete body

Exercise connect, stalled-read, total-deadline, and mid-body termination paths
with generous deterministic bounds. Assert one request at most, bounded
completion, preserved response metadata when available, no complete-body
claim, and sanitized errors. For a returned binary stream, assert that read and
total timeouts remain active while polling, a partial body ends in a fallible
stream item rather than clean EOF, and caller-owned storage budgets are not
silently enforced by the HTTP client.

### Credential redaction

Trigger construction, DNS/connect, TLS, redirect, timeout, status-envelope,
body-limit, and malformed-body errors with a sentinel credential. Scan all
public formatting and captured logs for the literal and encoded credential and
authenticated query key.

### Request construction

Assert that authorization adds exactly one `crtfc_key`, preserves the prepared
query encoding, rejects duplicate authorization, and does not change operation
identity or representation metadata.

### ZIP/XML discrimination

Return a successful ZIP body in irregular chunk boundaries and assert that the
archive stream reproduces it byte for byte. Separately cover a recognized XML
source error, the normal local-file and empty-archive ZIP signatures,
split/spanned and ZIP64-only prefixes, a self-extracting preamble, misleading
ZIP and XML content types, a truncated classification prefix, malformed XML,
and an XML candidate beyond the inspection bound. Assert that only a supported
positive ZIP signature becomes archive, only the recognized bounded envelope
becomes source status, and every other case returns a replay stream containing
all bytes consumed by the classifier.

## Acceptance criteria

- The SDK-owned client has exactly one construction path with explicit retry,
  redirect, proxy, TLS backend, DNS resolver, decoding, timeout, and HTTPS
  policy.
- Redirect and retryable protocol failures produce no second observed request.
- Ambient proxy variables cannot change the official client's route.
- All supported content encodings remain undecoded under all-features tests.
- Exact body chunks and headers are available without text conversion.
- The ergonomic client preserves sanitized response metadata for ordinary
  caller policy, including after buffered or streaming failures.
- ZIP/XML discrimination preserves every consumed byte for archive and
  unrecognized outcomes, requires positive supported ZIP evidence for archive,
  and never treats an alternate XML error as another callable operation.
- Credentials and authenticated URLs are absent from every public diagnostic
  and captured log path.
- Callers needing different transport behavior can use the prepared-request
  core without patching or forking generated request code.
