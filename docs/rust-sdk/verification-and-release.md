# Rust SDK verification and release

Planning source: [Public Rust SDK](../../tasks/rust/public-rust-sdk.md).

## Verification boundary

Rust verification extends the existing credential-free repository gate without
weakening specification, workflow-permission, or immutable-release checks.
The workflow may fetch locked registry dependencies; every build, test,
generation, documentation, and package command then runs offline.

### Repository gate

The Go verifier requires:

- complete canonical-to-generated physical-operation coverage;
- stable logical-operation pairing, response routing, and request vectors;
- deterministic generated source and ownership markers;
- approved Cargo package metadata, tracked package inventory, and source
  provenance matching the committed canonical bundle;
- aligned crate and workspace-lock versions; and
- the exact approved CI and Release Please policy.

It does not replace Cargo compilation or tests and never contacts OpenDART.

### Cargo gates

The pinned stable toolchain formats handwritten Rust and runs all-features and
no-default-features Clippy and tests, rustdoc with warnings denied, the reqwest
feature-unification fixture, and `cargo package`. The declared MSRV independently
runs all-features and no-default-features checks plus locked metadata.
Generated Rust uses generator-owned compact formatting; offline freshness is
its exact formatting gate, while every Cargo compile gate still includes it.

The package file list must exactly match `sdk/rust/package-files.txt`. This
requires the license, README, changelog, generated source, provenance, public
tests, normalized manifest, workspace lock, and Cargo-generated
`.cargo_vcs_info.json`, while excluding generator source, repository fixtures,
credentials, and local artifacts.

The default client integration suite uses loopback-only fixtures. It proves the
one-interaction contract under dependency feature unification: no automatic
retry, redirect, ambient proxy, alternate TLS backend, alternate DNS resolver,
or response-content decoding. No OpenDART credential is used.

The no-default-features graph must remain free of `reqwest`, Tokio, Hyper, TLS,
proxy, DNS, and streaming-runtime dependencies.

## Product versions and provenance

The repository has independent release components:

1. The canonical bundle uses root `vX.Y.Z` tags.
2. The `opendart` crate uses `opendart-vX.Y.Z` tags and Rust API SemVer.

They may release from the same commit but do not share a version by rule. A
crate package identifies:

- its Cargo version;
- its exact Git source revision through `.cargo_vcs_info.json`;
- the selected semantic specification source release, when applicable;
- the independently selected canonical bundle SHA-256;
- the generator schema; and
- the deterministic SDK projection SHA-256.

Generated freshness uses the SDK projection. A specification change outside
that projection does not rewrite or release the crate. The release guard proves
that the selected source tag exists and contains the canonical specification
inputs; generic verification does not freeze the current source tree to that
older tag. Full bundle provenance changes only when a crate release deliberately
selects a new generated artifact; that bundle need not be byte-identical to the
bundle originally generated at the source tag.

## Rust compatibility

Below `1.0.0`, compatible fixes are patches, compatible public additions are
minors, and breaking changes are majors. At and after `1.0.0`, use standard
SemVer. Examples:

| Change | Rust impact |
| --- | --- |
| Documentation or private refactor with identical output | None |
| Internal client fix with unchanged public contract | Patch |
| New operation, optional input, or open status constant | Minor |
| Required input, serialization change, public rename/removal, or narrowed wire type | Major |
| MSRV or guaranteed transport behavior change | Compatibility review |

Generated changes are not automatically compatible. Review public Rust API and
request behavior, not only the source OpenAPI label.

## Release Please boundary

The specification component is rooted at `openapi/generated`, so repository and
SDK commits cannot create a specification release. The separate Rust-aware
component is rooted at `sdk/rust/crates/opendart`, owns its `Cargo.toml` and `CHANGELOG.md`, uses
component-qualified tags, and updates the matching `opendart` entry in the
workspace lock through an explicit TOML extra file.

Before its first release, the Rust path is intentionally absent from
`.release-please-manifest.json`. Release Please proposes bootstrapping the
component at `0.1.0`, but the repository guard rejects that manifest transition
until work 6 implements the complete publication and recovery flow. Therefore a
Rust Release Please PR must not be merged before work 6; the same guard also
stops the release workflow if such a merge bypasses the required PR check.

The current Release Please workflow still finalizes only the specification
release assets. A Rust component can prepare a draft release proposal, but no
workflow has crates.io credentials or runs `cargo publish`. Path-qualified
specification outputs never authorize Rust publication.

## Work 6: crates.io publication

Publication is deliberately unimplemented. The later publication change must:

1. Consume the exact path-qualified Rust component release-created flag, tag,
   version, and immutable target revision from the pinned Release Please action.
   Before invoking Release Please, independently detect and resume an existing
   draft for that exact `opendart-vX.Y.Z` component tag and immutable target
   revision. A failed run may already have created the draft, so fresh action
   outputs alone are not a sufficient recovery mechanism.
2. Depend on the complete verification gate and reproduce the reviewed package
   inventory from that revision.
3. Run `cargo package --locked` and `cargo publish --locked --dry-run` before
   obtaining publication authority.
4. Use narrowly scoped crates.io trusted publishing when available, or a
   dedicated least-privilege token confined to the publication job.
5. Query crates.io before publishing and run `cargo publish --locked` only when
   the version is absent.
6. If the version already exists after an interrupted run, accept it only when
   its checksum, provenance, normalized manifest, and unpacked contents exactly
   match the reviewed candidate.
7. Download and inspect the accepted registry artifact, verify docs.rs, and
   finalize the matching GitHub component release only after those checks pass.

Because crates.io versions are immutable and `cargo publish` cannot accept a
prebuilt `.crate`, pre-publication review and post-publication verification are
separate required gates. Pull requests, branch builds, generator runs, and
specification-only releases must never receive registry authority.

## Adoption boundary

A strict collector uses the crate with default features disabled for generated
operation inputs, deterministic preparation and authorization, and bounded wire
inspection after its own artifact boundary. It retains HTTP execution evidence,
exact-byte storage, attempt lifecycle, retry scheduling, quotas, collection
closure, successful-empty classification, domain conversion, and persistence.

Production adoption waits for a verified crates.io version. A local path
dependency is suitable only for integration development before work 6.
