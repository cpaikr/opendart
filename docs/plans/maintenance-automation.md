# Repository Maintenance Automation Plan

## Purpose

Add operational automation after the repository extraction is accepted. This
is new repository functionality, not part of the history-preserving extraction
commit `885725f`, and should be implemented and reviewed separately.

## Current state

- The extracted repository has local package commands for tests, catalog
  validation, strict lint, and bundle freshness.
- No GitHub Actions workflow or guide-drift command is committed.
- The committed specification records source date `2026-07-17`, 85 logical
  endpoints, and 167 physical paths.
- The requirements below are planned only; no automation implementation is
  retained in the working tree.

## Boundaries

In scope:

- Credential-free pull-request and push verification.
- Semantic drift detection against the public OpenDART guide.
- A weekly and manually dispatched drift workflow.
- Diagnostic artifacts and one deduplicated tracking issue.

Out of scope:

- Authenticated OpenDART API probes; see
  [credentialed-probes.md](credentialed-probes.md).
- Automatic source refreshes, commits, pull requests, merges, or releases.
- Changes to endpoint coverage or the accepted specification contract.
- `dartdb` application work.

## Required behavior

### Offline verification

- Run `npm ci --ignore-scripts` and `npm run verify:opendart` on pull requests
  and pushes without contacting OpenDART.
- Use read-only repository permissions, pinned action commit SHAs, no persisted
  checkout credential, a fixed supported Node version, and a bounded timeout.
- Disable dependency telemetry and update checks during verification.

### Guide-drift command

- Regenerate into a temporary tree using the committed `info.version` and
  `x-opendart.source.checkedAt`; unchanged checks must not advance source dates.
- Compare canonicalized generated YAML so formatting and excluded page chrome
  do not count as drift. Preserve array order and exclude the derived bundle.
- Report `unchanged`, `changed`, or `error` as concise machine-readable JSON.
  For changes, include affected operation identities and file hashes.
- Write normalized group and endpoint inventory before enforcing expected
  totals. Preserve baseline/candidate evidence for changed files and structured
  source context on failures.
- Unit-test unchanged, semantic mutation, invalid date/YAML, inventory addition,
  and extraction failure paths without live requests.

### Scheduled workflow

- Run Monday at 09:00 Asia/Seoul (`0 0 * * 1` UTC) and support
  `workflow_dispatch`. Require the default branch for manual dispatch and
  explicitly check it out instead of a caller-selected ref.
- A full check may make exactly 91 guide requests: six group pages and 85 detail
  pages. Allow one attempt per request, enforce per-request and job timeouts,
  and fail before exceeding the ceiling.
- Use workflow concurrency to prevent overlapping checks.
- Upload the JSON result, normalized inventory, and changed-file evidence even
  when drift extraction fails. Choose and document a bounded retention period.
- Keep guide extraction in a `contents: read` job. Pass only the sanitized
  result to a separate notification job with job-level `issues: write`; that
  job must not check out or execute repository code.
- Protect the notification job with an environment whose deployment branch
  policy permits only the default branch, so a modified manual-dispatch
  workflow cannot gain issue-writing authority from another ref.
- On `changed` or `error`, find an open issue by a stable hidden marker and
  update it, or create it if absent. An `unchanged` run creates no issue. Never
  auto-close a drift issue or modify specification files.

## Implementation slices

1. Add the offline verifier workflow and tests/assertions for workflow security
   settings.
2. Add the guide-drift command, diagnostic seam in the generator, and offline
   fixtures.
3. Add the scheduled workflow, request-budget enforcement, artifact upload,
   and deduplicated issue handling.
4. Run one manual live check, review its artifacts and permissions, then enable
   the schedule.

Each slice should remain independently reviewable and must pass
`npm run verify:opendart` plus a focused code review before commit.

## Acceptance criteria

- Offline CI performs no OpenDART requests and needs no secret.
- Formatting-only input produces `unchanged`; one semantic mutation identifies
  only its affected operation.
- Live execution cannot exceed 91 requests or retry implicitly.
- Every non-success path leaves a machine-readable result and useful artifact.
- Repeated failures update one issue instead of creating duplicates.
- A non-default ref cannot obtain issue-writing authority.
- No workflow can commit, merge, release, or expose credentials.

## Open decisions

- Confirm the permanent GitHub remote before enabling Actions.
- Choose artifact retention and the stable drift-issue label/title.
- Confirm protected-environment availability for the chosen repository
  visibility and configure its default-branch deployment policy.
- Revalidate action versions and pin their current commit SHAs during
  implementation.

## Next action

After the extraction repository is published, implement slice 1 only. Do not
combine it with guide-drift scheduling or credentialed probes.
