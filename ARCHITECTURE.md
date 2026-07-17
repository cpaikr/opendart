# Repository architecture

## Purpose and boundary

This repository publishes a source-backed OpenAPI description of OpenDART. The
portable bundle is the product. Repository tooling exists only to acquire,
generate, validate, test, and release that specification; it is not a public
application, SDK, or reusable package.

The official development guide is authoritative for documented behavior.
Authenticated API observations are separate evidence about the live service and
must remain distinguishable from guide-sourced facts.

## Current system

```text
OpenDART guide
    -> staged source acquisition and normalization
    -> validated multi-file OpenAPI
    -> explicit bundle generation
    -> offline verification
    -> verified GitHub release assets

OpenDART API + local API key
    -> focused multi-company probe
    -> sanitized observation on stdout
```

Specification refresh is a deliberate local network operation. Pull-request
verification does not refresh from OpenDART, and release automation publishes
only the committed bundle after the offline gate passes.

## Runtime flows

### Refresh and bundle

`scripts/sync-opendart.mjs` fetches only the trusted OpenDART guide surface,
normalizes the discovered catalog, and renders managed files into a staging
directory. It runs the repository catalog check and strict Redocly lint against
that staging tree before publishing it.

Publication replaces the managed entries through a sequence of filesystem
renames and attempts rollback when publication fails. It is not an atomic
directory swap. Existing managed output is replaced only when its ownership
marker is valid. A successful refresh removes the prior portable bundle, which
must then be regenerated explicitly from the committed multi-file description.

### Verify and release

`npm run verify:opendart` runs the offline tests, catalog invariants, strict lint
for the multi-file description, a byte-for-byte bundle freshness check, and
strict lint for the bundle. The freshness check builds into a temporary
directory and does not rewrite the committed artifact.

`.github/workflows/verify.yml` runs that gate for pull requests, reusable
workflow calls, and manual dispatches with read-only repository permission.
On `main`, `.github/workflows/release-please.yml` first calls the same gate, then
allows Release Please to manage the version, changelog, tag, and draft release.
It attaches the bundle and checksum before publishing the immutable release.

### Focused live probe

`scripts/probe-multi-company.mjs` is the only credentialed implementation. It
uses `OPENDART_API_KEY` from the process environment to test the two documented
multi-company operations across JSON and XML, using the canonical
comma-separated encoding and a repeated-key control. Requests are sequential
and have no automatic retry. The probe emits a sanitized JSON observation, does
not change the specification, and does not persist response bodies. It has no
scheduled GitHub workflow.

## Code map

- Start with `openapi/openapi.yaml` for the canonical multi-file contract.
  `openapi/paths/`, `openapi/schemas/`, and `openapi/components/` contain its
  generated fragments; `openapi/.opendart-spec-output` marks the managed tree.
- `openapi/generated/openapi.bundle.yaml` is the portable release interface.
- Start with `package.json` for the current command surface. The corresponding
  implementations and offline tests live in `scripts/`.
- `scripts/sync-opendart.mjs` owns guide acquisition, normalization, generation,
  staged validation, and publication. `scripts/check-opendart.mjs` owns catalog
  and source-fidelity invariants. `scripts/check-opendart-bundle.mjs` owns bundle
  freshness.
- `.github/workflows/verify.yml` is the credential-free repository gate.
  `.github/workflows/release-please.yml`, `release-please-config.json`, and
  `.release-please-manifest.json` own release automation.
- `README.md` documents the artifacts and current commands. `RELEASING.md`
  defines compatibility classification and the manual release review gate.

## Current invariants

- OpenAPI 3.2 is canonical. Compatibility artifacts must not weaken or replace
  that contract.
- Generated OpenAPI files change through the generator and bundler, not by hand.
- Guide-sourced facts, empirical observations, and executable test policy remain
  separate and traceable.
- A canonical refresh validates the complete staging tree before replacing
  managed output. Partial refreshes are confined to noncanonical outputs. The
  publisher refuses broad, unsafe, symbolic-link, or unowned targets.
- Bundle generation is explicit after refresh, and verification requires the
  committed bundle to match a fresh build byte for byte.
- Offline verification makes no OpenDART request and requires no API key.
- The focused probe receives its key only from `OPENDART_API_KEY`; its output
  never contains the key, an authenticated URL, or an unrestricted response
  body.
- Release automation cannot publish until the read-only verification job
  succeeds.
- No current automation modifies the specification from guide drift or live API
  observations. Specification changes remain reviewed repository changes.

## Planned direction

[ADR 0001](docs/decisions/0001-go-repository-tooling.md) accepts migration of
repository-owned tooling from Node.js to one internal Go CLI. The active
[migration](docs/plans/go-tooling-migration.md),
[guide-drift](docs/plans/guide-drift.md), and
[live-conformance](docs/plans/live-conformance.md) plans describe work that is
not part of the current runtime.
