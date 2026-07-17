# OpenDART API Specification

This unofficial, community-maintained repository contains a source-backed
OpenAPI 3.2.0 description of every API operation published in the official
OpenDART development guide. It is not affiliated with or endorsed by
the Financial Supervisory Service or OpenDART.

## Coverage

The canonical inventory is generated into
[`openapi/openapi.yaml`](openapi/openapi.yaml) and its referenced path fragments.
Treat those files and the repository catalog check as the source of truth rather
than duplicating volatile totals here.

Each operation links to its official guide page and retains its group code, API
ID, source descriptions, request table, reference tables, and verification date
under `x-opendart`. The current source snapshot date is recorded in
`info.version` and `x-opendart.source.checkedAt`.

## Artifacts

- [`openapi/openapi.yaml`](openapi/openapi.yaml) is the canonical multi-file
  entry point.
- `openapi/paths/` contains one Path Item fragment per physical request URL.
- `openapi/schemas/` contains one conservative response schema per logical
  endpoint.
- `openapi/components/schemas.yaml` contains the shared status-code and observed
  XML-error definitions.
- [`openapi/generated/openapi.bundle.yaml`](openapi/generated/openapi.bundle.yaml)
  is the stable, portable consumer artifact.
- [`openapi/redocly.yaml`](openapi/redocly.yaml) contains the current
  transitional linting rules.
- [`ARCHITECTURE.md`](ARCHITECTURE.md) maps the repository boundaries, flows,
  and security invariants.
- [`docs/decisions/0001-go-repository-tooling.md`](docs/decisions/0001-go-repository-tooling.md)
  records the accepted Go tooling decision.
- [`docs/plans/go-tooling-migration.md`](docs/plans/go-tooling-migration.md)
  plans the migration from the current Node.js scripts to one repository-internal
  Go CLI.
- [`docs/plans/specification.md`](docs/plans/specification.md) records completed
  source-contract work and routes remaining empirical work.
- [`docs/plans/guide-drift.md`](docs/plans/guide-drift.md) plans weekly semantic
  drift detection against the public guide.
- [`docs/plans/live-conformance.md`](docs/plans/live-conformance.md) plans weekly
  content-aware tests of every physical API operation behind a separate secret
  boundary.

The fragments and bundle are generated from the official guide. Do not edit
them by hand; update the extractor or its source-normalization rules instead.
OpenAPI 3.2 is canonical. If a downstream generator only supports 3.1, produce
a separate compatibility artifact rather than changing these source files.

## Source fidelity

OpenDART documents response keys and descriptions but does not document their
types. The normalized schemas therefore leave scalar types open. The raw source
rows, indentation classes, icon classes, source order, and normalization
diagnostics are retained under each schema's `x-opendart` extension.
Known contradictions inside the official guide are preserved next to the
affected parameter or response property under
`x-opendart-source-diagnostics`; neither source value is silently corrected.

The guide records `result` as the response root. Schemas retain that as the XML
root name so bundled component names do not alter XML serialization. The XML
metadata remains a logical mapping of the guide table, not an XSD or wire-level
validation claim.

Request arguments are generally modeled as query strings, matching the guide's
`STRING(n)` declarations. Their exact documented type, required flag, and
description are retained, but narrative lengths, enums, defaults, ranges, and
date shapes are not promoted to validators. That would require interpreting
prose and could make a source-backed contract stricter than OpenDART itself.

The multi-company operations are the deliberate exception. Although their
request tables describe `corp_code` as one `STRING(8)`, each official test form
supplies two comma-separated company codes, and message `021` states a maximum
of 100 companies. The app-facing parameter is therefore an array with one to
100 items, `style: form`, and `explode: false`. This produces
`corp_code=CODE1,CODE2`; the conflicting table declaration and exact guide
examples remain under `x-opendart` metadata. Authenticated success verification
is still marked `pending` until the live probe below returns both companies for
both endpoints and both response formats.

The guide also does not specify HTTP status-code behavior or literal response
`Content-Type` headers. Operations use a `default` response, and media types are
marked as inferred from the documented output format. These are not presented
as verified wire-level facts. The exception is the empirically observed XML
error representation on the binary endpoints listed below; its observation
metadata remains separate from guide-sourced facts.

Coverage, acquisition identity, successful-empty semantics, dataset closure,
and historical availability remain `probe-required` until empirical evidence is
available. This keeps documented facts distinct from collection analysis.

### Binary exceptions

- `DS001-2019003`: `/document.xml`
- `DS001-2019018`: `/corpCode.xml`
- `DS003-2019019`: `/fnlttXbrl.xml`

