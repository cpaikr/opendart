# Public Rust SDK

## Outcome

Publish a first-party `opendart` crate derived from the canonical OpenAPI 3.2
contract. The crate must make complete OpenDART request construction reusable
without taking ownership of application-specific execution, persistence,
retry, quota, collection, or domain policy.

The specification remains the primary product and source of truth. The Rust
SDK is a reviewed, reproducible derivative published from this repository to
crates.io. Future Python and TypeScript/Node SDKs may reuse the same normalized
generator model without depending on Rust source.

## Current state

- The repository publishes only the OpenAPI bundle and its checksum. It has no
  Cargo workspace, public runtime package, SDK generator, or crates.io release
  flow.
- Go is the private repository-tooling language. `cmd/opendart-tool` and
  `internal/openapi` already provide the trusted OpenAPI loading, validation,
  and deterministic-artifact boundary.
- The canonical contract deliberately omits unsupported scalar assumptions,
  uses default responses where HTTP behavior is unknown, represents JSON and
  XML as distinct physical operations, and records source-specific identity
  and evidence under `x-opendart`.
- The current architecture, README, release policy, release guard, and Release
  Please configuration all assume that no runtime package exists. They must be
  revised deliberately rather than bypassed.

## Decisions

- Add a new ADR before implementation that expands the product boundary from
  "specification only" to "canonical specification plus first-party derived
  SDKs." ADR 0001 continues to govern private Go repository tooling.
- Keep OpenAPI 3.2 canonical. Do not weaken or replace it with a generator's
  preferred dialect.
- Extend the existing Go tool with a repository-owned, language-neutral SDK
  model and deterministic language emitters. Do not introduce a second OpenAPI
  parser or a consumer-side generator.
- Start with one public Rust crate. Its always-available core prepares and
  inspects OpenDART wire interactions without performing I/O. An optional,
  default-enabled `reqwest` client provides the ergonomic path for ordinary
  callers.
- Make the supported advanced seam an immutable prepared request, not a public
  async transport trait. A project with strict transport requirements can
  execute that value through its own adapter.
- Do not infer strong response scalar types that the source contract does not
  establish. Generated wire shapes preserve unknown values and fields; they
  are not application domain models.
- Publish the crate to crates.io with SemVer independent from the specification
  release version. Every crate release records the exact Git revision,
  specification release when one exists, and bundle checksum for the reviewed
  source snapshot it implements.
- Commit generated Rust source and verify its freshness offline against a
  deterministic SDK projection of the canonical contract. Specification
  changes outside that projection do not rewrite or release the crate. Consumer
  builds never run Go, parse OpenAPI, access the network, or depend on this
  repository layout.

## Scope

Included:

- Complete physical-operation inventory and stable logical-operation identity.
- Typed request inputs, requiredness, explicit constraints, and exact path and
  query serialization.
- Credential-safe authorization, request metadata, and representation routing.
- Conservative JSON/XML status-envelope and success-payload inspection,
  including XML error bodies returned by ZIP operations.
- An optional safe-default `reqwest` client with bounded, explicit transport
  configuration.
- Deterministic generation, offline fixtures, CI, documentation, crates.io
  packaging, provenance, and release policy.

Not included:

- Application retry scheduling, quotas, acquisition identity, deduplication,
  dataset closure, successful-empty policy, artifact storage, or domain models.
- Automatic promotion of guide prose into validation rules.
- Automatic updates from guide drift or live observations.
- A public Go package, a public generator API, or generation during a consumer
  build.
- Empty Python, TypeScript, or CLI packages created only to reserve structure.

## Target flow

```text
canonical OpenAPI 3.2 + x-opendart evidence
    -> private Go OpenAPI boundary
    -> repository-owned SDK model
    -> deterministic Rust emitter
    -> checked-in generated request and wire code
    -> public opendart crate
         |-> prepared-request API -> caller-owned executor
         `-> optional reqwest client -> ordinary SDK caller
```

Response interpretation remains layered:

```text
HTTP response metadata + entity bytes
    -> source envelope / representation evidence
    -> caller-owned collection and domain policy
