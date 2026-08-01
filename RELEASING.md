# Release policy

Release Please manages three independently versioned components, but they are
at different publication states:

| Component | Release identity | Current state |
| --- | --- | --- |
| Canonical OpenAPI bundle | root changelog and `vX.Y.Z` | Public release path is active. |
| `opendart` Rust crate | `sdk/rust/crates/opendart` and `opendart-vX.Y.Z` | Release automation is connected. The unpublished `0.1.0-beta.1` candidate is superseded and guarded as a recovery tombstone while a replacement beta awaits review. |
| `opendart-cli` Rust crate | `sdk/rust/crates/opendart-cli` and `opendart-cli-vX.Y.Z` | Independent ownership is configured for future use, but release eligibility and publication are disabled. |

The [Rust roadmap](ROADMAP_RUST.md) and
[public SDK task](tasks/rust/public-rust-sdk.md) own live delivery status.
This policy owns durable release rules, not incident or run history.

## Sources of truth

`release-please-config.json` defines component paths, version policy, tag
format, changelog ownership, and proposal behavior.
`.release-please-manifest.json`, package manifests, lockfiles, generated
changelogs, and GitHub drafts record Release Please proposal state. They may
advance before downstream publication succeeds, so none is proof that a
component is publicly available.

Publication evidence is component-specific:

- A specification release requires the immutable Git tag, published GitHub
  release, bundle, and matching checksum.
- A Rust release requires the accepted crates.io artifact and checksum,
  successful clean-consumer and docs.rs verification, the immutable component
  tag, and the finalized matching GitHub release.
- A draft, generated changelog entry, manifest version, or publish command
  result alone is not publication evidence. Reconcile crates.io after every
  attempt because a reported failure can follow an accepted upload.

Release Please owns generated versions, manifest state, changelogs, tags, and
release notes. Do not edit those outputs manually, move a published tag,
replace immutable assets, or publish outside the reviewed workflow.

## Authorization

Use `$release-please-release` for release preparation, proposal review, and
every release operation. Before merging a Release Please proposal, publishing,
creating or moving a tag, finalizing a release, or performing another release
side effect:

1. classify the exact component diff and SemVer impact;
2. present the exact component and version;
3. obtain explicit confirmation for that version and the specific operation.

Approval for one component or operation does not authorize another. Reviewing
and merging a component-specific Release Please PR is the routine human gate;
the workflow owns subsequent verification, publication, reconciliation, and
finalization. Do not run `cargo publish`, create or move a tag, edit the
manifest, or finalize a draft manually unless an explicitly authorized
recovery procedure requires that exact action.

## Compatibility classification

The OpenAPI dialect and upstream snapshot identifiers are not release
versions: `openapi: 3.2.0` selects the dialect, while `info.version` and
`x-opendart.source.checkedAt` identify guide evidence.

Below `1.0.0`, the specification component uses this policy:

| Impact | Commit input | Version impact |
| --- | --- | --- |
| Repository-only | `chore:`, `test:`, `ci:`, `docs:` | None |
| Compatible contract fix | `fix(openapi):` | Patch |
| Compatible contract addition | `feat(openapi):` | Patch |
| Breaking contract change | `feat(openapi)!:` plus `BREAKING CHANGE:` | Minor |

Classify the Rust SDK by its public API and wire behavior:

| Impact | Examples | Version impact |
| --- | --- | --- |
| None | Docs, tests, private refactor, identical generation | None |
| Compatible fix | Internal bug fix without contract change | Patch |
| Compatible addition | Operation, optional input, open status constant | Minor |
| Breaking | Required input, serialization change, public removal or narrowing | Major |

Classify the CLI independently by its invocation, discovery, output, exit, and
artifact contracts. A compatible command addition is minor; a corrected
implementation with the same contract is patch; removing or changing accepted
arguments, JSON envelopes, exit meanings, or artifact guarantees is breaking.

At and after `1.0.0`, every component uses standard SemVer. A change described
as a fix can still be breaking. Generated changes require compatibility review;
their source does not make them automatically safe.

## Eligibility and ownership

Specification eligibility requires a material change beneath
`openapi/generated` that matches a fresh bundle build. Formatting, tooling, CI,
tests, prose, and SDK changes are not specification release inputs.

A Rust component requires a material change beneath its own crate path.
Release Please owns that crate's `Cargo.toml`, `CHANGELOG.md`, workspace-lock
package version, manifest entry once proposed, and component-qualified tag.
Component versions may differ even when several products change in one commit.

An SDK version proposal also updates the CLI manifest's marked exact local SDK
pin. That workspace-resolution update does not change the CLI version or
changelog and must preserve the exact-pin operator. Moving the SDK or JSON
encoder pin requires explicit CLI compatibility review.

The CLI component remains ineligible while its own path is excluded, its
manifest entry is absent, and the release workflow rejects CLI release output.
Do not pre-seed or bypass those gates. Enabling it requires separately reviewed
CLI release work and an independent publication path.

## Verification gate

Before merging an implementation or Release Please PR, require completed
review, resolved conversations, and a successful `verify` result. Main branch
protection requires that stable context.

