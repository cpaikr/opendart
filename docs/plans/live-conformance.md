# Credentialed Live Conformance

## Objective

Exercise every physical OpenDART operation each week and verify that its live
response remains usable under the committed OpenAPI contract and reviewed
endpoint-specific expectations. This is observational test infrastructure; it
never updates the specification automatically.

Live conformance has a credential, evidence, and failure issue distinct from
credential-free [guide drift](guide-drift.md).

## Current state

- `scripts/probe-multi-company.mjs` is a targeted Node.js probe for the existing
  multi-company serialization decision, with focused offline tests.
- No Go live runner, complete case inventory, credentialed workflow, or live
  issue automation is committed.
- The Go compatibility gate covers OpenAPI loading, response routing and
  validation, and representative JSON, XML, and ZIP fixtures. Operation
  enumeration, request validation, and the shared HTTP safety and reporting
  foundations remain future work in the
  [Go tooling migration](go-tooling-migration.md). The existing probe is the
  only implemented credentialed command until the general runner replaces it.
- The approved [Go-only cleanup](go-only-tooling-cleanup.md) ports that focused
  probe to Go and removes the Node toolchain without claiming completion of the
  general runner or introducing scheduled credentialed automation.

## Constraints

- Derive the matrix from every physical operation in the canonical OpenAPI
  document. A physical operation is one path, method, and representation tuple;
  JSON, XML, and binary variants are therefore separate operations, each with
  exactly one reviewed primary case.
- Use OpenAPI for enumeration, routing, serialization, and structural
  validation. Keep public inputs, narrowly scoped discovery dependencies, and
  stable semantic assertions in typed Go cases rather than the released
  specification.
- Prefer committed stable inputs. Discovery is allowed only when a durable input
  is impractical; declare and budget each discovery request, reuse its result,
  and report a discovery failure separately from an endpoint assertion failure.
- Fail before network access when coverage, parameters, server trust, or the
  total request budget is invalid. Make one attempt per request and do not hide
  retries.
- Require meaningful success: bound and parse JSON, XML, and ZIP data; validate
  the OpenDART result envelope; and run named endpoint-specific assertions. A
  transport status or media type alone never passes a case.
- Run only trusted default-branch code. Keep `OPENDART_API_KEY` in a protected
  environment and expose it only to the request boundary, never to pull
  requests, non-default refs, arguments, logs, artifacts, reports, or the
  isolated issue-writing job.
- Discard raw response bodies after validation. Persist only allowlisted
  identities, statuses, sizes, hashes, schema locations, assertion IDs, and safe
  comparison evidence.
- Use the shared report, fixed-failure, deduplication, and recovery contract in
  the [tooling migration](go-tooling-migration.md), with an independent live
  failure issue.

## Ordered work

1. **Partially complete.** The Go compatibility gate covers OpenAPI loading,
   response routing and validation, and representative JSON, XML, and ZIP
   fixtures. Operation enumeration and request validation remain future work.
2. Port the existing focused multi-company probe to Go through the
   [Go-only cleanup](go-only-tooling-cleanup.md), preserving its current cases,
   assertions, credential isolation, and sanitized report. This remains a
   focused probe rather than the general runner.
3. Implement operation enumeration, request validation, the runner, request
   budget, representation adapters, typed assertions, sanitized report, and
   offline HTTP tests.
4. Curate one primary case for every physical operation, adding typed discovery
   only where stable public inputs are impractical. Add a preflight coverage and
   total-budget gate.
5. Add a manual protected workflow and isolated notifier. Configure the
   environment and credential only after offline coverage, budgets, and report
   sanitization have been reviewed.
6. Run the complete matrix under supervision, inspect logs and artifacts for
   leakage, and enable the weekly schedule only after the run is accepted.

## Acceptance criteria

- Every path, method, and representation tuple has exactly one reviewed primary
  case in every scheduled matrix.
- Each passing case proves representation and meaningful content, not only
  transport success.
- Discovery remains explicit, budgeted, reusable, and distinguishable from
  endpoint failure.
- The runner cannot exceed its derived request ceiling or retry implicitly.
- Pull requests and non-default refs cannot access the API key or issue-writing
  authority.
- Reports and issues contain no credential, authenticated URL, or unrestricted
  response body; workflow failures use only the fixed trusted fallback.
- Repeated failures update one live issue; recovery is recorded once and the
  issue is never closed automatically.
- Live observations never alter specification metadata or generated files.

## Next action

First complete the focused probe port and Node removal in the
[Go-only cleanup](go-only-tooling-cleanup.md). When general live-conformance
work is explicitly started afterward, implement operation enumeration, request
validation, and the shared HTTP and reporting seams before curating the complete
case inventory. Do not configure a credentialed workflow yet.
