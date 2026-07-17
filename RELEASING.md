# Release policy

The released product is `openapi/generated/openapi.bundle.yaml`. Release
Please manages the release manifest, changelog, tags, and GitHub Releases, but
humans classify compatibility from the bundle's public contract.

## Version meanings

- `openapi: 3.2.0` identifies the OpenAPI dialect.
- `info.version` and `x-opendart.source.checkedAt` identify the upstream guide
  snapshot date.
- The `vX.Y.Z` Git tag identifies this repository's bundle compatibility
  version. It does not version or make compatibility promises for the upstream
  OpenDART service.

## Release eligibility

Before choosing a Conventional Commit type, compare the candidate bundle with
the bundle at the latest release tag. Formatting-only changes and repository,
tooling, CI, test, or prose changes that leave the bundle's meaning unchanged
are not releasable.

A change is releasable only when `openapi/` contains a material public change
and the committed bundle matches a fresh build. Generator changes become
releasable in the commit that also updates the generated specification.
Release Please excludes repository-only directories, but its path exclusions
do not match individual files at the repository root. Root-file-only changes
must therefore use a non-releasable commit type, and reviewers must reject any
release proposal whose bundle has no material change.

While the project is below `1.0.0`:

| Impact | Examples | Commit input | Version impact |
| --- | --- | --- | --- |
| None | Tooling, tests, CI, repository docs, formatting | `chore:`, `test:`, `ci:`, `docs:` | None |
| Compatible fix | Corrected descriptions or schemas without consumer incompatibility | `fix(openapi):` | Patch |
| Compatible addition | Additive endpoints, optional inputs, or output fields | `feat(openapi):` | Patch |
| Breaking | Removed or renamed operations or fields, new required inputs, schema narrowing, incompatible security or serialization changes | `feat(openapi)!:` with a `BREAKING CHANGE:` footer | Minor |

At and after `1.0.0`, use standard SemVer: compatible fixes are patches,
compatible additions are minors, and incompatible changes are majors.
Corrections are classified by consumer impact; a change described as a fix can
still be breaking.

## Manual review gate

Branch protection for `main` must require the status check produced by the
**Verify** workflow in addition to review and conversation resolution. Select
the check shown by the first remote run after this workflow exists on the
default branch.

1. Record the latest release tag and summarize the material bundle diff in the
   implementation pull request.
2. Confirm the commit message matches the compatibility classification above.
3. Merge the implementation only after the bundle freshness and validation
   checks pass.
4. Wait for Release Please to create or update its release pull request. Do not
   edit the generated manifest or changelog manually.
5. Release Please uses the repository `GITHUB_TOKEN`, so its pull request's
   **Verify** run waits for manual approval. Inspect the generated changes and
   select **Approve workflows to run**. If GitHub did not queue a run, dispatch
   Verify manually on the Release Please branch.
6. Review the proposed version, changelog, bundle scope, and breaking-change
   note. Record the exact version in the approval and merge only after Verify
   passes.
7. After merge, confirm the immutable GitHub Release contains
   `openapi.bundle.yaml` and `openapi.bundle.yaml.sha256` and targets the release
   commit.

If a release fails after its draft is created, rerun the failed Release Please
workflow. If recovery requires a code change, merge the repair; the resulting
`main` push will run Verify before resuming the draft. Never move a published
tag or replace an immutable asset.
