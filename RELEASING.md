# Release policy

The released product is `openapi/generated/openapi.bundle.yaml`. Humans classify
changes to that public contract; Release Please turns the chosen Conventional
Commit inputs into a version, changelog, tag, and draft GitHub Release. The
release workflow runs the credential-free Go repository verifier, uploads the
bundle and checksum, and publishes the immutable release.

The release sources of truth are:

- [`release-please-config.json`](release-please-config.json) for versioning and
  path exclusions;
- [`.release-please-manifest.json`](.release-please-manifest.json) for the current
  release version;
- [`CHANGELOG.md`](CHANGELOG.md) for generated release notes;
- [`.github/workflows/verify.yml`](.github/workflows/verify.yml) for repository
  verification; and
- [`.github/workflows/release-please.yml`](.github/workflows/release-please.yml)
  for release creation, asset upload, recovery, and publication.

Do not manually edit the generated changelog or manifest, move published tags,
replace immutable assets, or publish a release outside this flow.

## Version meanings

- `openapi: 3.2.0` identifies the OpenAPI dialect.
- `info.version` and `x-opendart.source.checkedAt` identify the upstream guide
  snapshot date.
- The `vX.Y.Z` Git tag identifies this repository's bundle compatibility
  version. It does not version the upstream OpenDART service.

## Release eligibility

Compare the candidate bundle with the bundle at the latest release tag before
choosing a Conventional Commit type. A change is releasable only when the bundle
contains a material public-contract change and matches a fresh build.
Formatting-only changes and repository, tooling, CI, test, or prose changes are
not releasable. Generator changes become releasable in the commit that also
updates the generated specification.

Release Please excludes the repository-only paths listed in
[`release-please-config.json`](release-please-config.json). Root files are not
covered by those directory exclusions, so root-file-only changes must use a
non-releasable commit type. Reviewers must reject any release proposal whose
bundle has no material change, regardless of commit wording.

While the project is below `1.0.0`:

| Impact | Examples | Commit input | Version impact |
| --- | --- | --- | --- |
| None | Tooling, tests, CI, repository docs, formatting | `chore:`, `test:`, `ci:`, `docs:` | None |
| Compatible fix | Corrected descriptions or schemas without consumer incompatibility | `fix(openapi):` | Patch |
| Compatible addition | Additive endpoints, optional inputs, or output fields | `feat(openapi):` | Patch |
| Breaking | Removed or renamed operations or fields, new required inputs, schema narrowing, incompatible security or serialization changes | `feat(openapi)!:` with a `BREAKING CHANGE:` footer | Minor |

At and after `1.0.0`, use standard SemVer: compatible fixes are patches,
compatible additions are minors, and incompatible changes are majors. Classify
corrections by consumer impact; a change described as a fix can still be
breaking.

## Verification gate

The intended branch-protection policy for `main` is to require pull-request
review, conversation resolution, and the `verify` job produced by the
[`Verify` workflow](.github/workflows/verify.yml). Branch protection is external
GitHub state, not provisioned by this repository, so confirm the configured rule
before relying on it.

Release Please uses the repository `GITHUB_TOKEN`. GitHub therefore does not
trigger a new `pull_request` workflow run for a Release Please-authored event;
there is no approval-waiting Verify run. Validate an open Release Please pull
request by manually dispatching Verify on its head branch, for example:

```sh
gh workflow run verify.yml --ref <release-please-branch>
```

Inspect the resulting run in Actions and do not merge until it passes. A
different credential would be required if the repository later chooses to
trigger this CI automatically.

## Release flow

1. Record the latest release tag and summarize the material bundle diff in the
   implementation pull request.
2. Confirm that the intended merge commit follows the compatibility table and
   that the committed bundle matches a fresh build.
3. Merge the implementation only after review and Verify pass.
4. Review the Release Please pull request's proposed version, changelog, bundle
   scope, and any breaking-change note. Manually dispatch Verify on that branch
   and merge only after it passes.
5. After the Release Please pull request merges, the release workflow calls
   Verify before the release job creates or resumes the draft release.
6. Confirm that the published immutable GitHub Release targets the release
   commit and contains `openapi.bundle.yaml` and
   `openapi.bundle.yaml.sha256`.

If a run fails after its draft release is created, rerun the failed Release
Please workflow. If recovery requires a code change, merge the repair; the next
`main` push verifies that commit before resuming the draft. Never move a
published tag or replace an immutable asset.