```

The SDK may identify an OpenDART status such as `013`; it must not decide that
the status is a successful empty result for a particular collection profile.

## Workstream documents

- [Repository layout](../docs/rust-sdk/repository-layout.md) defines package
  placement,
  ownership, feature boundaries, and room for later SDKs and a public CLI.
- [Public contract](../docs/rust-sdk/public-contract.md) defines the low-level
  and ergonomic
  interfaces, invariants, errors, and supported escape path.
- [Generation](../docs/rust-sdk/generation.md) defines the source-to-SDK
  pipeline, conservative
  type policy, freshness gate, and complete operation coverage.
- [Transport and safety](../docs/rust-sdk/transport-and-safety.md) defines `reqwest`
  configuration, configurable limits, secret handling, exact-byte behavior,
  and one-interaction tests.
- [Verification and release](../docs/rust-sdk/verification-and-release.md) defines test layers,
  CI, release-guard changes, SemVer, crates.io publication, and consumer
  adoption.

Status and execution order live only in this file. Child documents hold the
target constraints and acceptance details for their workstreams.

## Ordered work

### 1. Record the product-boundary decision and prove dependencies

- Add a proposed ADR covering the public SDK boundary, private Go generator,
  independent package versioning, and transport-independent core.
- Update the ADR to accepted before publishing any supported crate surface.
- Verify the crates.io package name, Rust MSRV, Cargo resolver, license files,
  docs.rs metadata, and the selected `reqwest` minor line.
- Build focused throwaway compatibility tests for the selected XML/JSON
  libraries, secret wrapper, and `reqwest` retry/redirect/raw-body behavior.
- Confirm that the existing Go OpenAPI boundary can expose all required
  physical operations and `x-opendart` metadata without leaking third-party
  model types.

### 2. Establish the Rust workspace and public-contract skeleton

- Add the isolated workspace and one `opendart` library crate described in
  [repository layout](../docs/rust-sdk/repository-layout.md).
- Extend `.gitignore` only for the workspace's build output; keep the
  verification lockfile and checked-in generated source tracked.
- Commit the toolchain and MSRV policy, lockfile for repository verification,
  formatting, lint, test, documentation, and package-dry-run commands.
- Implement handwritten operation identity, prepared request, authorization,
  response metadata, source status, opaque source value, and error types using
  representative manually declared operations.
- Prove that the low-level core has no Tokio or `reqwest` dependency when the
  client feature is disabled.
- Review the public API before generating the complete surface. Remove any
  transport, application-policy, or third-party types that do not belong.

### 3. Add the normalized model and deterministic Rust emitter

- Extend `internal/openapi` with the narrow repository-owned data needed by SDK
  generation.
- Add the SDK model, validation, name-collision checks, and Rust emitter.
- Generate checked-in operation metadata, request input types, serializers,
  representation routing, and conservative wire shapes.
- Add a direct `opendart-tool` generation command and make repository verify
  fail when generated Rust is stale, incomplete, or contains unsupported
  constructs.
- Demonstrate complete physical-operation coverage without hard-coded endpoint
  totals.

### 4. Complete the wire inspector and optional HTTP client

- Implement bounded JSON/XML envelope inspection and a byte-replaying
  discriminator for ZIP success versus alternate XML source errors.
- Add the optional `reqwest` client on top of the prepared-request core.
- Enforce explicit no-retry, no-redirect, no-ambient-proxy, fixed Rustls and DNS
  resolver selection, and no-automatic-decompression behavior in one private
  client factory.
- Add configurable connect, read, and total timeouts; buffered-body limits; a
  user agent; and streaming download behavior without exposing knobs that
  violate the one-interaction contract.
- Add the transport and credential-safety integration suite before calling the
  client safe by default.

### 5. Close full coverage and package the crate

- Generate every physical operation and verify logical-operation pairing,
  request serialization, response routing, unknown-field retention, and source
  provenance.
- Complete crate-level guides for ordinary and advanced callers. Mark generated
  modules clearly and keep implementation-only details private.
- Run package-content, MSRV, stable, all-features, documentation, and offline
  reproducibility gates.
- Revise the architecture, README, release policy, release guard, and Release
  Please setup to represent the implemented SDK as a current product. Give the
  crate a Rust-aware release component that owns its Cargo version and alone
  authorizes crates.io publication.

### 6. Publish and adopt

- Publish a prerelease crate, verify installation from crates.io in a clean
  fixture, and inspect docs.rs output and package provenance.
- Adopt only the prepared-request and wire-inspection layer in strict
  collectors. Keep their executor, persistence, and application policy local.
- Publish the first non-prerelease SDK version only after the generated
  inventory and safe-default client acceptance suites pass and the public API
  has been reviewed against a real consumer.

## Cross-plan constraints

- SDK generation and verification are credential-free and offline. They do not
  depend on the planned live-conformance runner.
- Live observations may motivate reviewed specification or fixture changes but
  never rewrite generated SDK source automatically.
- The public SDK and the focused Go probe may share contract facts, not runtime
  packages, credentials, or HTTP implementations.
- Adding Rust CI must deliberately update the exact workflow and release-guard
  allowlists. Do not weaken existing Go, workflow-permission, or immutable
  release checks to make room for Cargo.
- A specification-only release and a Rust-crate release are independently
  classified and versioned even when one commit changes both products.

## Overall acceptance criteria

- Every canonical physical operation maps to exactly one generated Rust
  operation, with stable physical and logical identities and no silent
  unsupported construct.
- A caller can construct and authorize any supported request without enabling
  the HTTP client feature.
- Request preparation performs no I/O and emits deterministic method, trusted
  relative path, query serialization, authentication requirement, and expected
  representation metadata.
- No SDK API silently retries, follows redirects, reads ambient proxy settings,
  automatically decodes content encodings, or converts response bytes to text
  before an explicit bounded interpretation step.
- The SDK never exposes an API key or authenticated URL through `Debug`,
  `Display`, serialization, errors, logs, snapshots, or generated examples.
- Generated wire types preserve uncertainty and unknown fields rather than
  asserting unsupported domain meaning.
- The default-feature crate supports ordinary calls; the no-default-feature
  crate remains transport- and runtime-independent.
- Repository verification proves Go and Rust tests, generated freshness,
  package contents, and release policy without contacting OpenDART or requiring
  a credential.
- crates.io artifacts can be reproduced from a reviewed repository revision
  and identify the exact canonical specification artifact they implement.

## Next action

Start ordered work 1 by drafting ADR 0002 and running the narrow dependency and
public-seam compatibility gate. Do not add generated endpoint code or release
automation until that gate fixes the crate boundary, MSRV, selected `reqwest`
line, and conservative wire representation.

## Progress log

- 2026-07-18: Recorded the implementation plan and its repository, public API,
  generation, transport-safety, verification, and release workstreams. No SDK
  implementation or product-boundary documentation has been promoted to
  current state.
