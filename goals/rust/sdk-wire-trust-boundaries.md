# SDK wire trust boundaries

## Outcome

Make response metadata, XML status inspection, binary replay, and XML value
normalization fail closed at the Rust SDK's public wire boundaries.

## Status

In progress. The XML authority contract is resolved and implementation is
passing focused SDK and CLI coverage; full review, validation, and PR delivery
remain.

## Completed

- Response metadata now checks each of three bounded percent-decoding stages.
- Malformed, ambiguous, nested, credential-bearing, and marker-bearing values
  are omitted without rewriting retained bytes.
- Unit, transport, CLI, Clippy, stable, and MSRV-focused checks pass for the
  metadata slice in commit `010e47d`.
- Candidate behavior, compatibility gaps, and the stop decision are recorded
  in `plans/rust/sdk-wire-trust-boundaries.md`.
- `roxmltree` validates complete bounded UTF-8 XML 1.0 documents before the
  existing converter can construct values or classify status.
- Preflight bounds nesting and per-element attributes before the recursive
  authority pass.
- The adversarial and valid XML corpora, exact EOL assertions, malformed binary
  replay, and structured/binary CLI cases pass.

## Remaining

- Complete SDK, CLI, package, repository, MSRV, and CI validation.
- Resolve the required fresh-context code review.
- Deliver, review, and merge the feature PR.

## Next action

Run the required review, resolve findings, and execute the complete release
gate before PR delivery.
