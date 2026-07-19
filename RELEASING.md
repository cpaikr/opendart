# Release policy

This repository has two independently versioned public products:

- the canonical OpenAPI bundle, released from the root component as `vX.Y.Z`;
- the `opendart` Rust crate, prepared from
  `sdk/rust/crates/opendart` as `opendart-vX.Y.Z`.

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

At and after `1.0.0`, both products use standard SemVer. A change described as
a fix can still be breaking. Generated changes require compatibility review;
their source does not make them automatically safe.

## Eligibility and ownership

Root specification eligibility requires a material bundle change that matches a
fresh build. Formatting, tooling, CI, test, and prose changes are not releasable.
The root component excludes repository-only paths, including `sdk/`.

Rust eligibility requires a material change beneath the crate component path.
Its Release Please component owns `Cargo.toml`, the crate `CHANGELOG.md`, the
workspace-lock package version, and component-qualified tag. Specification and
Rust versions may differ even when both products change in one commit.

The Rust manifest may be absent from `.release-please-manifest.json` before the
first component release. Release Please proposes bootstrapping it at `0.1.0`;
pre-seeding that entry would falsely claim the version was already released.
The repository guard rejects a Rust manifest entry until work 6 is complete, so
do not merge a Rust component Release Please PR before that gate is deliberately
replaced by the reviewed publication flow.

## Verification gate

Before merging an implementation or Release Please PR, require review,
conversation resolution, and the `verify` job. Verification runs the Go
repository gate plus pinned stable, MSRV, all-features, no-default-features,
documentation, compatibility, offline, and exact package-content Cargo gates.

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

## Rust release boundary

The current configuration prepares a Rust-aware draft component release and
keeps the crate manifest and workspace lock aligned. It does not publish to
crates.io and has no registry credential, trusted-publishing permission, or
`cargo publish` command.

Work 6 must add publication as a separate guarded flow. It may authorize a
publish only from the exact Rust component output and immutable target revision,
never from the generic root release or a specification-only change. It must
detect and validate an already-existing draft for the exact Rust component tag
before relying on fresh path-qualified Release Please outputs, package and
dry-run without credentials first, publish once, download the registry artifact,
and verify its checksum, provenance, normalized manifest, and file contents
before finalizing the component release.
