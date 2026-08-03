# Rust SDK verification and release

## Purpose

This document defines the durable verification, compatibility, provenance, and
publication-authority boundaries for the `opendart` crate. It does not record a
release run or its next action.

Current delivery and recovery state belongs in the
[Public Rust SDK plan](../../plans/rust/public-rust-sdk.md). Maintainer release
operations and repository-wide policy belong in
[RELEASING.md](../../RELEASING.md).

The verification details below describe the current generated implementation.
[ADR 0004](../decisions/0004-handwritten-rust-sdk-conformer.md) accepts the
replacement contract. Its independent inventory, retained evidence, and
representative handwritten JSON/XML/ZIP pilot already run privately; generated
freshness remains authoritative for the public product until the handwritten
implementation cutover.

## Verification interface

From the repository root:

```sh
# Focused edit loop
./scripts/verify fast rust -p opendart

# Focused offline Rust-native inventory, evidence, and mutation gate
./scripts/verify rust-conformance

# Required Linux Rust contract
./scripts/verify rust

# Required Linux Go and Rust contracts
./scripts/verify pre-push
```

`scripts/verify` is the command source shared by local development and Linux
GitHub Actions. The Rust mode may fetch locked registry dependencies first;
formatting, compilation, tests, documentation, and packaging then run offline.
Native macOS and Windows artifact checks remain CI-owned.

### Repository verification

The private Go verifier requires:

- exact reviewed Rust and CLI name coverage for 85 logical operations and 167
  physical operations;
- one applicable obligation set per physical operation and representative
  executable JSON, XML, and ZIP coverage;
- retained fixture provenance, SHA-256 digest, and exact bounded byte size;
- exact canonical-to-generated physical-operation coverage;
- stable logical pairing, response routing, and generated request vectors;
- deterministic owned SDK and CLI projections;
- approved Cargo metadata, tracked package inventories, and aligned versions;
- release-selected source provenance matching the committed canonical bundle;
  and
- approved workflow, permission, action-pin, and Release Please policy.

It never contacts OpenDART and does not replace Cargo compilation.

The focused `rust-conformance` mode combines that repository inventory and
fixture verification with the private handwritten pilot and its deterministic
fault adapters. It requires only committed local inputs and locked offline
Cargo dependencies. It neither selects nor certifies the generated client as
an oracle.

### Cargo verification

The pinned stable toolchain runs:

- rustfmt over handwritten Rust;
- Clippy with warnings denied for all features and no default features;
- workspace tests plus explicit no-default-features tests;
- rustdoc with warnings denied;
- WebAssembly compilation with default and no default features;
- native dependency-graph assertions;
- the adversarial reqwest feature-unification package;
- exact package-inventory checks and workspace packaging; and
- a clean CLI installation from the workspace.

The generated SDK module is intentionally `#[rustfmt::skip]`; byte-for-byte
generator freshness is its formatting gate. Every Cargo compile gate still
builds generated code.

The declared MSRV independently runs locked all-features and
no-default-features checks and metadata loading. The toolchain pins and Cargo
manifests are authoritative for exact versions.

The no-default-features SDK graph must remain free of `reqwest`, Tokio, Hyper,
TLS, proxy, DNS, and streaming-runtime dependencies. The WebAssembly graph must
likewise exclude the native client and runtime packages.

### Package contents

`sdk/rust/package-files.txt` and
`sdk/rust/opendart-cli-package-files.txt` are reviewed golden inventories.
Packaging must include the normalized manifest, workspace lock, public source,
reviewed generated code, package documentation, applicable tests and fixtures,
and Cargo-generated provenance. It must exclude generator implementation,
repository-private specification inputs, credentials, and local artifacts.

Workspace packaging verifies the exact local SDK dependency used by the CLI; it
does not grant publication authority to either package.

### Credential-free and live boundaries

