# Credentialed Probe Automation Plan

## Purpose

Automate selected authenticated OpenDART observations without exposing the API
key or weakening the repository trust boundary. This is post-extraction work
and must not be combined with public-guide maintenance automation.

## Current state

- `scripts/probe-multi-company.mjs` implements the existing sequential
  ten-request multi-company probe.
- Offline tests cover URL serialization, JSON/XML identity extraction,
  malformed XML, unexpected identities, and non-distinct baselines.
- A missing `OPENDART_API_KEY` fails before requests begin.
- The live matrix has not run and no GitHub secret or workflow is configured.

## Security invariants

- Execute trusted default-branch code only. Do not use `pull_request`,
  `pull_request_target`, fork code, or a caller-selected ref.
- Store `OPENDART_API_KEY` in a protected GitHub Environment, not as an
  unguarded repository secret. Its deployment branch policy must allow only the
  default branch. Require a reviewer for the supervised/manual stage when the
  repository's GitHub plan supports it; otherwise agree on an equivalent
  external approval gate before configuring the key.
- Read `OPENDART_API_KEY` only in the probe step. Never place it in arguments,
  authenticated URLs, logs, issue bodies, caches, or artifacts.
- Use minimal permissions, no persisted checkout credential, pinned action
  commit SHAs, workflow concurrency, and bounded job/request timeouts.
- Permit exactly ten sequential API requests with one attempt each. Abort before
  exceeding the request ceiling; do not hide retries in a shared HTTP helper.
- Serialize artifacts through an explicit sanitized allowlist. Preserve status,
  response identity, and comparison evidence without request URLs or secrets.
- Keep the secret-bearing probe job at `contents: read`. If issue notification
  is later enabled, use a separate `issues: write` job that has no environment
  secret and receives only an allowlisted sanitized status.

## Implementation slices

1. Refactor the probe runner to expose an explicit request budget and timeout,
   then test missing-key, timeout, ceiling, sanitization, and mocked response
   behavior.
2. Add a manual-only workflow plus the protected environment and review its
   event, deployment-branch rule, ref, permissions, checkout, reviewer gate,
   log, and artifact boundaries before configuring the secret.
3. Configure `OPENDART_API_KEY`, run one supervised probe, inspect the complete
   log and downloaded artifact for leakage, and record only sanitized evidence.
4. Enable Monday 09:00 Asia/Seoul scheduling only after the supervised run is
   accepted. Keep failure notification deduplicated and separate from guide
   drift issues.
5. Consider representative JSON/XML/ZIP success/no-data fixtures and a rotating
   endpoint matrix only after quota and throttling behavior are known.

## Acceptance criteria

- Tests prove the key cannot appear in emitted JSON, errors, or request
  diagnostics.
- Pull requests and arbitrary refs cannot access the secret-bearing job.
- The job's protected environment, rather than editable workflow YAML alone,
  enforces the default-branch and approval boundary.
- The workflow makes no more than ten requests and performs no retry.
- Output is sanitized before artifact upload; raw responses are not retained.
- A failed or repeated scheduled run cannot create duplicate issues.
- Specification metadata changes remain a separate reviewed commit.

## Open decisions

- Confirm who may configure and rotate the repository secret.
- Confirm protected-environment and required-reviewer availability for the
  chosen repository visibility; select an equivalent external gate if needed.
- Define artifact retention and whether sanitized observations may be public.
- Decide which notification label/title uniquely identifies probe failures.
- Establish quota behavior before expanding beyond the ten-request matrix.

## Next action

Review the security boundary and offline test matrix. Do not add a workflow or
configure `OPENDART_API_KEY` until that review is explicitly accepted.

## Implementation references

- [GitHub environments and deployment branch rules](https://docs.github.com/en/actions/how-tos/managing-workflow-runs-and-deployments/managing-deployments/managing-environments-for-deployment)
- [GitHub Actions job-level token permissions](https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax#jobsjob_idpermissions)
