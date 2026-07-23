# Public Guide Semantic Drift

## Outcome

Detect whether the current public OpenDART development guide would change the
generated OpenAPI contract. The planned weekly job will report evidence only;
it will never edit the specification, create a pull request, or publish a
release.

This credential-free work is independent of authenticated
[live conformance](live-conformance.md).

## Current state

- `.github/workflows/verify.yml` provides read-only pull-request and manual
  verification of the committed repository.
- The internal Go CLI owns guide acquisition, normalization, generation,
  staged validation, and semantic compatibility checks. Credential-free
  offline verification also runs through the Go CLI.
- Drift-safe acquisition is implemented. It discovers only the validated
  canonical inventory table, ignores page chrome, permits current-inventory
  additions and removals, validates and deduplicates identities before detail
  requests, makes one transport attempt per page, and enforces the derived and
  absolute request ceilings. Regular synchronization retains strict accepted
  inventory checks and its retrying connection pool.
- Required guide tables reject duplicates and malformed message-code labels
  fail closed.
- The offline drift command generates and structurally validates a temporary
  candidate, normalizes only snapshot metadata, and emits a bounded versioned
  semantic report.
- Manual-only default-branch automation uploads exactly that bounded report.
  A separate notifier validates the report, substitutes fixed trusted failure
  state when necessary, and updates one marker-owned drift issue. Workflow
  policy is enforced offline. No workflow has been dispatched, no issue has
  been written, and no schedule is enabled.

## Implementation checklist

- [x] Add drift-safe acquisition with canonical inventory-table validation,
  dynamic inventory cardinality, one attempt per page, and enforced request
  budgets.
- [x] Add the drift command, narrow snapshot normalization, bounded versioned
  report, and offline fixtures for unchanged content, additions, removals,
  multi-company changes, truncation, unsafe evidence, and processing errors.
- [x] Add isolated manual automation, notifier validation, bounded artifacts,
  and credential-free workflow-policy verification.

## Constraints

- Report `changed` only when normalized guide content changes OpenAPI meaning.
  Formatting, page chrome, run timestamps, snapshot-version substitution, and
  the derived bundle do not independently constitute drift.
- Treat acquisition, normalization, generation, validation, and comparison
  failures as errors, not changes.
- Let valid inventory additions and removals reach semantic comparison. The
  drift path must not apply committed group counts, full-inventory totals, or
  equivalent baseline cardinality checks in a way that turns those changes
  into acquisition or candidate-validation errors.
- Discover the current inventory from exactly one canonical table on each
  trusted guide group page. Validate the table headers, ignore links outside
  that table, validate every identity and URL before detail requests, fetch
  each unique page once, and enforce both an inventory-derived request budget
  and an absolute ceiling. Missing, duplicate, or malformed inventory tables
  are processing errors. Do not retry implicitly.
- Before comparison, substitute the committed baseline snapshot value into the
  candidate `info.version` and every root or operation
  `x-opendart.source.checkedAt` field. Keep this normalization narrow so other
  `x-opendart` changes remain meaningful.
- Compare the committed baseline as the original document and the generated
  candidate as the updated document so additions, removals, detail values, and
  breaking-change counts have the expected direction.
- Execute trusted default-branch code with read-only contents permission.
  Issue-writing authority belongs to an isolated default-branch notification
  job with no OpenDART credential or access to unrestricted source artifacts.
- Use the shared report, fixed-failure, deduplication, and recovery contract in
  the [tooling migration](../../plans/main/go-tooling-migration.md). Drift uses
  its own report outcome and tracking issue, distinct from live failures.
- Emit `unchanged`, `changed`, or `error` through a versioned, bounded,
  allowlisted report. Count every semantic change while retaining only a
  deterministic bounded set of sanitized findings; when findings are omitted,
  keep the outcome `changed`, report exact aggregate counts, and set
  `truncated` to `true`. Raw values and the full semantic difference must not
  enter the report, notifier, or issue. The initial automation does not emit a
  separate exhaustive diagnostic artifact; use a supervised local rerun when
  more detail is needed.
- Update the same drift issue for `changed` and `error` outcomes. An error never
  records recovery; only a validated `unchanged` outcome records recovery once.

## Ordered work

1. **Complete.** The Go generation, validation, and semantic-comparison gate
   can compare a temporary candidate with the committed baseline.
2. **Complete.** Implement the drift command, request budget, versioned report,
   and offline fixtures for unchanged content, semantic additions and removals,
   multi-company changes, truncation, unsafe evidence, malformed sources, and
   processing failures.
3. **Complete.** Add manual default-branch automation, bounded artifacts, and
   the isolated drift notifier. Test missing, oversized, invalid, and
   conclusion-inconsistent reports without trusting producer-controlled text.
4. Run one supervised check, inspect its permissions and artifacts, then enable
   the weekly schedule.

## Acceptance criteria

- Formatting-only and irrelevant guide changes report `unchanged`.
- Representative semantic changes identify the affected guide identities,
  physical operations, and contract locations.
- Every command-controlled outcome produces a sanitized machine-readable
  report. Report construction retains bounded finding detail even when the
  complete comparison is large, and workflow failures use only the fixed
  trusted fallback.
- A run cannot exceed its validated request budget or retry implicitly.
- Repeated findings update one drift issue; recovery is recorded once and the
  issue is never closed automatically.
- Non-default refs cannot obtain issue-writing authority.
- Drift automation cannot modify the specification or initiate delivery work.

## Validation

- `go test -race ./...`
- `go vet ./...`
- `go run ./cmd/opendart-tool verify --repository-root .`
- `git diff --check`
- Read-only inspection of the public group-page markup confirmed the canonical
  container, table class, and header contract; caption text is intentionally
  treated as structural but not group-specific because the source reuses it.

## Next action

Authorization-gated: run one supervised manual default-branch check and inspect
its permissions, bounded artifact, and issue behavior. Enable a weekly schedule
only after that separate review and authorization. Do not dispatch the workflow,
write a GitHub issue, or enable scheduling in the completed work-3 slice.
