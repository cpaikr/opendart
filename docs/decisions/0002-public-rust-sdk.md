# ADR 0002: Add a first-party Rust SDK

- Status: accepted
- Date: 2026-07-19

## Context

The canonical OpenAPI 3.2 contract is useful to generators and tooling, but
ordinary Rust callers otherwise have to choose their own generator, reproduce
OpenDART request serialization, and independently defend query credentials and
wire bytes. That fragments the protocol contract and makes the repository
unable to review the behavior users actually run.

The repository must retain one canonical endpoint inventory. It also must not
turn its private Go tooling, an HTTP runtime, or collector policy into public
application architecture.

## Decision

Expand the product boundary from the canonical specification alone to the
canonical specification plus first-party SDKs derived from it. The first SDK is
one crates.io package named `opendart`. Its package version and release stream
are independent from the specification version.

Keep the OpenAPI 3.2 document canonical. Extend the private Go tooling governed
by [ADR 0001](0001-go-repository-tooling.md) with a repository-owned normalized
SDK model and deterministic language emitters. Third-party OpenAPI model types
remain inside `internal/openapi`; generated Rust is committed and verified
offline. Consumer builds do not run Go or parse OpenAPI.

The Rust crate has one always-available, transport-independent core. It prepares
immutable requests, authorizes them at an explicit credential boundary, and
inspects bounded response bytes without performing I/O. A default-enabled
`client-reqwest` feature supplies the ordinary async client. The supported
advanced seam is `PreparedRequest`, not a public transport trait or a caller-
provided `reqwest::Client`.

The core exposes repository-owned public types. JSON and XML libraries, URL
machinery, secret storage, and HTTP-client types remain implementation details.
Generated wire values preserve unknown scalar kinds and fields rather than
becoming application domain models. Source status is evidence, including
unknown future strings; it does not define retry, successful-empty, collection,
quota, persistence, or domain policy.

Every crate release records its own version, the exact Git revision, the source
specification release when one exists, the canonical bundle checksum, and the
SDK generator schema version. Specification and Rust changes are classified,
versioned, tagged, and authorized independently.

## Compatibility gate

The initial dependency and repository policy is:

- Package name: `opendart`. The crates.io API reported the name unregistered on
  2026-07-19. Availability is rechecked before publication because this check
  does not reserve the name.
- License: MIT, using the repository `LICENSE` in the packaged crate.
- Rust: edition 2024, Cargo resolver 3, and MSRV 1.85.0. Repository verification
  pins stable Rust 1.97.1 initially and runs a separate MSRV job.
- HTTP: `reqwest` 0.13.4 with default features disabled and only `rustls` and
  `stream` enabled by the crate. `stream` is retained for the required fallible,
  byte-replaying binary response interface.
- Wire parsing: `serde_json` 1.0 and `quick-xml` 0.41 behind repository-owned
  bounded inspectors. XML document types are rejected and depth is limited by
  the SDK; the parser is never asked to resolve external entities.
- Secrets: `secrecy` 0.10 behind `ApiKey`, with explicit exposure only at the
  authorization boundary, redacted diagnostics, no serialization or display,
  and zeroization on drop.
- docs.rs: build all crate features and pass the crate's documentation cfg. The
  crate manifest owns repository, documentation, keywords, categories, README,
  and `rust-version` metadata.

A throwaway Cargo fixture enabled `reqwest` HTTP/2, native TLS, Hickory DNS, and
all compression features together. It proved that the selected builder exposes
explicit no-retry, no-redirect, no-proxy, Rustls, non-Hickory, and no-decoding
controls under feature unification. A local HTTP/2 `REFUSED_STREAM` fixture
observed the default original request plus its protocol retries, while
`retry(reqwest::retry::never())` observed one stream. Local redirect and gzip
fixtures observed no redirect follow and exact compressed entity bytes with the
encoding header retained.

The same fixture proved conservative JSON scalar and unknown-field retention,
explicit XML document-type/depth rejection, and secret debug redaction. It
passed on stable Rust. Its deliberate Hickory feature-unification build requires
Rust 1.88 through Hickory's current optional graph, so that safety fixture runs
on pinned stable; the published crate does not enable Hickory, and default and
no-default crate builds form the MSRV contract.

`internal/openapi.InspectSDKSurface` now proves that the existing private
OpenAPI model exposes every canonical physical operation, stable logical
identity, source provenance, request serialization fact, security scheme, and
response representation through repository-owned values. Its test compares
physical and logical coverage with the canonical catalog without embedding an
endpoint total. The final normalized model may deepen or replace this probe,
but it may not expose libopenapi types.

## Consequences

- The repository now owns a supported Rust protocol surface and its SemVer,
  packaging, documentation, and security guarantees.
- The specification remains the only endpoint inventory; generated source is a
  reproducible derivative rather than a second authority.
- The default client offers a deliberately narrow safe path. Callers requiring
  different proxy, DNS, connector, retry, decoding, or persistence behavior use
  the prepared-request core and own execution policy.
- Dependency upgrades that can change MSRV, wire bytes, retry, redirect, proxy,
  TLS, DNS, parsing, or credential behavior require a renewed compatibility
  review and focused fixtures.
- Later SDKs reuse the private normalized model and contract fixtures, not Rust
  syntax or a public Go API.

## Alternatives considered

- A generated client from a generic consumer-side generator would make OpenAPI
  3.2 support and safety behavior depend on each consumer and would duplicate
  the generator boundary.
- A public async transport trait would stabilize application execution policy
  and third-party types that are unnecessary for request reuse.
- Multiple Rust packages would add release and compatibility seams before an
  independent consumer proves they are useful.
- `reqwest` 0.12.28 permits a lower dependency MSRV, but the selected project
  MSRV already supports 0.13.4 and its explicit retry control. Maintaining an
  older HTTP line would not improve the chosen compatibility contract.

## Related work

- [Public Rust SDK task](../../tasks/rust/public-rust-sdk.md)
- [Rust SDK public contract](../rust-sdk/public-contract.md)
- [Rust SDK generation](../rust-sdk/generation.md)
- [Rust transport and safety](../rust-sdk/transport-and-safety.md)
- [Rust verification and release](../rust-sdk/verification-and-release.md)
