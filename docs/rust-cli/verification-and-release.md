# Rust CLI verification and release

## Source-package gate

The supported pre-publication distribution is the reviewed source checkout at
`sdk/rust/crates/opendart-cli`. Its manifest exact-pins the local `opendart`
dependency and the workspace `serde_json` behavior, includes its own lockfile,
and authorizes only crates.io as a future registry.

Credential-free verification checks the exact package inventory, packages the
SDK and CLI workspace together, and installs the CLI from source with
`--locked --offline`. Linux, macOS, and Windows runners execute both
`opendart --version` and keyless `opendart operations list` from a clean install
root. A local POSIX equivalent is:

```sh
cargo +1.97.1 fetch --locked --manifest-path sdk/rust/Cargo.toml
install_workspace="$(mktemp -d)"
CARGO_TARGET_DIR="${install_workspace}/target" \
  cargo +1.97.1 install --locked --offline \
  --path sdk/rust/crates/opendart-cli --root "${install_workspace}/root"
"${install_workspace}/root/bin/opendart" --version
"${install_workspace}/root/bin/opendart" operations list
```

Moving the SDK or JSON encoder pin requires explicit CLI compatibility review.
The SDK Release Please component updates the CLI's exact local dependency pin,
but that dependency-only update does not change the CLI version or changelog.

## Credentialed developer checks

Varlock 1.12.0 appends its own child-failure diagnostic to standard output when
the wrapped command exits nonzero. Do not run a CLI failure-contract assertion
as `varlock run -- opendart ...`: the wrapper turns the CLI's single JSON output
document into a concatenated stream. Instead, use Varlock to run a test harness
that invokes and captures the CLI as its child, or provide the credential to the
CLI through another inherited environment that preserves its process channels.
This is Varlock behavior, not part of the public CLI contract; recheck it before
updating this note.

## Independent release ownership

Release Please has an independent `sdk/rust/crates/opendart-cli` component. It
owns the CLI Cargo version, changelog, `opendart-cli-vX.Y.Z` tag identity, and
matching workspace-lock entry. CLI-only changes do not belong to the root
specification or `opendart` SDK components.

The CLI and SDK entries remain absent from `.release-please-manifest.json`
until their publication flows are authorized and recoverable. The current
release workflow has no crates.io credential, trusted-publishing permission,
or `cargo publish` command. A Release Please PR that tries to bootstrap either
unpublished component therefore fails the repository guard and must not merge.

## Prepared artifact comparison

`opendart-tool verify-crate-artifact` is the local-only post-publication
verification seam. It never downloads, queries, or publishes a package. Its
caller must supply distinct local candidate and accepted `.crate` files, the
accepted registry checksum, exact package metadata and revision, and the
reviewed inventory.

The verifier checks the accepted checksum, safe bounded gzip/tar structure,
zero padding and tails, full expanded tar identity, exact files and contents,
both Cargo manifests, and clean Cargo VCS metadata. Reports and failures omit
local paths and file contents. Work 8 must acquire the accepted artifact and
checksum through separately authorized registry logic before invoking this
command.

## Stop gate and interrupted-release recovery

Work 7 ends before registry ownership checks or publication. Resume the public
Rust SDK task at its crates.io publication work, publish and verify the exact
`opendart` version, and only then return to CLI work 8. Reconfirm that the CLI's
already-reviewed exact SDK pin matches that immutable registry version.

The later CLI publication change must recover conservatively:

1. Detect an existing `opendart-cli-vX.Y.Z` draft before relying on fresh
   Release Please outputs, and require its target to be the reviewed immutable
   revision.
2. Reproduce the source package and reviewed inventory without credentials.
3. Recheck crates.io name and owner state immediately before explicit
   authorization. A conflict stops for a product decision.
4. If the version is absent, publish once through dedicated least privilege.
   If it is present after interruption, do not republish.
5. Download the accepted crate and checksum, run the prepared local verifier,
   perform a clean registry install and keyless discovery, and inspect docs.rs.
6. Finalize only the matching GitHub component draft after every immutable
   registry check passes. A mismatch leaves the draft unpublished for manual
   investigation; tags, assets, and registry versions are never replaced.

Prebuilt binaries, installers, and package-manager releases remain outside this
source-distribution flow.