For ordinary pull requests, `Verify / verify` succeeds only when the
independent Go, Rust, macOS artifact, and Windows artifact jobs all succeed.
The repository-owned commands and exact job policy live in `scripts/verify`,
`.github/workflows/verify.yml`, and `internal/releaseguard`.

Release Please proposals do not receive the ordinary pull-request workflow
from the repository `GITHUB_TOKEN`. The trusted `main` orchestrator therefore
dispatches `verify.yml` and `full-race.yml` for the exact proposal SHA,
revalidates the proposal identity, ancestry, and generated-file scope, and
posts the combined `verify` status. A missing, stale, skipped, ambiguous, or
failed exact-SHA run is a merge blocker.

Manual dispatch is for incident recovery only. A manual run does not replace
the orchestrator's combined status; restore or rerun the trusted orchestrator
before treating the proposal as mergeable.

The full-race workflow also runs weekly. Treat a scheduled failure as a
maintainer-owned regression: reproduce it with `./scripts/verify exhaustive`,
land a reviewed repair, rerun the full-race workflow on the candidate, and
retain the successful run as incident evidence.

## Specification releases

1. Compare the candidate bundle with the latest specification tag and classify
   the material public-contract diff.
2. Merge the implementation after review and Verify pass.
3. Review the root Release Please PR, present and confirm its exact version,
   and merge only after its exact-SHA release gates pass.
4. Let the release workflow rerun Verify, create or resume the matching draft,
   attach `openapi.bundle.yaml` and its checksum, and publish the release.
5. Confirm that the immutable release targets the intended commit and assets.

If a run stops after draft creation, repair or rerun the workflow against the
same release identity. Never move a published tag or replace an immutable
asset. Specification releases alone may become repository-global Latest.

## Rust crate releases

Only the SDK is connected to the reusable crate-publication workflow. The CLI
remains a stop condition. Its source-package gate and future release boundary
are documented in the
[CLI verification guide](docs/rust-cli/verification-and-release.md).

### Current bootstrap boundary

The configured first-release implementation is token-based, not OIDC-based.
The SDK publish job is bound to the protected `crates-io-opendart` environment
and uses only its `CARGO_REGISTRY_TOKEN`; the repository owner variable
supplies the expected crates.io identity. The environment and release guard
restrict the job to the approved workflow and `main`.

Credential access for this bootstrap must use `op-agent`. When provisioning is
explicitly authorized, inject the existing 1Password value directly through
`op-agent` to `gh secret set`; do not use plain `op`, print or export the value,
place it in a command argument, or persist it in the repository or shell
configuration. Verify only secret metadata with `gh secret list`.

The [SDK verification guide](docs/rust-sdk/verification-and-release.md) owns
the crate's compatibility, package, and provenance evidence. Authorization,
bootstrap, recovery, and automation remain owned here.

### Automated flow

After a component-specific proposal is confirmed and merged:

1. Accept only that component's path-qualified Release Please version, tag, and
   candidate SHA, or an exactly matching interrupted draft recovered by the
   orchestrator. A generic push, another component, or a manually supplied
   version grants no publication authority.
2. Check out and attest the immutable candidate, run credential-free
   verification, package and dry-run without registry authority, and upload
   immutable candidate evidence.
3. Enter the component's protected environment, recheck package name, owner,
   version, and registry state, then publish at most once.
4. Reconcile after every publish attempt. Cargo may report failure after an
   upload succeeds, so command status alone is not registry evidence.
5. Download the accepted crate and require matching registry checksum,
   provenance, normalized manifest, inventory, and unpacked contents. Prove a
   clean exact-version consumer and wait for docs.rs.
6. Finalize only the matching GitHub draft with the expected beta or stable
   flag. Rust releases must use `latest=false`.

Authority remains split:

| Stage | Authority |
| --- | --- |
| Release Please | Repository contents, issues, and pull requests write; no registry credential |
| Proposal dispatch and reporting | Actions/status authority only; no registry credential |
| Verify and package | Repository read and workflow-artifact upload; no registry or release credential |
| Publish crate | Repository read and only the protected crates.io bootstrap credential |
| Reconcile accepted crate | Public registry and documentation reads; no publication credential |
| Finalize GitHub draft | Repository contents write; no crates.io credential |

On interruption, resume only the same component, version, tag, and candidate
SHA. Query crates.io before deciding that publication is still necessary.
Never create a replacement tag, republish an existing version, or treat a
draft as permission to bypass the pipeline. The accepted registry artifact,
verification report, docs.rs result, and finalized release are the completion
evidence.

### Selected steady state

After the first accepted SDK artifact is verified, revoke and remove the
bootstrap token, archive its source item, and configure crates.io trusted
publishing for the exact repository, workflow, and environment. A separately
reviewed hardening change must replace token authentication with a
commit-pinned OIDC action and only the required `id-token: write` permission.
Do not retain a token fallback.

Stable SDK promotion and later CLI publication remain separately reviewed
delivery work. Their selected version inputs, eligibility commits, and
component-specific gates are authoritative only in the linked task and
component guides until implemented.
