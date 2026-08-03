# Public agent-first OpenDART CLI

## Outcome

Publish the `opendart-cli` crate so its `opendart` binary exposes every logical
OpenAPI operation through a generated, non-interactive CLI interface. Commands
and discovery use explicit CLI-owned product names and protocol identities;
private exhaustive dispatch calls the handwritten SDK, emits its source-shaped
compact JSON, and preserves binary bodies at explicit paths without duplicating
endpoint contracts.

## Implemented state

- [ADR 0003](../../docs/decisions/0003-agent-first-opendart-cli.md)
  records the accepted product and release-order decisions.
- The binary-only crate, shared generator projection, keyless discovery,
  generated typed dispatch, structured execution, and transactional binary
  artifact path are implemented. The
  [architecture](../../docs/rust-cli/architecture.md) owns those module seams,
  and the [public contract](../../docs/rust-cli/public-contract.md) owns
  observable behavior.
- Canonical request constraints flow through OpenAPI, SDK preparation, CLI
  parsing, discovery, and sanitized errors. SDK and CLI projections remain
  independently checksummed and verified.
- Credential-free verification covers generated freshness, package inventory,
  package dry runs, locked source installation, process behavior, and
  compatibility loopbacks. Clean source installs run on Linux, macOS, and
  Windows. Credentialed smoke coverage is separately and explicitly gated.
- The package has its own README, changelog, lockfile, exact local SDK pin, and
  prepared local registry-artifact verifier. It is package-ready but
  unpublished.
- Release Please defines an independent CLI component and tag identity, but the
  CLI excludes its own path from proposal eligibility. The release manifest
  contains the SDK component but no CLI component, the release guard rejects
  CLI release output, and the reusable crate workflow is still SDK-specific.
  No current workflow can publish `opendart-cli`.
- The SDK's generated conformer is scheduled for replacement by the
  [handwritten contract](handwritten-sdk-contract-and-conformance.md) and
  combined [implementation and cutover](handwritten-sdk-conformer.md) plans.
  The retained CLI interface projection may own commands, discovery, and
  exhaustive typed wiring, but it must not expose Rust implementation names or
  own provider validation, request serialization, or response decoding.

## Source-publication gate

CLI publication depends on the public SDK, and an SDK prerelease does not
satisfy that dependency. The gate opens only after all of the following are
true:

1. An exact non-prerelease `opendart` version is present on crates.io and the
   accepted registry artifact, checksum, manifests, contents, inventory, and
   source provenance have been verified after the handwritten SDK cutover.
2. Clean consumers install and exercise that exact SDK version, and its
   component-isolated trusted-publishing and interrupted-release recovery paths
   have succeeded.
3. The CLI's exact SDK dependency is updated to that verified immutable version
   before a CLI release candidate is created.
4. Registry ownership and name availability are rechecked immediately before
   granting CLI publication authority.

Until this gate is satisfied, do not add the CLI release-manifest entry, enable
CLI proposal eligibility, create a CLI publication environment or token,
publish the crate, or finalize a CLI release.

## Remaining source-publication work

### Enable the isolated CLI component

- Parameterize the proven crate-release workflow and release guard for the
  exact CLI package path, component outputs, package inventory, tag identity,
  and consumer checks.
- Use a distinct `crates-io-opendart-cli` environment. Do not share SDK
  component outputs, registry credentials, or publication evidence.
- Enable the CLI Release Please path and admit only its reviewed manifest
  transition after the SDK gate. Preserve exact candidate-SHA and
  interrupted-draft recovery.

### Bootstrap and verify the crate

- Follow the beta-to-stable bootstrap in the
  [verification and release guide](../../docs/rust-cli/verification-and-release.md).
  Use a short-lived environment-scoped API token only to create the crate,
  verify the accepted beta, revoke the token, configure the exact trusted
  publisher, and remove the token path before another release.
- Package and dry-run without credentials. Publish at most once from the exact
  reviewed candidate, reconcile timeouts or an already-present version, and
  compare the accepted registry artifact with the candidate before finalizing
  the draft.
- Install the exact registry version with `--locked` in clean Linux, macOS, and
  Windows environments and run `--version` plus keyless discovery. Keep any
  real OpenDART call outside the release gate.

### Promote and adopt

- Promote the reviewed CLI beta to a non-prerelease version through OIDC, then
  remove temporary `release-as` and prerelease bootstrap configuration.
- Keep Release Please PR review and merge as the routine human gate. Publication
  and finalization remain automated, component-isolated, and recoverable.
- Record actual source-package consumer evidence before expanding aliases,
  input modes, output transformations, or distribution channels.

## Completion criteria

- Each exact reviewed `opendart-cli` version is published at most once from an
  immutable revision with matching registry checksum, package contents,
  manifests, inventory, tag, and provenance.
- Clean supported hosts install the registry package with its lockfile and
  keyless discovery matches the retained canonical CLI interface projection.
- The release workflow receives no OpenDART credential, and only its protected
  CLI publication job receives temporary or trusted crates.io authority.
- SDK, CLI, and specification versions, changelogs, tags, component outputs,
  package evidence, and publication environments remain independent.

## Boundaries

- The [public SDK plan](public-rust-sdk.md) owns SDK registry
  publication and verification. This plan consumes its verified non-prerelease
  artifact; it does not broaden SDK release authority.
- Initial CLI distribution is crates.io source only. Prebuilt archives,
  installers, and package-manager publication remain in the
  [prebuilt-release task](../../tasks/rust/opendart-cli-prebuilt-releases.md).
- CLI generation and ordinary verification remain offline and
  credential-free. Generated CLI code may own explicit CLI grammar, discovery,
  and exhaustive private typed wiring, with its own checksum and freshness
  gate. Public discovery must not expose SDK field or response-type symbols,
  and the projection must not own provider serialization, validation, or
  response decoding. A real OpenDART smoke call is optional post-release
  evidence, never a publication or finalization gate.

## Next action

Wait for the handwritten SDK plans to establish the CLI-owned interface and
complete the handwritten-only product cutover, then wait for the SDK task to
publish and verify the exact non-prerelease `opendart` registry artifact. Align
the CLI's exact dependency pin only after that gate; do not bootstrap CLI
publication from generated-SDK or prerelease evidence alone.
