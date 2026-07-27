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
version was already released. The repository guard admits the SDK entry only
when the first beta proposal aligns the manifest, crate, and lock versions; it
continues to reject every CLI entry until dependent CLI work 9 adds and proves
that publication path.

## Automation contract

The guarded Rust release path is event-driven. In steady state, the only routine
manual action is reviewing and merging one component-specific Release Please
PR. That merge is the maintainer's explicit confirmation of the exact component,
version, changelog, tag, and immutable source revision. Everything after it is
workflow-owned:

```text
Conventional commits
  -> separate component Release Please PR
  -> maintainer review and merge
  -> exact-SHA verification and credential-free packaging
  -> crates.io publication authority
  -> accepted-artifact and docs.rs verification
  -> matching draft GitHub release finalization
```

The setup change enables separate manifest release PRs so the specification,
SDK, and CLI cannot be approved as one aggregate release. It also extends the
repository guard to recognize only the approved jobs, component outputs,
permissions, environment, actions, commands, and recovery paths. Combined
Release Please PR #32 was closed without merging; after this setup reaches
`main`, its push may create or update independent proposals.

The setup sets both Rust `bump-minor-pre-major` and
`bump-patch-for-minor-pre-major` options to `false` while retaining the
specification's separate pre-1.0 policy. The release guard enforces that split
before any Rust proposal is mergeable. Because accumulated bootstrap
history already contains breaking commits, use the two temporary package-scoped
overrides described below instead of letting that history imply `1.0.0`.

