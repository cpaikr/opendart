# OpenDART API Specification

This unofficial, community-maintained repository provides a source-backed
OpenAPI 3.2 description of the operations published in the official OpenDART
development guide. It is not affiliated with or endorsed by the Financial
Supervisory Service or OpenDART.

## Use the specification

The supported consumer artifact is
[`openapi/generated/openapi.bundle.yaml`](openapi/generated/openapi.bundle.yaml).
Use the copy attached to a versioned GitHub Release when reproducible builds or
long-lived pins matter. The repository also keeps the canonical multi-file
source at [`openapi/openapi.yaml`](openapi/openapi.yaml), with referenced Path
Items under `openapi/paths/` and response schemas under `openapi/schemas/`.

The Git tag versions this repository's bundle contract. The OpenAPI
`info.version` and `x-opendart.source.checkedAt` fields instead record the date
of the upstream guide snapshot; they do not imply that OpenDART follows SemVer.

Each operation records its official guide URL, group code, API ID, source
tables, and check date under `x-opendart`. The root document, its referenced
fragments, and the Go `catalog` validation command are the inventory source of
truth. Avoid copying volatile endpoint or field totals into documentation.

## Source fidelity

OpenDART's response tables document keys and descriptions but not field types.
Guide-derived endpoint fields therefore omit scalar type constraints unless
separate evidence supports them. Raw source rows, source order, indentation and
icon classes, and normalization diagnostics remain available under `x-opendart`.
Known contradictions in the guide are preserved under
`x-opendart-source-diagnostics`; the generator does not silently choose one
conflicting source value.

The guide records `result` as the response root. Schemas retain that XML name so
bundled component names do not change XML serialization. This metadata is a
logical mapping of the guide table, not an XSD or a claim of wire-level
validation.

Request arguments are generally strings, matching the guide's `STRING(n)`
declarations. Documented types, required flags, and descriptions are retained,
but narrative lengths, enums, defaults, ranges, and date shapes are not promoted
to validators without stronger evidence.

The multi-company operations are the deliberate exception. Their guide test
forms use comma-separated company codes and the guide documents a maximum of 100
companies, even though the request tables describe one `STRING(8)`. The public
parameter is therefore an array with `style: form` and `explode: false`, producing
`corp_code=CODE1,CODE2`. The conflicting source declaration, guide examples, and
current verification status remain in the operation's `x-opendart` metadata.

The guide does not define HTTP status behavior or literal response
`Content-Type` headers. Operations consequently use a `default` response, and
media types inferred from the documented format are marked as inferred. The
documented ZIP endpoints use `application/zip` for their binary success
representation and also model the empirically observed XML API-error shape.
See the associated `x-opendart` observation metadata for the evidence and date.

Coverage, acquisition identity, successful-empty semantics, dataset closure,
and historical availability remain `probe-required` unless empirical evidence
states otherwise. This keeps guide-sourced facts separate from observations and
collection analysis.

## Refresh and verify

The repository tooling requires the Go version declared in `go.mod`. Node.js
`>=22.12.0` or `20.19.x` and npm `>=10` remain temporarily required only for
the focused multi-company probe and its offline tests:

```sh
npm ci --ignore-scripts
npm run sync:opendart -- --checked-at YYYY-MM-DD
npm run bundle:opendart
npm run verify:opendart
```

`sync:opendart` runs the internal Go CLI, refreshes the canonical files from the
public guide through in-process validated staging and owned-output publication,
and invalidates the old bundle. `bundle:opendart` deterministically rebuilds the
portable artifact. `verify:opendart` runs the focused probe's offline tests, Go
tests, catalog and confined-reference checks, strict linting, release/workflow
guards, and a byte-for-byte bundle freshness check.

Generated OpenAPI files are reviewed artifacts. Do not edit them by hand; change
the extractor or its normalization rules and regenerate them. OpenAPI 3.2 is
canonical. If a consumer requires OpenAPI 3.1, create a separate compatibility
artifact rather than changing the source contract.

## Credentialed probe

Refresh and verification require no OpenDART API key. The only currently
implemented credentialed command is the targeted multi-company probe:

```sh
npm run probe:opendart-multi-company
```

Pass `OPENDART_API_KEY` only through the process environment. Do not commit it,
put it in command arguments, or capture authenticated URLs or raw response
bodies. The probe runs sequentially without automatic retries and emits a
sanitized JSON observation; it does not print the key or persist response
bodies.

The planned full live-conformance runner is not implemented. Its intended
credential, reporting, and evidence boundaries are tracked in the
[live-conformance plan](docs/plans/live-conformance.md).

## Releases

Humans classify bundle compatibility and choose the corresponding Conventional
Commit input. Release Please applies the configured version policy, updates the
generated changelog and manifest, creates the tag and draft release, and the
release workflow validates, attaches, and publishes the immutable release.
[`RELEASING.md`](RELEASING.md) is the maintainer policy and review checklist.

Each release contains `openapi.bundle.yaml` and
`openapi.bundle.yaml.sha256`. Consumers can verify GitHub's signed release
attestation and, after downloading an asset, its origin:

```sh
gh release verify vX.Y.Z --repo cpaikr/opendart
gh release verify-asset vX.Y.Z openapi.bundle.yaml --repo cpaikr/opendart
```

This repository publishes no runtime package and exposes no supported tooling
API.

## Repository documentation

- [`ARCHITECTURE.md`](ARCHITECTURE.md) maps repository boundaries, runtime flow,
  and security invariants.
- The [Go tooling ADR](docs/decisions/0001-go-repository-tooling.md) records the
  accepted direction; the [migration plan](docs/plans/go-tooling-migration.md)
  tracks work that is not yet implemented.
- The [guide-drift plan](docs/plans/guide-drift.md) and
  [live-conformance plan](docs/plans/live-conformance.md) track remaining
  maintenance and empirical work.
