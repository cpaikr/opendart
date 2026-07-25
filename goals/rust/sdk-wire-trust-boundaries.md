# SDK wire trust boundaries

## Outcome

Make response metadata, XML status inspection, binary replay, and XML value
normalization fail closed at the Rust SDK's public wire boundaries.

## Status

Blocked on the accepted XML grammar or authority contract. The maintained
parser spike found no candidate satisfying all current requirements without a
broader surface or a documented restriction.

## Completed

- Response metadata now checks each of three bounded percent-decoding stages.
- Malformed, ambiguous, nested, credential-bearing, and marker-bearing values
  are omitted without rewriting retained bytes.
- Unit, transport, CLI, Clippy, stable, and MSRV-focused checks pass for the
  metadata slice in commit `010e47d`.
- Candidate behavior, compatibility gaps, and the stop decision are recorded
  in `plans/rust/sdk-wire-trust-boundaries.md`.

## Remaining

- Decide the supported XML grammar and authoritative parser.
- Add public malformed and valid XML corpora plus exact EOL assertions.
- Preserve malformed binary bodies byte-for-byte as unrecognized evidence.
- Complete SDK, CLI, package, repository, MSRV, and CI validation.
- Deliver, review, and merge the feature PR.

## Next action

Obtain the XML contract decision described in the plan, then resume from the
public-boundary regression corpus.
