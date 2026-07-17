# OpenDART API Specification

This directory contains a source-backed OpenAPI 3.1.2 description of every API
operation published in the six official OpenDART development-guide groups.

## Inventory

- 85 logical endpoints
- 167 physical request paths
- 82 endpoints with separate JSON and XML paths
- 3 binary ZIP endpoints whose advertised path ends in `.xml`
- 337 documented request arguments
- 2,383 documented response-field rows
- 13 shared API-level message codes

The source was checked on 2026-07-17 (Asia/Seoul). Each operation links to its
official guide page and retains its group code, API ID, source descriptions,
request table, reference tables, and verification date under `x-opendart`.

## Files

- [`openapi.yaml`](openapi.yaml) is the canonical multi-file entry point.
- `paths/` contains one Path Item fragment per physical request URL.
- `schemas/` contains one conservative response schema per logical endpoint.
- `components/schemas.yaml` contains the shared status-code definition.
- [`generated/openapi.bundle.yaml`](generated/openapi.bundle.yaml) is the
  portable single-file bundle for external tools.
- [`redocly.yaml`](redocly.yaml) contains strict linting rules.

The fragments and bundle are generated from the official guide. Do not edit
them by hand; update the extractor or its source-normalization rules instead.

## Source fidelity

OpenDART documents response keys and descriptions but does not document their
types. The normalized schemas therefore leave scalar types open. The raw source
rows, indentation classes, icon classes, source order, and normalization
diagnostics are retained under each schema's `x-opendart` extension.

Request arguments are modeled as query strings, matching the guide's
`STRING(n)` declarations. Their exact documented type, required flag, and
description are retained, but narrative lengths, enums, defaults, ranges, and
date shapes are not promoted to validators. That would require interpreting
prose and could make a source-backed contract stricter than OpenDART itself.

The guide also does not specify HTTP status-code behavior or literal response
`Content-Type` headers. Operations use a `default` response, and media types are
marked as inferred from the documented output format. These are not presented
as verified wire-level facts.

Coverage, acquisition identity, successful-empty semantics, partition closure,
and historical availability remain `probe-required` until empirical evidence is
available. This keeps documented facts distinct from collection analysis.

### Binary exceptions

- `DS001-2019003`: `/document.xml`
- `DS001-2019018`: `/corpCode.xml`
- `DS003-2019019`: `/fnlttXbrl.xml`

The guide labels each as `Zip FILE (binary)`. It warns that Chrome and Edge may
save the download with an `.xml` extension and instructs users to rename it to
`.zip`. Only `corpCode.xml` documents fields inside the archived XML file.

## Refresh and validation

The tooling is isolated from future application technology under
`tools/openapi/`:

```sh
cd tools/openapi
npm ci --ignore-scripts
npm run sync:opendart -- --checked-at YYYY-MM-DD
npm run bundle:opendart
npm run verify:opendart
```

The repository-owned catalog check verifies inventory counts, group coverage,
physical representations, normalized-parameter parity, schema ownership,
provenance, source-table totals, local references, and orphaned fragments.
Redocly then validates both the multi-file description and its bundled form;
verification also proves the committed bundle matches a fresh build without
rewriting it.
