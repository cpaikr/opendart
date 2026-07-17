# Credentialed Live Conformance Plan

## Purpose

Exercise every physical OpenDART operation on a weekly schedule and verify that
its actual response remains usable according to the committed contract and
reviewed endpoint-specific expectations. This is an observational test system,
not application logic, and it never updates the specification automatically.

Live conformance is deliberately separate from public-guide semantic drift in
[`guide-drift.md`](guide-drift.md): it has a secret, a different failure issue,
and different evidence.

## Current state

- `scripts/probe-multi-company.mjs` is a targeted Node.js probe for the
  multi-company serialization decision. It is legacy input to the general Go
  runner, not the desired coverage model.
- Offline tests cover that probe's serialization, JSON/XML parsing, identity
  checks, malformed input, and sanitized output.
- No credentialed GitHub workflow, protected environment, complete case
  inventory, or repository secret is configured in the worktree.
- The [Go migration](go-tooling-migration.md) and full live runner are pending.

## Settled coverage decisions

- Derive the execution matrix from every physical operation in the canonical
  OpenAPI document. JSON, XML, and binary paths are independent test cases.
- Execute the complete matrix on every scheduled run. Do not partition or
  rotate endpoints unless measured quota or runtime evidence later requires a
  reviewed change.
- Keep public, non-secret request inputs in the repository. The API key is the
  only secret input.
- Require exactly one primary request per physical operation, one attempt per
  request, and no hidden retry. Derive the request ceiling from the validated
  matrix before network activity begins.
- Fail before network activity if an operation has no case, more than one case,
  an untrusted server, or an invalid parameter set.

## Contract and case data

The released OpenAPI document remains limited to guide-supported facts. Live
tests consume three layers:

1. OpenAPI identifies the operation, parameters, expected representations, and
   source-backed schema.
2. A test-only Overlay may add reviewed empirical constraints without changing
   the canonical files or release bundle.
3. Arazzo is preferred for committed inputs, operation relationships, and
   success criteria if the Go compatibility gate proves the required version
   and JSON/XML expression support. A small typed Go case manifest covers only
   behavior that cannot be expressed or executed faithfully.

Every empirical constraint records its source or observation date. Test inputs
must be public identifiers and dates that are safe to expose and likely to keep
producing meaningful data.

## Validation policy

A status code alone never passes a case. Each response goes through the
applicable layers:

- enforce timeout, size, decompression, and media-type boundaries;
- parse JSON or XML, or open ZIP responses with bounded entry and expanded-size
  limits;
- validate the request and response against the OpenAPI operation wherever the
  selected validator supports the representation faithfully;
- validate the OpenDART status/message envelope and require the case's expected
  successful business outcome;
- assert endpoint-specific stable content, such as a non-empty identifier or
  name, requested company/report identity, a non-empty result collection, or
  expected archive entry shape; and
- compare paired JSON/XML operations on selected semantic fields when they
  represent the same logical endpoint.

Strings used as evidence are non-empty after trimming. Collections are
non-empty by default. An endpoint may allow a documented no-data outcome only
through an explicit reviewed case rule; even then, parsing, envelope, schema,
and endpoint-specific no-data assertions still run. ZIP success requires a
valid non-empty archive, not merely a ZIP content type or signature.

Assertions are named and reported by ID so failures remain useful without
including full response bodies.

## Security and evidence boundary

- Execute trusted default-branch code only. Do not expose the secret to pull
  requests, `pull_request_target`, forks, or caller-selected refs.
- Store `OPENDART_API_KEY` in a protected GitHub Environment whose deployment
  branch policy permits only the default branch. Use a required reviewer for
  the first manual run when the repository plan supports it.
- Read the key only in the request step and use it only to construct the
  in-memory outgoing query. Never place it in arguments, logs, caches,
  artifacts, issue bodies, or persisted/reported authenticated URLs.
- Use minimal permissions, no persisted checkout credential, pinned action
  SHAs, concurrency control, and bounded job/request timeouts.
- Serialize results through an explicit allowlist. Retain operation identity,
  representation, safe HTTP/API status, response size/hash, assertion IDs, and
  selected redacted comparison evidence. Discard raw bodies after validation.
- Keep the secret-bearing job at `contents: read`. Give `issues: write` only to
  a separate job that has no protected environment secret and receives a
  schema-validated sanitized report.
- Protect that notification job with a separate default-branch-only Environment
  containing no API key, so a caller-selected ref cannot obtain issue-writing
  authority.

## Schedule and issue behavior

Add a manual default-branch workflow first. After a supervised full run passes
leakage and quota review, schedule it for Monday at 09:00 Asia/Seoul. Keep low
concurrency and sequential requests until observed service limits justify a
different policy.

On any case, setup, or runner failure, find the open issue containing
`<!-- opendart-live-conformance:v1 -->` and update it, or create one titled
`OpenDART live conformance failure`. The issue must mention `@sjunepark`, link
the workflow run, summarize affected operation IDs and failed assertions, and
contain only sanitized evidence. Successful runs create no issue, and
automation never auto-closes an existing issue.

## Implementation slices

1. Complete the Go compatibility gate for operation enumeration, request and
   response validation, Overlay/Arazzo needs, and JSON/XML/ZIP fixtures.
2. Implement the runner, request budget, representation adapters, semantic
   assertions, sanitized report schema, and offline tests.
3. Curate committed inputs and assertions for every physical operation. Add a
   coverage gate that fails before requests when the matrix and OpenAPI diverge.
4. Add the manual protected workflow, configure the environment and secret, run
   the complete matrix once, and inspect all logs and artifacts for leakage.
5. Add the isolated deduplicated issue job and enable the weekly schedule only
   after the supervised run is accepted.

## Acceptance criteria

- Every physical operation has exactly one reviewed primary case and runs in
  every scheduled matrix.
- Every passing case proves representation and meaningful content, not only
  transport status.
- JSON, XML, and ZIP validation failures identify the operation and assertion
  without retaining unrestricted bodies.
- Pull requests and non-default refs cannot access the API key or issue-writing
  authority.
- The runner cannot exceed its derived request ceiling or retry implicitly.
- Repeated failures update one live-conformance issue, distinct from guide
  drift, and mention `@sjunepark`.
- Live observations never alter specification metadata or generated files.

## Next action

Complete the Go compatibility gate, then generate the physical-operation case
inventory and identify the endpoint-specific inputs and assertions that require
manual curation. Do not configure `OPENDART_API_KEY` until offline coverage,
budgets, sanitization, and the protected workflow have been reviewed.
