# Rust SDK Verification and Release

Planning source: [Public Rust SDK](../../tasks/rust/public-rust-sdk.md).

## Purpose

Add Rust verification and crates.io publication without weakening the existing
offline OpenAPI, Go tooling, workflow-permission, or immutable-release gates.
Keep specification and SDK versions independently meaningful and traceable.

## Verification layers

### Go repository gate

Extend the existing verifier to cover:

- complete SDK-model construction;
- generator diagnostics and unsupported constructs;
- exact canonical-to-generated operation coverage;
- deterministic Rust generation and byte freshness;
- generated ownership markers and confined output paths;
- required Cargo workflow steps and release configuration; and
- release-managed source provenance embedded in the handwritten crate module.

The Go verifier remains offline and credential-free. It may inspect committed
Cargo metadata and generated source but does not replace Cargo compilation or
tests.

### Rust static and unit gate

Run direct Cargo commands against the isolated workspace:

```sh
cargo fmt --manifest-path sdk/rust/Cargo.toml --all -- --check
cargo clippy --locked --manifest-path sdk/rust/Cargo.toml --workspace --all-targets --all-features -- -D warnings
cargo test --locked --manifest-path sdk/rust/Cargo.toml --workspace --all-features
cargo test --locked --manifest-path sdk/rust/Cargo.toml -p opendart --no-default-features
RUSTDOCFLAGS="-D warnings" cargo doc --locked --manifest-path sdk/rust/Cargo.toml --workspace --all-features --no-deps
cargo package --locked --manifest-path sdk/rust/crates/opendart/Cargo.toml
```

Final flags may change during the compatibility gate, but validation remains
direct rather than hidden behind a task runner. Add a separate MSRV job using
the crate's declared `rust-version`; run the main gate on the pinned stable
toolchain.

Package verification must inspect the produced archive, not only command exit
status. Require the license, crate README, generated code, source provenance,
and intended documentation while excluding repository fixtures, credentials,
local artifacts, and private generator source.

### HTTP integration gate

Run the local, credential-free transport suite from
[transport and safety](transport-and-safety.md), including all-features
crate coverage plus a dedicated dependency-unification fixture that enables
`reqwest` HTTP/2, `native-tls`, `hickory-dns`, and compression features for the
protocol-NACK, fixed-backend, fixed-resolver, and raw-body cases. Tests use only
loopback fixtures and no OpenDART credential.

The Hickory graph has a higher MSRV than the published crate, so its persistent
build proof is a nested stable-only workspace:

```sh
RUSTFLAGS="--cfg opendart_compat" cargo test --locked \
  --manifest-path sdk/rust/compat/reqwest-feature-unification/Cargo.toml
```

The [live-conformance task](../../tasks/main/live-conformance.md) remains
separate, protected observation work. A crate release does not depend on an
unreviewed live call or expose its
credential to package publication.

### Consumer fixture

Install the packaged crate tarball in clean temporary projects:

- ordinary default-feature async client compilation;
- `default-features = false` prepared-request compilation;
- minimal supported Rust compilation;
- downstream denial of generated private internals; and
- documentation examples as compile tests.

After a crates.io prerelease exists, repeat against the registry artifact rather
than a path dependency.

## CI integration

The current Verify workflow and release guard allow an exact Go-only step set.
Modify them intentionally:

- add pinned Rust setup and cache behavior with read-only permissions;
- keep Go vet, race tests, and repository verification unchanged;
- add the approved Cargo formatting, lint, test, documentation, and package
  steps;
- update release-guard tests to require the complete approved sequence and
  reject unapproved script, network, credential, or package-publish steps in
  pull-request verification;
- keep generated freshness before package publication; and
- keep pull-request verification free of API keys and crates.io credentials.

Do not relax exact step or action-pin checks merely because the workflow gains
another toolchain.

## Independent products and versions

The repository will contain two released products:

1. The canonical specification bundle, versioned by the existing repository
   specification release policy.
