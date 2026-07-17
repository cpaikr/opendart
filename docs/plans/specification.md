# OpenDART Specification Plan

## Objective

Create a source-backed OpenAPI 3.2 specification for every endpoint published
in the six OpenDART development-guide groups. Preserve all endpoint information
shown by the official guide while keeping crawler-specific analysis explicit and
separate from documented source facts.

## Authority and scope

- The official OpenDART development guide is the source of truth.
- The extraction baseline was accepted against the consumer-owned
  [`dartdb` specification contract](https://github.com/cpaikr/dartdb/blob/81093c36f0c68ae6aefa36bc0af5a66c403e0c52/docs/research/opendart-spec-contract.md).
  This repository owns the external-source contract and does not mirror that
  application-specific document.
- The initial inventory is 85 logical endpoints across `DS001` through `DS006`.
- Interactive test controls, page chrome, commented-out content, and transient
  API results are outside the persisted specification, except test-form values
  that are necessary to establish an otherwise undocumented request encoding.

## Decisions

- OpenAPI 3.2 is the canonical format; generated prose is secondary. If a
  downstream SDK generator still requires 3.1, generate a compatibility
  artifact instead of weakening the canonical contract.
- Physical JSON, XML, and download URLs remain distinct OpenAPI paths.
- Exact Korean source descriptions are retained.
- Source provenance and operational coverage fields use `x-opendart` extensions.
- Undocumented behavior is marked as requiring a probe, not inferred silently.
- Request prose is preserved without promoting narrative constraints to
  machine validators; unambiguous constraints can be curated separately if a
  code-generation contract is later required.
- The specification is an executable contract for applications. The two
  multi-company `corp_code` parameters are arrays because the official test
  forms demonstrate comma-separated values; `style: form` and `explode: false`
  encode that wire form. Message `021` supplies the 100-company maximum.
- Repeated parameters, schemas, and message codes are shared with `$ref` where
  doing so does not erase endpoint-specific requirements or source wording.
- ZIP success bodies use the canonical raw-binary representation. Empirically
  observed XML API errors remain a second media type on the same catch-all
  response, with observation metadata distinct from guide facts.
- Curated guide contradictions are attached to the affected parameter or
  response field without correcting either source value.
- Complete refreshes must pass catalog validation and strict OpenAPI lint in a
  staging tree before publication.

## Current state

- The specification, generator, offline validation, and reusable probe now live
  in a standalone filtered-history repository with root-level package commands.
- The official guide was checked on 2026-07-17 and all six groups were
  extracted: 85 logical endpoints and 167 physical request paths.
- The generated catalog preserves 337 request-argument rows, 2,383
  response-field rows, 13 shared message codes, and endpoint-specific reference
  tables.
- The canonical multi-file OpenAPI 3.2.0 description and portable bundle both
  pass the repository catalog checker and strict Redocly validation.
- Response schemas retain the guide's `result` XML root independently of
  generated component names.
- Guide tables and notes are complete. Remaining undocumented wire behavior,
  quota semantics, response scalar types, and coverage-planning claims remain
  empirical work and are explicitly marked `probe-required` rather than
  inferred.
- All three ZIP endpoints model both documented ZIP success and the observed
  HTTP 200 XML status-`010` error representation.
- Known request-length, request-cardinality, and response-label contradictions
  are machine-readable. Both multi-company `corp_code` parameters preserve the
  request-table contradiction while exposing the guide-supported array
  contract. Authenticated success verification remains pending.

## Work

- [x] Define the root document, shared components, provenance extension, and
      extraction rules.
- [x] Generate all endpoint path and schema fragments from the official guide.
- [x] Bundle the multi-file description into one portable OpenAPI document.
- [x] Validate OpenAPI syntax, references, inventory counts, and source-table
      preservation.
- [x] Review the generated specification against the handoff contract and record
      remaining empirical probes.
- [x] Adopt OpenAPI 3.2 raw-binary and XML element semantics.
- [x] Model observed ZIP XML errors and curate known source contradictions.
- [x] Validate complete and partial refresh staging trees before publication.
- [x] Resolve multi-company array serialization from the official test examples
      and message `021`, and add a credential-safe authenticated probe.
- [ ] Run the authenticated multi-company probe and persist its sanitized
      observation in the specification metadata.

## Validation

- `npm run sync:opendart -- --checked-at 2026-07-17` regenerated the complete
  catalog from the official guide.
- Node syntax checks passed for the sync, catalog-check, and bundle-freshness
  tooling scripts.
- `npm run verify:opendart` passed the catalog checker, strict lint of the
  multi-file entry point, byte-for-byte bundle freshness checking, and strict
  lint of the bundle.
- Refresh ownership guards preserve an unmarked bundle, and invalid calendar
  dates are rejected before source requests begin.
- Catalog validation rejects missing, altered, or symlinked ownership markers
  before publication.
- A complete live refresh passed catalog and Redocly validation before
  publication; a one-endpoint `--only` refresh passed structural-only catalog
  validation and strict lint before publication.
- Verified totals: 85 logical endpoints, 167 physical paths, 337 request
  arguments, 2,383 response rows, and 13 message codes.
- A complete 2026-07-17 refresh extracted both official multi-company test
  examples and message `021`, passed staged catalog/OpenAPI validation, and
  published the array contract.
- Eight offline probe tests cover canonical, repeated-key, and single-value URL
  serialization; JSON/XML identity extraction; malformed XML; unexpected
  canonical identities; and non-distinct single-company baselines.
- Four offline synchronization tests cover the trusted guide URL boundary,
  endpoint identity validation, path-like and duplicate query rejection, and
  curated response-field contradictions.
  A missing `OPENDART_API_KEY` is rejected before any request. The live
  authenticated matrix has not run because the key is not present in this
  process environment.
- After repository extraction, `npm ci --ignore-scripts` and
  `npm run verify:opendart` passed from the repository root with the portable
  bundle unchanged at SHA-256
  `f622a6a849207523fd1f675c7b681fa65fd0c019b4066b61009d863b13081f3f`.

## Next action

Run `npm run probe:opendart-multi-company` with `OPENDART_API_KEY` available,
then promote the four physical parameters from authenticated verification
`pending` to the sanitized observed result. Afterward continue quota,
throttling, successful-emptiness, response-type, enumerability, acquisition,
closure, and historical-coverage probes.
