# SDK wire trust boundaries

## Outcome

Make response metadata, XML status inspection, binary replay, and XML value
normalization fail closed at the Rust SDK's public wire boundaries.

## Status

Complete. [PR #46](https://github.com/cpaikr/opendart/pull/46) and its reviewed
follow-up passed the required Linux, macOS, and Windows verification before
merging into `rust`.

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
- Fresh security and design review findings are resolved, and the complete
  local stable, MSRV, package, compatibility, repository, and install gates
  pass.
- CodeRabbit's full review found a partially form-decoded credential case.
  Post-decoding `+` bytes are now treated as ambiguous literal-plus or
  form-space bytes, with focused unit and transport coverage.

## Remaining

- None.

## Next action

No further action. This file remains as the completion record.