2. The `opendart` crate, versioned by Rust public API SemVer.

They may be released from the same commit but do not share a version number by
rule. The crate records:

- its own crate version;
- the source Git revision;
- the source specification release tag when one exists;
- the canonical bundle SHA-256; and
- the SDK generator schema/version.

Generated-source freshness uses the SDK-projection checksum, not the complete
bundle checksum. Full bundle provenance is selected and updated only in a crate
release PR. A specification change that leaves the projection unchanged must
leave the Rust component byte-for-byte unchanged.

If a specification change does not change the supported SDK contract, no crate
release is required; the last released crate continues to identify the older
snapshot it implements. If generated public behavior changes, classify and
release the crate independently.

## Rust SemVer classification

While the crate is below 1.0, document the project's pre-1.0 interpretation
explicitly. At and after 1.0, use standard SemVer.

Typical classifications:

| Change | Rust crate impact |
| --- | --- |
| Documentation or private generator refactor with identical output | None |
| Internal client bug fix with unchanged public contract | Patch |
| New operation, optional input, or additive convenience method | Minor |
| New recognized status constant on an open status type | Minor |
| Required input, serialization change, public rename/removal, or narrowed wire type | Major |
| Dependency update changing MSRV or guaranteed transport behavior | Compatibility review; major when consumers break |

Generated changes are not automatically nonbreaking. Diff public Rust API and
request behavior, not only the OpenAPI change label.

## Release Please and release guard

The existing Release Please configuration has one root specification package,
and `sdk/` changes would currently participate in that release stream. Before
the first crate release:

- exclude `sdk/` from the root specification component unless the same change
  materially changes the bundle;
- add a separate Rust SDK component rooted at the crate package path, using a
  Rust-aware release type rather than the root component's current `simple`
  type;
- make that component own the published version in
  `sdk/rust/crates/opendart/Cargo.toml`, its changelog, and its release tag;
- make every Rust release PR update `sdk/rust/Cargo.lock` for the proposed crate
  version, using a Rust-aware Release Please workspace mechanism or a controlled
  equivalent, and reject the proposed commit unless `cargo metadata --locked`
  plus the standard locked gates pass;
- use component-qualified tags for the crate so existing `vX.Y.Z`
  specification tags remain unambiguous;
- maintain separate manifest/version state and changelog ownership;
- teach the release guard the two allowed products and their independent
  eligibility rules; and
- test spec-only, SDK-only, combined, repository-only, interrupted-draft, and
  non-releasable changes; for each case assert the Cargo version, component
  tag, changelog, workspace-lock consistency, and crate-publication eligibility.

The spec-only case must include a canonical bundle checksum change with an
unchanged SDK projection and prove that no Rust component path or release state
changes.

The crates.io job consumes the Rust component's draft-release output and
immutable target revision specifically. A generic repository release, root
specification release, or change to the workspace alone cannot make a crate
publishable. Verify the exact component-qualified Release Please fields and Rust
version-file behavior against the pinned action version before committing the
configuration.

Do not force specification releases for SDK-only implementation changes, and do
not publish a crate merely because the specification bundle changed.

## crates.io publication

Before implementation, verify that the intended package name is available and
that repository, license, documentation, categories, keywords, and MSRV
metadata meet crates.io requirements.

The publication job must:

1. Depend on the same immutable source revision and complete verification gate.
2. Run `cargo package --locked` and inspect the resulting candidate archive.
3. Run `cargo publish --locked --dry-run` from the same immutable tree before
   any authenticated step.
4. Obtain narrowly scoped publication authority only in the release job,
   preferably through crates.io trusted publishing when configured and
   supported; otherwise use a dedicated least-privilege token.
5. Run `cargo publish --locked` once and treat the registry version as
   immutable.
6. Download the exact registry artifact and repeat the archive inspection;
   verify its registry checksum, provenance, normalized manifest, file
   inventory, and unpacked file-content digests against the reviewed candidate,
   then verify the docs.rs build.
