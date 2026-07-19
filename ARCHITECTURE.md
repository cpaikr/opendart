# Repository architecture

## Product boundary

This repository maintains two public products derived from one source-backed
OpenDART contract:

- the portable OpenAPI bundle; and
- the `opendart` Rust protocol SDK.

The official OpenDART development guide is authoritative for documented
behavior. Authenticated observations remain separate evidence. Go packages and
`cmd/opendart-tool` are private repository tooling, not supported public APIs.

## System map

```text
OpenDART guide
    -> staged acquisition and normalization
    -> canonical multi-file OpenAPI 3.2
    -> deterministic bundle ----------------------> GitHub specification release
    -> repository-owned SDK model
    -> deterministic checked-in Rust -------------> opendart crate package

OpenDART API + local API key
    -> fixed, bounded probes
    -> sanitized empirical evidence
```

Pull-request verification never refreshes from OpenDART and never receives an
API key. It validates the committed specification, regenerated artifacts, Rust
workspace, package contents, and release policy.

## Runtime flows

### Specification refresh

`go run ./cmd/opendart-tool sync` acquires the trusted guide surface through
`internal/guide`, renders a complete staging tree, and validates it before
replacing the owned canonical output. A successful refresh invalidates the old
bundle; `opendart-tool bundle` regenerates it explicitly.

`internal/openapi` confines third-party OpenAPI types and local reference
loading. It owns strict linting, deterministic bundling, semantic comparison,
response validation, and the repository-owned SDK input projection.

### Rust generation and use

`opendart-tool generate-sdk` passes the canonical contract through
`internal/sdkgen/model` and the Rust renderer. Generated source is reviewed and
committed beneath `sdk/rust/crates/opendart/src/generated`; consumer builds do
not run Go or parse OpenAPI.

The crate exposes generated operation types plus handwritten request,
authorization, wire-inspection, and provenance contracts. Its core performs no
I/O. The optional default `client-reqwest` feature adds one-attempt bounded HTTP
execution with redirects, retries, ambient proxies, and automatic response
decoding disabled. Applications retain persistence, quota, retry, collection,
and domain policy.

### Verify and package

`.github/workflows/verify.yml` runs read-only Go and Cargo gates. Dependency
fetching is explicit; compilation, tests, documentation, generation checks,
and package inspection then run offline. The verifier requires exact generated
operation coverage, a tracked crate inventory, matching Cargo versions, source
provenance, and the approved workflow and release configuration.

Cargo records the exact packaged source revision in `.cargo_vcs_info.json`.
The crate's `source_provenance()` additionally records the canonical bundle,
SDK projection, generator schema, and applicable specification release.

### Releases

Release Please owns independent components:

- root `vX.Y.Z` tags and `CHANGELOG.md` for the OpenAPI bundle; and
- `opendart-vX.Y.Z` tags and the crate changelog/version for the Rust SDK.

Rust changes are excluded from root release eligibility. The Rust component is
configured to prepare a draft component release and keep the crate manifest and
workspace lock aligned. This repository does not yet authorize `cargo publish`;
registry publication and post-publication verification belong to work 6.

### Focused live probes

`probe-multi-company` and `probe-auditor-evidence` use a Varlock-injected local
`OPENDART_API_KEY`, fixed request matrices, bounded bodies, sequential attempts,
and sanitized output. They do not change the specification or SDK.

`internal/liveprobe` confines their one-attempt HTTP policy. Its dated TLS
compatibility exception is empirical probe behavior and must not be reused by
the released SDK transport.

## Code map

- `openapi/openapi.yaml` and its referenced fragments are the canonical source.
  `openapi/generated/openapi.bundle.yaml` is the portable bundle.
- `cmd/opendart-tool` is the private command surface.
- `internal/guide` owns guide acquisition and guarded generation.
- `internal/openapi` owns the confined OpenAPI boundary and SDK projection.
- `internal/sdkgen/model` and `internal/sdkgen/rust` own deterministic SDK
  normalization and rendering.
- `sdk/rust` is the isolated Cargo workspace. The public crate is under
  `sdk/rust/crates/opendart`; generated files have one owned subtree.
- `internal/verification` coordinates repository verification;
  `internal/releaseguard` enforces workflow, package, provenance, and release
  policy.
- `internal/multicompanyprobe`, `internal/auditorprobe`, and
  `internal/liveprobe` own credentialed empirical work.
- `.github/workflows/verify.yml` is the credential-free repository gate.
  Release Please configuration and `.github/workflows/release-please.yml` own
  component release preparation and specification asset publication.

## Invariants

- OpenAPI 3.2 remains canonical; generated SDK files never become an alternate
  endpoint inventory.
- Generated OpenAPI and Rust files change through their generators, not by
  hand, and verification requires byte-for-byte freshness.
- Guide facts, empirical observations, and application policy remain separate.
- Offline verification makes no OpenDART request and requires no credential.
- The SDK core can prepare and inspect every supported operation without an
  HTTP runtime. No SDK path silently retries or exposes a credential-bearing
  URL through safe diagnostics.
- Specification and crate versions, tags, changelogs, and release eligibility
  are independent.
- No current workflow has crates.io publication authority.

## Decisions and evolution

[ADR 0001](docs/decisions/0001-go-repository-tooling.md) keeps Go as private
repository tooling. [ADR 0002](docs/decisions/0002-public-rust-sdk.md)
accepts the first-party Rust SDK boundary. Current SDK implementation and the
remaining publication/adoption work are tracked in the
[public Rust SDK task](tasks/rust/public-rust-sdk.md).
