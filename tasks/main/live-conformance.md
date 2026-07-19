# Credentialed Live Conformance

## Outcome

Exercise every physical OpenDART operation each week and verify that its live
response remains usable under the committed OpenAPI contract and reviewed
endpoint-specific expectations. This is observational test infrastructure; it
never updates the specification automatically.

Live conformance has a credential, evidence, and failure issue distinct from
credential-free [guide drift](guide-drift.md).

## Current state

Implementation progress is maintained in the task-local
[live-conformance progress](live-conformance-progress.md) document.

- The focused multi-company probe is ported to the Go CLI with its existing
  request matrix, assertions, sanitized report, expanded offline HTTP coverage,
  and direct operational entry point.
- The general Go runner owns physical-operation enumeration, the complete typed
  primary-case registry, bounded reusable discovery, offline request
  validation, fail-closed preflight, bounded execution, semantic assertions,
  and the sanitized report contract.
- Repository verification runs its coverage, request-budget, and sanitization
  preflight without reading a credential or contacting OpenDART.
- A manual-only producer is constrained to trusted `main` code, declares the
  protected environment, and uploads only the sanitized report. An isolated
  default-branch notifier validates that report or uses a fixed workflow
  failure, deduplicates the independent live issue, records recovery once, and
  never closes it.
- The GitHub environment and credential remain unconfigured. No workflow has
  been dispatched, no live request has been made, and no schedule is enabled.
- The completed [Go-only cleanup](../../plans/main/go-only-tooling-cleanup.md)
  does not
  claim completion of the general runner or introduce scheduled credentialed
  automation.

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
  the [tooling migration](../../plans/main/go-tooling-migration.md), with an
  independent live failure issue.

## Ordered work

1. **Complete.** Lasting Go tests cover OpenAPI loading, semantic comparison,
   response routing and validation, representative JSON, XML, and ZIP fixtures,
   physical-operation enumeration, and offline request validation.
2. **Complete.** Ported the existing focused multi-company probe to Go through
   the [Go-only cleanup](../../plans/main/go-only-tooling-cleanup.md), preserving
   its cases,
   assertions, credential isolation, and sanitized report. This remains a
   focused probe rather than the general runner.
3. **Complete.** Implemented operation enumeration, request validation, the
   runner, request budget, representation adapters, typed assertions, sanitized
   report, and offline HTTP tests.
4. **Complete.** Curated one primary case for every physical operation, added
   bounded reusable disclosure discovery for rare event coordinates, and wired
   coverage, pagination closure, total-budget, and sanitization preflight into
   repository verification.
5. **Complete.** Added the manual protected workflow and isolated notifier with
   offline workflow-policy, report-validation, budget, sanitization,
   deduplication, and recovery gates. Environment and credential configuration
   remains deliberately deferred.
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

After explicit authorization, configure the protected GitHub environment and
credential, then perform ordered work 6 under supervision. Do not make a real
OpenDART request or enable the weekly schedule before that review.
