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

The repository-local `scripts/with-opendart-env` wrapper validates the ignored
`.env.local` file and injects `OPENDART_API_KEY` only into its child process. It
uses `exec`, so the CLI's exit status and standard streams remain unchanged.
This wrapper is developer tooling and does not add dotenv loading to the public
CLI contract.

## Independent release ownership

Release Please has an independent `sdk/rust/crates/opendart-cli` component. It
owns the CLI Cargo version, changelog, `opendart-cli-vX.Y.Z` tag identity, and
matching workspace-lock entry. CLI-only changes do not belong to the root
specification or `opendart` SDK components.

The CLI and SDK entries remain absent from `.release-please-manifest.json`
until their first authorized releases. The guard now admits only the aligned
SDK beta transition; it still rejects every CLI manifest entry and the CLI
component excludes its own path from proposal eligibility until work 9.

The setup enables separate Release Please PRs, so SDK and CLI approval cannot
be coupled to each other or to a specification release. The CLI
publication job may reuse the SDK crate-release workflow only after the SDK
prerelease and non-prerelease paths have proved its component isolation and
interrupted-run recovery. Reuse does not share component outputs, package
evidence, environments, or credentials.

## Prepared artifact comparison

`opendart-tool verify-crate-artifact` is the local-only post-publication
verification seam. It never downloads, queries, or publishes a package. Its
caller must supply distinct local candidate and accepted `.crate` files, the
accepted registry checksum, exact package metadata and revision, and the
reviewed inventory.

The verifier checks the accepted checksum, safe bounded gzip/tar structure,
zero padding and tails, full expanded tar identity, exact files and contents,
both Cargo manifests, and clean Cargo VCS metadata. Reports and failures omit
local paths and file contents. Work 9 must acquire the accepted artifact and
checksum through separately authorized registry logic before invoking this
command.

## Stop gate and interrupted-release recovery

Work 8 ends before registry ownership checks or publication. Resume the public
Rust SDK task at its crates.io publication work, publish and verify the exact
`opendart` version, and only then return to CLI work 9. Reconfirm that the CLI's
already-reviewed exact SDK pin matches that immutable registry version.

Work 9 must add the CLI path to the proven automated pipeline and recover
conservatively:

1. Consume only the exact
   `sdk/rust/crates/opendart-cli--release_created`, `--tag_name`, `--version`,
   and `--sha` outputs, or an exactly matching interrupted
   `opendart-cli-vX.Y.Z` draft detected before Release Please runs. Require its
   target to be a full immutable candidate SHA that is an ancestor of current
   `main`; keep that candidate fixed if a newer workflow repair resumes it.
2. Confirm that the CLI's exact SDK dependency equals the already verified
   non-prerelease registry version. Reproduce the CLI source package, lockfile,
   and reviewed inventory without credentials; run locked package and publish
   dry-runs before entering an authority-bearing job.
3. Enter only the `crates-io-opendart-cli` environment and recheck name, owner,
   and exact-version state immediately before publication. A name conflict or
   unexpected owner stops for a product decision.
4. If the version is absent, publish once with
   `cargo +1.97.1 publish --locked --no-verify` at the exact candidate SHA. The
   credential-free dry-run already performed the build, so `--no-verify` keeps
   registry authority out of dependency and build scripts. If Cargo times out
   or the version is already present after interruption, reconcile instead of
   republishing.
5. Download the accepted crate and registry checksum, run the prepared local
   verifier, and require exact checksum, manifests, contents, inventory, and VCS
   provenance. Then install the exact registry version with `--locked` in clean
   Linux, macOS, and Windows roots, run keyless discovery, and wait for docs.rs.
6. Give the finalizer GitHub contents-write authority but no crates.io
   credential. Finalize only the matching component draft after every immutable
   registry check passes and its beta/stable prerelease flag matches the
   proposal. Explicitly keep `latest=false`; repository-global Latest belongs
   to the independent specification convention. A mismatch leaves the draft
   unpublished; tags, assets, and registry versions are never moved or replaced.

A real OpenDART smoke call is optional post-release evidence, not a publication
or finalization gate. Run it only through the separately authorized protected
live path after the release succeeds. The crate pipeline never receives
`OPENDART_API_KEY`, and volatile upstream service health cannot block or roll
back an otherwise verified immutable release.

The first `opendart-cli` version follows the same beta-to-stable bootstrap as the
SDK. Work 9 temporarily configures `versioning: prerelease`, `prerelease: true`,
`prerelease-type: beta`, and package-scoped `release-as: 0.1.0-beta.1`, then
publishes the reviewed beta with a short-lived API token capable of creating the
new crate in its protected component environment. After public verification,
revoke the token, configure the trusted publisher for the exact
repository/workflow/environment, and replace the token path with commit-pinned
OIDC authentication. Keep prerelease versioning, set `prerelease: false`, and
update `release-as` to `0.1.0` for the explicitly reviewed stable promotion.
That delivery also needs a release-eligible Conventional Commit under
`sdk/rust/crates/opendart-cli` recording the verified beta and stable support
contract; out-of-path configuration alone does not make the component eligible
for a Release Please proposal, and an empty bump commit is not acceptable.
After stable succeeds, remove `release-as` and the three temporary prerelease
fields. Do not retain a token fallback; Release Please PR review and merge
remains the only routine human gate after bootstrap.

Prebuilt binaries, installers, and package-manager releases remain outside this
source-distribution flow.