Each Rust publication consumes only its own path-qualified Release Please
`release_created`, `tag_name`, `version`, and `sha` outputs, or an exactly
matching interrupted draft recovered before Release Please runs. A generic push,
another component release, a branch build, or a manually supplied version never
authorizes publication. Release Please documents both
[path-qualified outputs](https://github.com/googleapis/release-please-action#path-outputs)
and [separate manifest pull requests](https://github.com/googleapis/release-please/blob/main/docs/manifest-releaser.md).

Recovery starts from the component version in the manifest and the current
`main` history. With `draft: true` and `force-tag-creation: false`, an
interrupted draft has no Git tag yet. Resume it before running Release Please
only when its exact tag name and full-SHA `targetCommitish` identify the reviewed
candidate and that SHA is an ancestor of the current workflow revision; the
candidate SHA remains fixed even when a newer repaired workflow resumes it.
When both known components have valid interrupted states, recover each from its
own path-qualified identity in the same run; neither component's state can
authorize the other. A
matching published tag at the current event SHA is reconciled as an idempotent
completion. A matching published tag that is a verified ancestor is the normal
previous release, so Release Please may process later component commits. A
tag-only state, duplicate tag matches, duplicate component recovery records, a
branch-shaped draft target, non-ancestor, or other version/tag/SHA mismatch
stops without another release or publish attempt.

Keep authority split by job:

| Job | Authority |
| --- | --- |
| Release Please | Repository contents, issues, and pull requests write; no workflow dispatch or registry credential |
| Dispatch PR verification | Actions write plus repository and pull-request read; no contents write or registry credential |
| Verify and package | Repository read plus workflow-artifact upload; no registry or release credential |
| Publish crate | Repository read and only the selected crates.io credential path |
| Verify accepted crate | Public registry and docs reads; no publication credential |
| Finalize GitHub draft | Repository contents write; no crates.io credential |

The finalizer also checks the draft's `prerelease` flag. A beta is published as
a GitHub prerelease; a stable crate release clears that flag. Neither Rust
component may become repository-global Latest, which remains the specification
release convention, so both Rust finalizer commands explicitly keep
`latest=false`. The release guard must reject reuse of the specification's
`gh release edit ... --latest` command for either crate.

The publication job uses a component-specific GitHub environment whose
deployment branch is restricted to `main`; the repository guard and trusted
publisher separately restrict the canonical workflow. For established crates
the job obtains a short-lived token through crates.io trusted publishing and
has only `contents: read` plus `id-token: write`; GitHub documents that OIDC
permission as token-request authority, not repository write authority. The
exact third-party authentication action must be commit-pinned before the
repository guard admits it.

crates.io requires the first version of a new crate before trusted publishing
can be configured. Bootstrap each crate once with a short-lived API token that
can create that new crate, stored only in the component's protected
environment. For the SDK, set repository variable
`OPENDART_CRATES_IO_OWNER` to the exact crates.io owner login and store the
token only as `CARGO_REGISTRY_TOKEN` in `crates-io-opendart`. Require an
environment reviewer for that first run, delete the secret immediately after
the accepted artifact is verified, configure the
trusted publisher for the exact repository/workflow/environment, and land the
OIDC-only follow-up before the next release. Do not retain an automatic fallback
to a long-lived token.

The reusable crate workflow binds its publish job to the literal
`crates-io-opendart` environment. Do not parameterize that environment or add a
`workflow_call` secret for the registry token; the protected environment is the
only bootstrap credential source.

Keep the bootstrap token's source copy in the `Developer` 1Password vault as
API Credential item `opendart crates.io bootstrap`, secret field `credential`.
Provision the GitHub environment secret without printing or exporting the
value:

```sh
op read -n 'op://Developer/opendart crates.io bootstrap/credential' \
  | gh secret set CARGO_REGISTRY_TOKEN \
      --repo cpaikr/opendart \
      --env crates-io-opendart
```

Verify only the secret metadata with `gh secret list`; never echo, log, or
place the token in a command argument. After the accepted crate is verified,
delete the GitHub environment secret, revoke the crates.io token, and archive
the 1Password item. The trusted-publishing cutover removes this token path from
subsequent releases.

The [crates.io trusted-publishing announcement](https://blog.rust-lang.org/2025/07/11/crates-io-development-update-2025-07/#trusted-publishing)
documents this first-release boundary.

Use `$release-please-release` for setup review and every release operation. It
requires the exact per-component version to be presented and explicitly
confirmed before any merge, publication, tag, or release side effect. The
normal merge approval satisfies the one routine release gate; one-time
environment approval is limited to each crate's bootstrap run.

## Verification gate

Before merging an implementation or Release Please PR, require review,
conversation resolution, and the stable aggregate `verify` job. It waits for
the independent Go, Rust, macOS artifact, and Windows artifact jobs and
succeeds only when all four succeed. The Go job runs normal tests, the audited
targeted-race set, and the repository gate; the Rust job runs the pinned stable,
MSRV, all-features, no-default-features, documentation, compatibility, offline,
and exact package-content Cargo gates. The CLI is also installed with
`cargo install --locked --offline --path` into a clean root and exercised on
Linux, macOS, and Windows.

Release Please uses the repository `GITHUB_TOKEN`, so its pull-request event
does not create a new `pull_request` workflow run under GitHub's token recursion
policy. A `workflow_run` event from a `GITHUB_TOKEN`-dispatched workflow is also
suppressed, so status reporting stays inside the trusted `main` orchestrator.
An `actions: write` job dispatches both `verify.yml` and `full-race.yml` for
every managed component PR head SHA and records the preceding run IDs. A
separate `actions: read` and `statuses: write` job then waits for exactly one
new GitHub Actions-created run of each workflow at that SHA. Before reporting,
it revalidates the open GitHub Actions-owned proposal, current `main` ancestry,
and exact generated-file scope. The `verify` commit status succeeds only when
both exact-SHA runs succeed. This bridges GitHub's token-recursion behavior to
branch protection without giving the credential-free verification workflows a
token or requiring manual approval of the suppressed `pull_request` run.
Update the reporter job's generated-file allowlist and releaseguard invariant
before enabling another component or changing a component's Release Please
outputs.
Add the stable `Verify / verify` aggregate to `main` branch protection. The
reported release-proposal `verify` status also covers the automatically
produced, release-only full-race run without requiring the expensive sweep on
every ordinary PR. The release skill and this runbook treat a missing, stale,
skipped, or failed exact-SHA dispatch as a merge blocker. See GitHub's
[token behavior](https://docs.github.com/en/actions/concepts/security/github_token).

For incident recovery only, maintainers may dispatch the workflows directly.
Manual runs alone cannot complete recovery: verify that both exact-SHA runs
succeed, then restore or rerun the trusted Release Please orchestrator until it
posts the combined `verify` commit status. Do not treat the proposal as
mergeable before that status exists.

```sh
gh workflow run verify.yml --ref <release-please-branch> \
  -f expected_sha=<full-release-please-head-sha>
gh workflow run full-race.yml --ref <release-please-branch> \
  -f expected_sha=<full-release-please-head-sha>
gh run list --workflow full-race.yml --branch <release-please-branch> \
  --event workflow_dispatch --limit 1
gh run watch <run-id> --exit-status
```

Do not merge the release PR until that run passes and the proposed component,
version, changelog, tag, Cargo lock, and artifact scope are correct.

The full-race workflow also runs weekly on the default branch. Its failure is a
maintainer-owned regression: reproduce with `./scripts/verify exhaustive`,
land a repair through normal review, rerun `full-race.yml` on the candidate
branch, and link the successful run before closing the incident. The scheduled
gate is not best-effort telemetry.

## Rust operator runbook

This runbook is active only after SDK work 6 lands the guarded publication
workflow and its release-guard tests on `main`. Until then, every Rust Release
Please PR remains a stop condition.

1. Confirm the proposal contains exactly one Rust component. Inspect its path,
   version, changelog, manifest entry, Cargo manifest, workspace lock, exact CLI
   SDK pin when applicable, tag, and downstream workflow effects.
2. Invoke `$release-please-release` to classify the component diff, review the
   proposal, and present the exact version for explicit confirmation. Do not
   treat approval of one component as approval of another.
3. Require the automatically dispatched `Verify / verify` and
   `Full race verification / full-race` jobs for the exact PR head SHA. Resolve
   every review conversation before merge. If either job is missing, fix the
   dispatcher instead of making manual dispatch part of the routine process.
4. After exact-version confirmation, merge that component Release Please PR.
   Do not run `cargo publish`, create a tag, edit the manifest, or publish the
   draft manually.
5. Watch the `Release Please` workflow through candidate verification,
   publication, registry reconciliation, consumer/docs.rs verification, and
   finalization. The first release of each crate additionally waits for the
   one-time protected-environment approval; steady-state OIDC releases do not.
6. On failure, inspect the failed stage before rerunning. Resume only the same
   component version, tag, and SHA. The workflow must query and compare the
   registry result before it decides whether publishing is still necessary.
   If finalization already succeeded, reconcile that matching published release
   as complete. A workflow repair may run from a newer `main`, but it must still
   publish or verify the original candidate SHA. Never create a replacement tag
   or republish an existing version. Verify the expected beta/stable flag and
   confirm that a Rust release did not become repository-global Latest.

The workflow outcome, accepted registry checksum, consumer verification, docs.rs
result, and final GitHub release URL are the release evidence. A draft left
unpublished is a failed or interrupted release, not permission to bypass the
pipeline.

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

The setup configuration gives each product independent Release Please proposal
ownership. It keeps each crate manifest and workspace-lock entry aligned,
excludes the CLI path until work 9 authorizes it, and connects only the SDK to
the protected reusable crate workflow. The workflow is inert until this setup
lands on `main`, the protected environment is configured, and a separately
reviewed SDK Release Please PR is explicitly confirmed and merged.

The first SDK proposal must be a prerelease, not the non-prerelease `0.1.0`
that was present in closed PR #32. The SDK component uses
`versioning: prerelease`, `prerelease: true`, and `prerelease-type: beta`; the
setup sets the SDK package's temporary `release-as` to `0.1.0-beta.1` so
historical breaking commits do not imply `1.0.0-beta`. After the beta passes
public-consumer verification, the OIDC hardening change keeps prerelease
versioning, sets `prerelease: false`, and updates package-scoped `release-as` to
`0.1.0`. That change must also include a reviewed, release-eligible Conventional
Commit in the SDK path that records the verified beta and stable support
boundary; configuration-only changes do not make a component eligible for a
Release Please proposal. Do not use an empty version-bump commit. Remove
`release-as` and the temporary
prerelease fields only after stable succeeds. Both exact overrides remain
subject to release-skill review and confirmation. After that immutable SDK
artifact is verified,
CLI work 9 may reuse the guarded flow with distinct `opendart-cli` outputs,
environment, bootstrap credential, and tag identity.

Both flows package and dry-run without credentials, query crates.io before
publishing, recheck name, owner, and exact-version state immediately before
authority is used, publish at most once, and reconcile after every publish
attempt. The credential-free job performs the full locked dry-run. The
credentialed job then uses `cargo +1.97.1 publish --locked --no-verify` at the
same candidate SHA so Cargo does not rerun dependency or build scripts while the
crates.io credential is present; the accepted artifact comparison remains the
post-publication proof.
Cargo may time out while polling the registry even though upload succeeded, so
a failed command is not evidence that the version is absent. If a version is
present, download it and accept it only when checksum, provenance, normalized
manifest, and unpacked contents match the reviewed candidate. Then prove a
clean exact-version consumer or install, wait for docs.rs, and finalize only
the matching draft. The [Cargo publish reference](https://doc.rust-lang.org/cargo/commands/cargo-publish.html)
defines the dry-run and post-upload polling behavior.

The prepared CLI artifact comparator is local-only and accepts an already
downloaded candidate, accepted crate, registry checksum, reviewed inventory,
and expected VCS metadata. It grants no acquisition or publication authority.
The detailed SDK bootstrap, recovery, and stable-promotion gates are in the
[SDK verification guide](docs/rust-sdk/verification-and-release.md). The
dependent CLI stop gate and interrupted-run procedure are in the
[CLI verification guide](docs/rust-cli/verification-and-release.md).
