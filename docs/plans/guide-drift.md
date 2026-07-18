# Public Guide Semantic Drift

## Objective

Detect whether the current public OpenDART development guide would change the
generated OpenAPI contract. The weekly job reports evidence only; it never
edits the specification, creates a pull request, or publishes a release.

This credential-free work is independent of authenticated
[live conformance](live-conformance.md).

## Current state

- `.github/workflows/verify.yml` provides read-only pull-request and manual
  verification of the committed repository.
- The internal Go CLI owns guide acquisition, normalization, generation,
  staged validation, and semantic compatibility checks. Credential-free
  offline verification also runs through the Go CLI.
- No drift command, scheduled workflow, or drift issue automation is committed.

## Constraints

- Report `changed` only when normalized guide content changes OpenAPI meaning.
  Formatting, page chrome, run timestamps, snapshot-version substitution, and
  the derived bundle do not independently constitute drift.
- Treat acquisition, normalization, generation, validation, and comparison
  failures as errors, not changes.
- Discover the current inventory from trusted guide group pages, validate every
  identity and URL before detail requests, fetch each unique page once, and
  enforce both an inventory-derived request budget and an absolute ceiling. Do
  not retry implicitly.
- Execute trusted default-branch code with read-only contents permission.
  Issue-writing authority belongs to an isolated default-branch notification
  job with no OpenDART credential or access to unrestricted source artifacts.
- Use the shared report, fixed-failure, deduplication, and recovery contract in
  the [tooling migration](go-tooling-migration.md). Drift uses its own report
  outcome and tracking issue, distinct from live failures.

## Ordered work

1. **Complete.** The Go generation, validation, and semantic-comparison gate
   can compare a temporary candidate with the committed baseline.
2. Implement the drift command, request budget, versioned report, and offline
   fixtures for unchanged content, semantic additions and removals, malformed
   sources, and processing failures.
3. Add manual default-branch automation, bounded artifacts, and the isolated
   drift notifier. Test missing, oversized, invalid, and conclusion-inconsistent
   reports without trusting producer-controlled text.
4. Run one supervised check, inspect its permissions and artifacts, then enable
   the weekly schedule.

## Acceptance criteria

- Formatting-only and irrelevant guide changes report `unchanged`.
- Representative semantic changes identify the affected guide identities,
  physical operations, and contract locations.
- Every command-controlled outcome produces a sanitized machine-readable
  report, and workflow failures use only the fixed trusted fallback.
- A run cannot exceed its validated request budget or retry implicitly.
- Repeated findings update one drift issue; recovery is recorded once and the
  issue is never closed automatically.
- Non-default refs cannot obtain issue-writing authority.
- Drift automation cannot modify the specification or initiate delivery work.

## Next action

When drift work is explicitly started, implement the credential-free drift
command and its offline fixtures before adding GitHub write permissions or
scheduling. No drift implementation is part of the completed credential-free
[Go tooling migration](go-tooling-migration.md) scope.
