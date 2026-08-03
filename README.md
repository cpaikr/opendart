# OpenDART API Specification, Rust SDK, and CLI

This unofficial, community-maintained repository provides a source-backed
OpenAPI 3.2 description of the operations published in the official OpenDART
development guide, a first-party Rust protocol SDK derived from that contract,
and an agent-first command-line client built on the SDK. It is not affiliated
with or endorsed by the Financial Supervisory Service or OpenDART.

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
automatically. A closed, reviewed set of stable request constraints—such as
company-code and date formats, documented code sets, and pagination bounds—is
curated explicitly by the guide generator and enforced through the canonical
OpenAPI and handwritten SDK rather than inferred from prose.

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

## Rust SDK

The first-party `opendart` crate provides handwritten typed requests,
source-backed response wrappers, bounded wire inspection, a one-attempt
convenience client, and a transport-independent core. Usage examples and the
supported caller contract live in the
[crate guide](sdk/rust/crates/opendart/README.md); repository build and package
gates live in the [SDK workspace guide](sdk/rust/README.md).

## Rust CLI

The binary-only `opendart-cli` crate provides strict machine-readable discovery
and one-attempt structured or binary execution through the typed SDK. Its public
behavior is documented in the
[CLI contract](docs/rust-cli/public-contract.md), and source-package and release
boundaries are in the [CLI verification guide](docs/rust-cli/verification-and-release.md).

From the repository root, install and inspect one operation with:

```sh
cargo +1.97.1 install --locked --path sdk/rust/crates/opendart-cli
opendart --version
opendart operations describe company-overview
opendart call company-overview --help
```

