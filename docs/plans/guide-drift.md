# Public Guide Semantic Drift Plan

## Purpose

Detect whether the public OpenDART development guide would change the generated
OpenAPI contract. The job reports evidence only: it never edits the
specification or creates a pull request.

This credential-free workflow is separate from authenticated live endpoint
conformance in [`live-conformance.md`](live-conformance.md).

## Current state

- `.github/workflows/verify.yml` already provides credential-free pull-request
  and manual verification of the committed repository.
- Source refresh, validation, and bundling currently use Node.js; the accepted
  [Go migration](go-tooling-migration.md) will replace them before drift
  automation is built.
- No guide-drift command, scheduled workflow, or deduplicated issue notification
  is committed.
- The permanent repository is `cpaikr/opendart`.

## Definition of drift

A run is `changed` only when current guide content normalizes into a different
generated OpenAPI contract. The comparison excludes:

- formatting and key-serialization differences;
- page chrome and source markup not represented by the normalized catalog;
- the check timestamp and snapshot version supplied by the run; and
- the derived portable bundle as an independent input.

A guide edit that produces the same specification is `unchanged`. Acquisition,
normalization, generation, or validation failures are `error`, not drift.

The Go command regenerates into a temporary tree using the committed snapshot
metadata, loads baseline and candidate through the same OpenAPI 3.2 boundary,
and performs an OpenAPI-aware semantic comparison through the gated
`libopenapi` adapter. It reports every semantic change, not only changes
classified as breaking. A changed report identifies affected guide identities,
physical operations, and contract locations.

## Acquisition and request budget

- Fetch each configured guide group page once to discover the current endpoint
  inventory, including additions and removals.
- Validate every discovered identity and URL against trusted OpenDART origins
  and the expected guide URL grammar before fetching detail pages.
- Fetch each unique detail page once, with one attempt and a per-request
  timeout. Shared HTTP code must not add hidden retries.
- Determine the normal request budget from the validated discovered inventory
  and enforce a separately configured absolute safety ceiling before detail
  acquisition begins.
- Reject duplicates, untrusted URLs, an excessive inventory, or budget
  exhaustion with a structured `error` report.

The configured groups and safety ceiling live in tested Go code rather than in
this document, so ordinary catalog growth does not make the plan stale.

## Report contract

Every run emits a small versioned JSON report with:

- `unchanged`, `changed`, or `error` outcome;
- repository commit, baseline snapshot identity, and observation time;
- normalized inventory hashes and baseline/candidate specification hashes;
- affected group/API/operation identities and semantic change summaries; and
- safe phase and error classifications when processing fails.

Changed-file evidence may contain normalized diffs and source locations. It
must not contain unrestricted source pages, credentials, or authenticated URLs.
Artifacts use a bounded retention period; the initial workflow should use 14
days unless repository policy requires less.

Logs remain on stderr and are not treated as issue input.

## GitHub Actions design

- Run Monday at 09:00 Asia/Seoul (`0 0 * * 1` UTC) and support manual dispatch.
- Always execute the trusted default branch. A manual dispatch from another ref
  must not execute caller-selected repository code.
- Use pinned action commit SHAs, no persisted checkout credentials, explicit
  job and request timeouts, and workflow concurrency.
- Keep acquisition and comparison in a `contents: read` job.
- Pass only the schema-validated sanitized report to a separate notification
  job with `issues: write` when the report is valid. That job has no API key and
  cannot consume raw source artifacts.
- Protect issue writing with a default-branch-only GitHub Environment.
- Run the notification job after the producer with `if: always()`. Use pinned
  `actions/github-script` to independently validate the report under a strict
  byte ceiling and check its consistency with the producer conclusion. Report
  retrieval is non-fatal. If the report is unavailable, oversized, invalid, or
  inconsistent, discard its contents and synthesize a fixed workflow-failure
  envelope from the producer conclusion, commit, and workflow run identity in
  trusted GitHub contexts. Never accept producer logs, step output, exception
  messages, or unrestricted workflow text.

On `changed`, `error`, or a synthesized workflow failure, find the open issue
containing `<!-- opendart-guide-drift:v1 -->` and update its automation-owned
body, or create one titled `OpenDART guide semantic drift`. The issue must
mention `@sjunepark` and link the workflow run with the sanitized evidence.

`unchanged` creates no issue. When an open drift issue exists and the check
transitions from a reported `changed`, `error`, or workflow failure to
`unchanged`, add exactly one recovery comment linking the successful run and
leave the issue body and state unchanged. Later unchanged runs remain silent
until another reported failure. Automation never closes the issue. This marker
and issue are distinct from live conformance failures.

## Implementation slices

- [x] Add credential-free repository verification with read-only permissions.
- [ ] Complete the Go OpenAPI compatibility gate and port the generator and
      verifier needed by drift detection.
- [ ] Implement the drift command, versioned report schema, semantic comparator,
      request budget, and offline fixtures.
- [ ] Add the scheduled read-only job, artifact upload, isolated deduplicated
      issue notification, fixed fallback envelope, and report-contract tests.
- [ ] Run one manual live check, inspect its artifacts and permissions, and then
      enable the schedule.

## Acceptance criteria

- Pull-request verification remains offline and credential-free.
- Formatting-only or irrelevant page changes report `unchanged`.
- A representative semantic mutation reports only its affected contract
  locations; additions and removals are visible.
- A run cannot exceed its validated request budget or retry implicitly.
- Every command-controlled non-success path leaves a machine-readable sanitized
  result. Setup or launch failures that prevent a report, artifact retrieval
  failures, oversized reports, invalid reports, and conclusion mismatches
  instead produce the fixed workflow-failure notification when the notifier can
  run.
- Repeated drift or processing failures update one guide-drift issue and mention
  `@sjunepark`.
- The first unchanged run after a reported failure adds one recovery comment;
  later unchanged runs are silent and no issue is automatically closed.
- A non-default ref cannot obtain issue-writing authority.
- No guide-drift workflow or job can commit, open a pull request, merge,
  release, or modify the specification.

## Next action

Complete the compatibility gate in
[`go-tooling-migration.md`](go-tooling-migration.md). The drift command is the
first automation consumer after generation and verification have moved to Go.
