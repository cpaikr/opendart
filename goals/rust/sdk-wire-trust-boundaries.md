# SDK wire trust boundaries

## Outcome

Make response metadata, XML status inspection, binary replay, and XML value
normalization fail closed at the Rust SDK's public wire boundaries.

## Status

Implementation and local release validation are complete. Delivery is active
in [PR #46](https://github.com/cpaikr/opendart/pull/46); remote verification,
review resolution, and merge remain.

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

## Remaining

- Complete remote Linux, macOS, and Windows verification.
- Resolve any PR feedback and merge PR #46.

## Next action

Monitor PR #46 checks, address any actionable feedback, and merge only after
required verification and review state permit it.