The guide labels these paths as `Zip FILE (binary)`. It warns that Chrome and
Edge may save the download with an `.xml` extension and instructs users to
rename it to `.zip`. Only `corpCode.xml` documents fields inside the archived
XML file.
Raw ZIP bodies use the OpenAPI 3.2 representation: the `application/zip` media
type with an unconstrained schema. Calls made with an invalid 40-character API
key on 2026-07-17 returned HTTP 200, `application/xml;charset=UTF-8`, and API
status `010`; that XML error is modeled as a second media type on the same
catch-all response.

## Current refresh and validation

The implemented tooling currently requires Node.js `>=22.12.0` or `20.19.x`,
with npm `>=10`:

```sh
npm ci --ignore-scripts
npm run sync:opendart -- --checked-at YYYY-MM-DD
npm run bundle:opendart
npm run verify:opendart
```

These commands remain authoritative until their Go replacements pass artifact
and validation parity. The accepted target is one repository-internal Go module
and CLI for source refresh, verification, semantic drift detection, and live
conformance. It is repository infrastructure, not a supported package. See the
[migration plan](docs/plans/go-tooling-migration.md).

The confirmed target uses the current stable Go toolchain, standard-library
CLI/HTTP/logging/encoding primitives, goquery for guide HTML, and libopenapi
behind one internal OpenAPI boundary. Live tests use OpenAPI for operation and
schema behavior plus typed Go cases for public inputs and endpoint-specific
meaning. Arazzo, Overlay, and general application frameworks are deferred.
Redocly remains only during parity comparison; the completed repository-owned
CLI toolchain is Go-only and may use Vacuum plus tested Go checks for equal or
stronger lint coverage. The
[Go ADR](docs/decisions/0001-go-repository-tooling.md) records the dependency
and compatibility policy.

To verify multi-company serialization with a real key, expose it only through
the process environment and run:

```sh
npm run probe:opendart-multi-company
```

The probe reads `OPENDART_API_KEY`, makes ten sequential requests, and emits a
sanitized JSON observation. It tests the canonical comma-separated form and a
repeated-query-key control across both endpoints and JSON/XML. Two single-company
baselines map the first endpoint's `corp_code` inputs to its `stock_code` response
identities. The probe never prints the key, an authenticated URL, or a complete
response body, and it performs no automatic retries.

This targeted probe is transitional. The planned Go runner derives every
physical operation from the canonical OpenAPI document, executes the complete
matrix weekly with committed public inputs plus narrowly budgeted discovery
where stable values are impractical, and requires representation and
endpoint-specific content assertions. It does not partition the matrix. See the
[live conformance plan](docs/plans/live-conformance.md).

The repository-owned catalog check verifies inventory counts, group coverage,
physical representations, normalized-parameter parity, schema ownership,
provenance, source-table totals, local references, and orphaned fragments.
Refresh runs the catalog check and strict Redocly lint against a staging tree
before publishing a complete catalog. Redocly also validates the committed
multi-file description and bundle; verification proves the bundle matches a
fresh build without rewriting it. During the Go migration, one reviewed
formatting-only bundle cutover is allowed after semantic equivalence is proven;
the replacement output then becomes the deterministic freshness golden.

## Provenance and releases

Every operation records its official guide URL, group code, API ID, source
tables, and check date under `x-opendart`. The specification is generated from
those guide pages; committed fragments are reviewed artifacts, not independent
claims about undocumented wire behavior.

The portable bundle is the release interface. Release Please owns semantic
versioning, `CHANGELOG.md`, the release manifest, tags, and GitHub Releases
from Conventional Commit history. Do not edit those generated release files or
move published tags manually; merge a Release Please pull request instead.
The Git tag versions this repository's bundle; `info.version` remains the
upstream guide snapshot date and does not claim that OpenDART itself follows
SemVer. [`RELEASING.md`](RELEASING.md) defines the manual compatibility policy,
pre-1.0 behavior, commit inputs, and release review gate.

Each release is validated before publication. The workflow creates a draft,
attaches `openapi.bundle.yaml` plus `openapi.bundle.yaml.sha256`, and only then
publishes it under GitHub's immutable-release policy. Consumers can pin a
version and verify the downloaded bundle with either the checksum or GitHub's
release attestation. This repository does not publish a runtime package or a
supported tooling API.

## Secrets and live conformance

Offline verification and source refreshes require no OpenDART API key. The
multi-company probe is the only currently implemented credentialed command.
Supply `OPENDART_API_KEY` through the process environment; never commit it,
place it in arguments, or include authenticated URLs or raw response bodies in
reports.
The probe makes ten sequential requests without automatic retries and emits
only a sanitized JSON observation. The future complete live runner preserves
the same boundary: the secret-bearing job remains read-only, and a separate job
may create or update only a sanitized, deduplicated conformance issue. If setup
or runner launch fails before a valid report exists, that job may render only a
fixed workflow-failure notice from trusted Actions metadata. Raw responses are
never persisted. After a reported failure recovers, automation adds one
recovery comment but leaves issue closure to a maintainer.