After configuring `OPENDART_API_KEY` as described under
[Credentialed probe](#credentialed-probe), make a read-only call through the
repository's local environment wrapper:

```sh
./scripts/with-opendart-env -- opendart call company-overview \
  --company-code 00126380 \
  --representation json
```

## Refresh and verify

Specification tooling uses the Go version declared in `go.mod`; Rust
verification uses the pinned toolchains documented in the SDK workspace guide:

```sh
go run ./cmd/opendart-tool sync --checked-at YYYY-MM-DD
go run ./cmd/opendart-tool bundle \
  --root openapi/openapi.yaml \
  --output openapi/generated/openapi.bundle.yaml
./scripts/verify rust-conformance
./scripts/verify fast rust -p opendart
./scripts/verify pre-push
./scripts/verify exhaustive
go run ./cmd/opendart-tool live-conformance --preflight-only --repository-root .
```

`sync` refreshes the canonical files from the public guide through in-process
validated staging and owned-output publication, then invalidates the old
bundle. `bundle` deterministically rebuilds the portable artifact. The
repository verifier checks catalog and confined references, strict linting, the
sanitized auditor-evidence manifest, the live-matrix coverage, budget, and
sanitization preflight, release/workflow guards, and byte-for-byte bundle
freshness.

The fast tier requires an explicit Go package or Cargo test selection and is
the normal edit loop. `pre-push` runs the complete Linux pull-request contract:
Go vet, all normal Go tests, the repository verifier, the audited targeted-race
set, and every pinned stable, compatibility, MSRV, package-content, and clean
Rust install gate. `exhaustive` adds `go test -race ./...`. Credentialed live
conformance is deliberately outside all three tiers and still runs only
through `scripts/with-opendart-env`.

CI runs the Go and Rust contracts independently and adds native CLI artifact
checks on macOS and Windows. The stable aggregate `verify` job succeeds only
when every required job succeeds. The repository-owned verification script and
release guard are the source of truth for the exact package, race, compatibility,
and artifact gates.

`.github/workflows/full-race.yml` runs the exhaustive Go race sweep weekly on
the default branch and supports manual candidate-branch runs. A scheduled
failure is a maintainer-owned regression: reproduce with
`./scripts/verify exhaustive`, land the repair, manually rerun the workflow on
the candidate branch, and link the successful run. Do not dismiss it as
best-effort telemetry.

Generated OpenAPI files are reviewed artifacts. Do not edit them by hand; change
the extractor or its normalization rules and regenerate them. OpenAPI 3.2 is
canonical. If a consumer requires OpenAPI 3.1, create a separate compatibility
artifact rather than changing the source contract.

`opendart-tool guide-drift` compares the current public guide with the committed
contract without modifying either. Its trusted producer emits a bounded report,
and an isolated notifier validates that report before updating the persistent
drift issue. The [guide-drift task](tasks/main/guide-drift.md) owns delivery
status. Complete Rust verification commands live in
[`sdk/rust/README.md`](sdk/rust/README.md).

## Credentialed probe

Refresh, verification, and live-conformance preflight require no OpenDART API
key. Credentialed commands use reviewed, bounded request matrices:

```sh
./scripts/with-opendart-env -- go run ./cmd/opendart-tool probe-multi-company
./scripts/with-opendart-env -- go run ./cmd/opendart-tool probe-auditor-evidence
./scripts/with-opendart-env -- \
  go run ./cmd/opendart-tool live-conformance --repository-root .
```

Copy `.env.example` to the ignored `.env.local`, add the 40-character
`OPENDART_API_KEY`, and set its permissions to `600`. The dependency-free
wrapper rejects symbolic links, unsafe ownership or permissions, unexpected
entries, and invalid key syntax before injecting the key only into its child
process:

```sh
cp .env.example .env.local
chmod 600 .env.local
```

The Go and Rust commands still read only `OPENDART_API_KEY` from their inherited
process environment. They do not load dotenv files themselves.

`.env.local` is plaintext. Its file mode and ignore rule reduce accidental
exposure, but any process running as the same user can read it.

Do not commit the local override or capture authenticated URLs or raw response
bodies. The probes run sequentially without automatic retries and emit
sanitized JSON observations; they do not print the key or persist response
bodies. The auditor probe is the reproducible source for the committed,
sanitized [auditor evidence manifest](docs/api/evidence/auditor-2026-07-18.json).

The full runner covers every canonical physical operation, emits only its
strict versioned report, and stops on the first discovery or primary-case
failure. The protected workflow runs that conformance pass and an audited Rust
CLI smoke test; its isolated notifier has no access to the OpenDART credential
and accepts only the sanitized report. The
[live-conformance task](tasks/main/live-conformance.md) owns environment,
execution, and scheduling status.

## Releases

Humans classify public compatibility and choose the corresponding Conventional
Commit input. Release Please owns independent specification, Rust SDK, and Rust
CLI components. Specification releases use `vX.Y.Z`; crate releases use
`opendart-vX.Y.Z` or `opendart-cli-vX.Y.Z` and update their owned manifest,
lockfile entry, and changelog. An SDK proposal also updates the CLI manifest's
marked exact local SDK pin without changing the CLI version or changelog; this
does not constitute a CLI release.
[`RELEASING.md`](RELEASING.md) is the maintainer policy and review checklist.

Each specification release contains `openapi.bundle.yaml` and
`openapi.bundle.yaml.sha256`. Consumers can verify GitHub's signed release
attestation and, after downloading an asset, its origin:

```sh
gh release verify vX.Y.Z --repo cpaikr/opendart
gh release verify-asset vX.Y.Z openapi.bundle.yaml --repo cpaikr/opendart
```

Repository Go tooling remains private. Rust crate publication is independently
authorized and must preserve the SDK-before-CLI dependency recorded in
[ADR 0003](docs/decisions/0003-agent-first-opendart-cli.md). See the
[Rust roadmap](ROADMAP_RUST.md) for delivery priority and
[`RELEASING.md`](RELEASING.md#rust-crate-releases) for the release contract.

## Repository documentation

- [`ROADMAP.md`](ROADMAP.md) owns non-Rust delivery priority, while
  [`ROADMAP_RUST.md`](ROADMAP_RUST.md) owns Rust product priority and
  scheduling. Their linked work items own implementation status, blockers, and
  next actions.
- [`ARCHITECTURE.md`](ARCHITECTURE.md) maps repository boundaries, runtime flow,
  and security invariants.
- The [Go tooling ADR](docs/decisions/0001-go-repository-tooling.md) records the
  accepted private-tooling boundary and migration rationale.
- The [guide-drift task](tasks/main/guide-drift.md) and
  [live-conformance task](tasks/main/live-conformance.md) track remaining
  maintenance and empirical work.
- The [external-auditor retrieval guide](docs/api/auditor.md) separates the
  canonical endpoint contracts from a layered, empirically informed lookup
  strategy.
- The [Rust SDK decision](docs/decisions/0002-public-rust-sdk.md) and
  [CLI decision](docs/decisions/0003-agent-first-opendart-cli.md) record the
  accepted public-product boundaries; their guides document the implemented
  contracts.
