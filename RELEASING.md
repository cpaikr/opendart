# Release policy

This repository has three independently versioned public products:

- the canonical OpenAPI bundle, released from the root component as `vX.Y.Z`;
- the `opendart` Rust crate, prepared from
  `sdk/rust/crates/opendart` as `opendart-vX.Y.Z`; and
- the `opendart-cli` Rust crate, prepared from
  `sdk/rust/crates/opendart-cli` as `opendart-cli-vX.Y.Z`.

Release Please owns each component's version and changelog. Do not manually
edit its manifest state or generated changelogs, move published tags, replace
immutable assets, or publish outside the reviewed workflow.

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

Classify the Rust component by its public API and wire behavior:

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

At and after `1.0.0`, all products use standard SemVer. A change described as
a fix can still be breaking. Generated changes require compatibility review;
their source does not make them automatically safe.

## Eligibility and ownership

Specification eligibility requires a material change beneath the
`openapi/generated` component path that matches a fresh bundle build.
Formatting, tooling, CI, test, prose, and SDK changes are not releasable.

Each Rust component requires a material change beneath its own crate path. Its
Release Please component owns that crate's `Cargo.toml`, `CHANGELOG.md`,
workspace-lock package version, and component-qualified tag. Specification,
SDK, and CLI versions may differ even when several products change in one
commit.

An SDK version proposal also updates the CLI manifest's marked exact local SDK
pin. That workspace-resolution update does not change the CLI version or
changelog and must preserve the exact-pin operator. Moving the SDK or JSON
encoder pin requires explicit CLI compatibility review.

The SDK and CLI entries are absent from `.release-please-manifest.json` before
their first component releases. Pre-seeding either entry would falsely claim a
version was already released. The repository guard rejects both transitions:
do not merge an SDK Release Please PR before SDK work 6 authorizes publication,
or a CLI Release Please PR before dependent CLI work 8 does so.

## Verification gate

Before merging an implementation or Release Please PR, require review,
conversation resolution, and the `verify` job. Verification runs the Go
repository gate plus pinned stable, MSRV, all-features, no-default-features,
documentation, compatibility, offline, and exact package-content Cargo gates.
The CLI is also installed with `cargo install --locked --offline --path` into a
clean root and exercised on Linux, macOS, and Windows.

Release Please uses the repository `GITHUB_TOKEN`, so its own PR may require a
manual Verify dispatch:

```sh
gh workflow run verify.yml --ref <release-please-branch>
```

Do not merge the release PR until that run passes and the proposed component,
version, changelog, tag, Cargo lock, and artifact scope are correct.

## Specification release flow

1. Compare the candidate bundle with the latest specification tag and classify
   the material public-contract diff.
2. Merge the implementation after review and Verify pass.
3. Review and verify the root Release Please PR, then merge it.
4. The release workflow reruns Verify, creates or resumes the draft release,
   attaches `openapi.bundle.yaml` and its checksum, and publishes the release.
5. Confirm the immutable release targets the intended commit and assets.

If a run fails after draft creation, rerun the failed workflow or merge a repair
and let the next run resume the draft. Never move a published tag or replace an
immutable asset.

## Rust source-package release boundary

The current configuration prepares independent Rust-aware component proposals
and keeps each crate manifest and workspace-lock entry aligned. It does not
publish to crates.io and has no registry credential, trusted-publishing
permission, or `cargo publish` command.

SDK work 6 must first add publication as a separate guarded flow for `opendart`.
After that immutable artifact is verified, CLI work 8 may add separate authority
for `opendart-cli`. Each flow must authorize only its exact path-qualified
component output and immutable target revision, detect an already-existing
component draft before relying on fresh action outputs, package without
credentials first, publish at most once, and verify the downloaded registry
artifact before finalizing the matching draft.

The prepared CLI artifact comparator is local-only and accepts an already
downloaded candidate, accepted crate, registry checksum, reviewed inventory,
and expected VCS metadata. It grants no acquisition or publication authority.
The detailed stop gate and interrupted-run procedure are in the
[CLI verification guide](docs/rust-cli/verification-and-release.md).
