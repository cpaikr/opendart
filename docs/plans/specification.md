# OpenDART Specification Plan

## Objective

Create a source-backed OpenAPI 3.2 specification for every endpoint published
in the OpenDART development guide. Preserve all endpoint information
shown by the official guide while keeping crawler-specific analysis explicit and
separate from documented source facts.

## Authority and scope

- The official OpenDART development guide is the source of truth.
- The extraction baseline was accepted against the consumer-owned
  [`dartdb` specification contract](https://github.com/cpaikr/dartdb/blob/81093c36f0c68ae6aefa36bc0af5a66c403e0c52/docs/research/opendart-spec-contract.md).
  This repository owns the external-source contract and does not mirror that
  application-specific document.
- The canonical inventory is the generated root document and its path
  references; prose does not duplicate its volatile totals.
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
- Repository-owned extraction, generation, validation, drift, and live-test
  tooling will converge on one repository-internal Go CLI. This does not make
  the tooling a supported application or package; see the accepted
  [Go decision](../decisions/0001-go-repository-tooling.md).

## Current state

- The source-backed specification and initial extraction contract are complete.
  The canonical inventory and snapshot identity live in `openapi/openapi.yaml`.
- The generated multi-file OpenAPI 3.2 description and portable bundle pass the
  repository catalog, lint, reference, and freshness checks.
- Response schemas retain the guide's `result` XML root independently of
  generated component names.
- Guide tables, notes, source diagnostics, and endpoint-specific reference
  tables are preserved. Undocumented wire behavior remains `probe-required`
  rather than inferred.
- Binary operations model documented ZIP success and the observed XML API-error
  representation as separate media types.
- Known request and response contradictions are machine-readable. The
  multi-company `corp_code` parameters expose the guide-supported array wire
  contract while retaining the conflicting request-table declaration.
- Node.js still implements repository tooling. Its replacement is planned in
  [`go-tooling-migration.md`](go-tooling-migration.md); this completed source
  contract is not the migration tracker.

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
- [x] Move remaining guide-drift and authenticated observation work into
      separate plans with independent permission and issue boundaries.

## Validation

- A complete guide refresh passed staged catalog and OpenAPI validation before
  atomic publication; partial refresh behavior is covered separately.
- Repository verification checks source-table preservation, inventory and
  representation invariants, local references, ownership markers, strict lint,
  and byte-for-byte bundle freshness.
- Refresh tests cover trusted guide URLs, endpoint identities, invalid dates,
  rejected query shapes, curated contradictions, staging ownership, and
  rollback behavior.
- Probe tests cover supported query serialization, JSON/XML identity parsing,
  malformed responses, comparison failures, missing credentials, and sanitized
  output without making live requests.
- The targeted authenticated matrix has not run. It is subsumed by the complete
  physical-operation conformance plan rather than remaining a specification
  completion gate.

## Related plans

- [Go tooling migration](go-tooling-migration.md) replaces the current Node.js
  implementation without changing the product boundary.
- [Public-guide semantic drift](guide-drift.md) covers credential-free guide
  comparison, artifacts, and its deduplicated issue.
- [Credentialed live conformance](live-conformance.md) covers every physical API
  operation behind a separate security and issue boundary.

## Next action

The source-contract plan is complete. Begin with the compatibility gate in
[`go-tooling-migration.md`](go-tooling-migration.md), then implement public-guide
drift and complete live conformance as separate consumers. Any empirical fact
promoted into specification metadata still requires a separate reviewed change.