7. Attach or link the crate provenance to the corresponding GitHub release
   without duplicating the crate tarball as an alternate package channel.

`cargo publish` does not accept a prebuilt `.crate`, so pre-publication review
and post-publication verification are deliberately separate gates. The latter
is authoritative for the bytes accepted by the registry.

### Recoverable publication state

Treat crates.io as publication truth because an accepted version cannot be
replaced. Release Please first creates or recovers a draft component release
anchored to the verified target revision. The publication job then:

1. Queries crates.io for the proposed crate name and version.
2. Publishes only when that version is absent.
3. If the version already exists, downloads it and treats it as a successful
   resumed publish only when its registry checksum, provenance, normalized
   manifest, and unpacked file contents match the reviewed candidate exactly.
   Any mismatch is a terminal conflict.
4. Verifies the accepted registry artifact and bounded docs.rs completion.
5. Publishes the GitHub component release and creates its immutable tag only
   after registry verification succeeds.

A crash after crates.io accepts the package therefore leaves a recoverable draft
rather than a final GitHub release with a missing crate. Reruns use the registry
comparison above and finish the same draft. Rust-only and combined releases
recover each product independently by component-qualified identity and never
reuse the root specification outputs as crate state.

Pull requests, branch builds, generator runs, and specification-only releases
receive no crates.io credentials.

## Full-coverage release gate

The first non-prerelease crate version requires:

- exact generated coverage of every canonical physical operation;
- reviewed logical-operation grouping and public naming;
- request-vector coverage for every physical operation;
- representative JSON, XML, ZIP, source-error, unknown-field, and unknown-
  scalar fixtures;
- the complete no-retry/no-redirect/no-proxy/no-decoding integration suite;
- credential redaction across all public error paths;
- successful no-default-feature and default-feature consumer fixtures;
- clean package and documentation builds on MSRV and stable Rust;
- accepted product-boundary and release ADRs/policy updates; and
- one real strict consumer review of the prepared-request seam.

Typed convenience response coverage may improve incrementally only where the
wire contract remains conservative and every operation is still callable and
inspectable. Do not claim a complete typed domain SDK when the source does not
provide those types.

## Strict-consumer adoption

A strict collector depends on the crate with default features disabled and
uses:

- generated operation inputs and identity;
- deterministic request preparation and authorization; and
- source-envelope/wire inspection after its own artifact boundary.

It retains:

- HTTP client construction and one-wire execution evidence;
- exact-byte streaming and artifact publication;
- attempt lifecycle and send-certainty policy;
- retry scheduling, quotas, and closure;
- successful-empty classification; and
- domain conversion and persistence.

Adoption begins against a released prerelease or local path only for integration
development. Production pins a crates.io version and records the crate/spec
provenance. A consumer-specific experimental endpoint may coexist locally, but
reusable source behavior is upstreamed before it becomes permanent.

## Future SDK and CLI releases

Python, TypeScript/Node, and a public CLI each receive independent package
versions, changelogs, publication credentials, and release gates. They may
share generator model fixtures and the canonical spec checksum. They do not
inherit Rust SemVer or publication merely because the Rust crate changes.

Do not make the internal Go tool a public CLI release. A future public CLI must
consume a supported SDK interface and define stable machine-readable output,
credential input, configuration precedence, and agent-friendly error behavior
before publication.

## Acceptance criteria

- Pull-request verification remains live-service-free, read-only, and
  credential-free; Cargo may access the package registry through the approved
  dependency-resolution steps.
- Existing Go/specification gates remain required and gain explicit Rust
  freshness and workflow-policy coverage.
- MSRV, stable, all-features, and no-default-feature builds pass.
- Specification and Rust products have independent versions, tags, changelogs,
  eligibility rules, and publication jobs.
- The published crate identifies its exact source revision and specification
  checksum and installs successfully from crates.io.
- No release path can publish stale generated code, an unverified package, or a
  credential-bearing artifact.
