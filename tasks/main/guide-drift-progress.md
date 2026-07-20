# Public guide semantic drift progress

This current-state handoff tracks [Public guide semantic drift](guide-drift.md).
Update it in place and keep authorization-gated actions explicit.

## Current slice

- Integration target: `dev`; feature work is delivered through sequential,
  non-stacked pull requests and never targets `main`.
- Ordered work 2 was reviewed in PR #31 and merged into `dev` with its commits
  preserved.
- Ordered work 3 implements and validates the manual trusted-main producer,
  attempt-scoped bounded artifact, isolated notifier, and credential-free
  workflow-policy checks on `feat/guide-drift-automation`.
- No workflow has been dispatched, no external issue has been written, and no
  schedule has been enabled.

## Decisions

- Compare the committed baseline as original and the generated candidate as
  updated after replacing only `info.version` and root/operation source
  `checkedAt` values in the candidate comparison model.
- Count every semantic difference, retain a deterministic bounded set of
  allowlisted operation/location findings, and never report source values or
  arbitrary diagnostics.
- Structural candidate validation permits valid inventory, representation,
  reference-table, message-code, and detail changes to reach comparison while
  malformed generation and validation remain errors.
- The notifier maps validated `changed` and fixed workflow failures to one
  active marker-owned issue; only validated `unchanged` records recovery once.
  It never closes the issue and never consumes producer-controlled text.
- The producer has read-only contents permission and no credential. Issue-write
  authority exists only in the default-branch `workflow_run` notifier, which
  checks out the exact trusted producer revision.

## Validation

- `go test -race ./...` passes.
- `go vet ./...` passes.
- `go run ./cmd/opendart-tool verify --repository-root .` passes all phases.
- `git diff --check` passes.
- `go test -race ./...`, `go vet ./...`, credential-free repository
  verification, and `git diff --check` pass.
- The required security-focused code review is complete. Its bounded issue-body
  finding was fixed with complete-line omission and an explicit shown count;
  the decoder-valid oversized-identifier regression test passes.

## Blockers

None.

## Next action

Review and merge the ordered-work-3 PR into `dev` with its commits preserved.
Stop before workflow dispatch, external issue writes, or schedule enablement;
ordered work 4 remains authorization-gated.
