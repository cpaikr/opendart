# OpenDART Specification Plan

## Objective

Create a source-backed OpenAPI 3.1 specification for every endpoint published
in the six OpenDART development-guide groups. Preserve all endpoint information
shown by the official guide while keeping crawler-specific analysis explicit and
separate from documented source facts.

## Authority and scope

- The official OpenDART development guide is the source of truth.
- The acceptance contract is
  [`docs/research/opendart-spec-contract.md`](../research/opendart-spec-contract.md).
- The initial inventory is 85 logical endpoints across `DS001` through `DS006`.
- Interactive test controls, page chrome, commented-out content, and transient
  API results are outside the persisted specification.

## Decisions

- OpenAPI 3.1 is the canonical format; generated prose is secondary.
- Physical JSON, XML, and download URLs remain distinct OpenAPI paths.
- Exact Korean source descriptions are retained.
- Source provenance and operational coverage fields use `x-opendart` extensions.
- Undocumented behavior is marked as requiring a probe, not inferred silently.
- Request prose is preserved without promoting narrative constraints to
  machine validators; unambiguous constraints can be curated separately if a
  code-generation contract is later required.
- Repeated parameters, schemas, and message codes are shared with `$ref` where
  doing so does not erase endpoint-specific requirements or source wording.

## Current state

- The official guide was checked on 2026-07-17 and all six groups were
  extracted: 85 logical endpoints and 167 physical request paths.
- The generated catalog preserves 337 request-argument rows, 2,383
  response-field rows, 13 shared message codes, and endpoint-specific reference
  tables.
- The canonical multi-file OpenAPI 3.1.2 description and portable bundle both
  pass the repository catalog checker and strict Redocly validation.
- Guide-documented contracts are complete. Wire behavior, quota semantics,
  response scalar types, and coverage-planning claims remain empirical work and
  are explicitly marked `probe-required` rather than inferred.

## Work

- [x] Define the root document, shared components, provenance extension, and
      extraction rules.
- [x] Generate all endpoint path and schema fragments from the official guide.
- [x] Bundle the multi-file description into one portable OpenAPI document.
- [x] Validate OpenAPI syntax, references, inventory counts, and source-table
      preservation.
- [x] Review the generated specification against the handoff contract and record
      remaining empirical probes.

## Validation

- `npm run sync:opendart -- --checked-at 2026-07-17` regenerated the complete
  catalog from the official guide.
- Node syntax checks passed for the sync, catalog-check, and bundle-freshness
  tooling scripts.
- `npm run verify:opendart` passed the catalog checker, strict lint of the
  multi-file entry point, byte-for-byte bundle freshness checking, and strict
  lint of the bundle.
- Verified totals: 85 logical endpoints, 167 physical paths, 337 request
  arguments, 2,383 response rows, and 13 message codes.

## Next action

Run the empirical probe program for quota and throttling behavior, HTTP status
and content-type behavior, successful-empty semantics, response scalar types,
enumerability, acquisition identity, closure evidence, and historical coverage.
