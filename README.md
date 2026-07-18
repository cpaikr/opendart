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

The repository tooling requires only the Go version declared in `go.mod`:

```sh
go run ./cmd/opendart-tool sync --checked-at YYYY-MM-DD
go run ./cmd/opendart-tool bundle \
  --root openapi/openapi.yaml \
  --output openapi/generated/openapi.bundle.yaml
go vet ./...
go test -race ./...
go run ./cmd/opendart-tool verify --repository-root .
```

`sync` refreshes the canonical files from the public guide through in-process
validated staging and owned-output publication, then invalidates the old
bundle. `bundle` deterministically rebuilds the portable artifact. CI owns Go
vetting and race-enabled tests separately; `verify` checks catalog and confined
references, strict linting, the sanitized auditor-evidence manifest,
release/workflow guards, and byte-for-byte bundle freshness.

Generated OpenAPI files are reviewed artifacts. Do not edit them by hand; change
the extractor or its normalization rules and regenerate them. OpenAPI 3.2 is
canonical. If a consumer requires OpenAPI 3.1, create a separate compatibility
artifact rather than changing the source contract.

## Credentialed probe

Refresh and verification require no OpenDART API key. Credentialed commands are
focused probes with fixed request matrices:

```sh
varlock run -- go run ./cmd/opendart-tool probe-multi-company
varlock run -- go run ./cmd/opendart-tool probe-auditor-evidence
```

Install the standalone Varlock CLI with `brew install dmno-dev/tap/varlock`.
The committed `.env.schema` marks `OPENDART_API_KEY` as required and sensitive;
put the local value in the ignored `.env.local`, then run
`varlock encrypt --file .env.local` if it is plaintext. Varlock validates and
injects the key into the child process without putting it in command arguments.
The Go command still reads only `OPENDART_API_KEY` from its process environment.

Do not commit the local override or capture authenticated URLs or raw response
bodies. The probes run sequentially without automatic retries and emit
sanitized JSON observations; they do not print the key or persist response
bodies. The auditor probe is the reproducible source for the committed,
sanitized [auditor evidence manifest](docs/api/evidence/auditor-2026-07-18.json).

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
  records the completed repository-tooling migration.
- The [guide-drift plan](docs/plans/guide-drift.md) and
  [live-conformance plan](docs/plans/live-conformance.md) track remaining
  maintenance and empirical work.
- The [external-auditor retrieval guide](docs/api/auditor.md) separates the
  canonical endpoint contracts from a layered, empirically informed lookup
  strategy.
- The [Rust SDK plan](docs/plans/rust-sdk/README.md) proposes a future public
  crate derived from the canonical contract. It does not describe a currently
  published runtime package.
