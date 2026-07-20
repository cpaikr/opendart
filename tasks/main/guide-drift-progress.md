# Public guide semantic drift progress

This current-state handoff tracks [Public guide semantic drift](guide-drift.md).
Update it in place and keep authorization-gated actions explicit.

## Current slice

- Integration target: `dev`; feature work is delivered through sequential,
  non-stacked pull requests and never targets `main`.
- Ordered work 2 is implemented and validated on `feat/guide-drift-command`.
  Review findings about candidate-owned multi-company evidence, bounded safe
  findings, real end-to-end fixtures, and path/location integrity are resolved.
- Ordered work 3 begins only after the ordered-work-2 PR is reviewed and merged.

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

## Validation

- `go test -race ./...` passes.
- `go vet ./...` passes.
- `go run ./cmd/opendart-tool verify --repository-root .` passes all phases.
- `git diff --check` passes.
- Required local code review is complete; PR checks and CodeRabbit review are
  pending.

## Blockers

None.

## Next action

Commit ordered work 2, merge its reviewed PR into `dev`, then create the ordered
work 3 automation branch. Stop before workflow dispatch, external issue writes,
or schedule enablement.