The ordinary Rust gate uses loopback fixtures and no OpenDART credential. It
proves request serialization, source-envelope behavior, exact-byte streaming,
and the fixed transport contract under hostile dependency feature unification.

The Rust CLI live smoke is a separate opt-in producer. It reads the key only
when both its explicit live gate and `OPENDART_API_KEY` are present, performs
only reviewed read-only structured and binary calls, and asserts structure
rather than business data. Protected CI builds the reviewed Go and Rust
executables before the credential-bearing step and runs only those binaries
afterward, so dependency build scripts never inherit the secret.

Local credentialed probes must use `scripts/with-opendart-env`; never read,
print, or source `.env.local`.

## Versions and provenance

The repository has independent release components:

1. The canonical specification bundle uses root `vX.Y.Z` tags.
2. The `opendart` crate uses `opendart-vX.Y.Z` tags and Rust API SemVer.
3. The `opendart-cli` package uses `opendart-cli-vX.Y.Z` tags and CLI contract
   SemVer.

They may release from the same commit but never share a version by implication.
A packaged crate identifies:

- its Cargo version;
- its exact Git revision through `.cargo_vcs_info.json`;
- the selected semantic specification source release, when applicable;
- the independently selected canonical bundle SHA-256;
- the generator schema; and
- the deterministic SDK projection SHA-256.

Generated freshness uses the SDK projection checksum. A specification change
outside that projection does not rewrite or release the crate. Release
verification proves that the selected source tag contains the canonical source
inputs without asserting that a later generated bundle is byte-identical to
the tag's bundle.

## Compatibility policy

Below `1.0.0`, compatible fixes are patches, compatible public additions are
minors, and breaking changes are majors. At and after `1.0.0`, standard SemVer
applies.

| Change | Rust impact |
| --- | --- |
| Documentation or private refactor with identical behavior | None |
| Internal client fix with an unchanged public contract | Patch |
| New operation, optional input, or open status constant | Minor |
| Required input, serialization change, public rename/removal, or narrower wire type | Major |
| MSRV or guaranteed transport-policy change | Compatibility review |

Generated changes are not automatically compatible. Review the public Rust
surface and request behavior, not only the source OpenAPI label.

## Release Please ownership

The specification, SDK, and CLI are separate Release Please components. Each
Rust component owns its package manifest, changelog, component-qualified tag,
and matching workspace-lock entry. SDK version updates also maintain the
compatibility lock and the CLI's marked exact local SDK pin without changing
the CLI version or changelog.

`.release-please-manifest.json` already contains the SDK component. The CLI
component is configured for ownership but remains absent from the manifest and
proposal eligibility until its separate publication and recovery contract is
enabled. The manifest and `release-please-config.json` are authoritative; do
not describe the SDK as awaiting manifest enrollment.

Specification-only commits cannot create a Rust release, and Rust-only commits
cannot create a specification release. A component's path-qualified Release
Please outputs never authorize another component.

## SDK release evidence

[RELEASING.md](../../RELEASING.md) is the single owner of release
authorization, bootstrap credentials, automated stage boundaries, interrupted
release recovery, and the selected trusted-publishing transition.

The SDK-specific release evidence is:

- a credential-free candidate package built from the exact reviewed revision;
- the reviewed package inventory and both normalized and original Cargo
  manifests;
- Cargo VCS metadata plus matching specification, bundle, generator-schema,
  and SDK-projection provenance;
- the accepted crates.io checksum and archive matching that candidate;
- a clean consumer built against the exact registry version; and
- the public source documentation for that same version.

Crates.io versions are immutable, and `cargo publish` cannot upload a previously
built archive. Pre-publication review and post-attempt registry reconciliation
are therefore both required. A local candidate, manifest version, generated
changelog, draft, or command result alone is not release evidence.

The live presence of a token, draft, tag, registry version, or documentation
build belongs only in the
[Public Rust SDK plan](../../plans/rust/public-rust-sdk.md).
