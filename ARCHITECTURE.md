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
    -> focused multi-company or auditor-evidence probe
    -> sanitized observation on stdout

OpenDART API + protected environment key
    -> manual trusted-main live conformance producer
    -> bounded sanitized report artifact
    -> isolated default-branch notifier
    -> one persistent GitHub issue
```

Specification refresh is a deliberate local network operation. Pull-request
verification does not refresh from OpenDART, and release automation publishes
only the committed bundle after the offline gate passes.

Guide synchronization, offline repository verification, and the focused live
probes are owned by `cmd/opendart-tool`, the single internal Go CLI.
`internal/openapi` isolates the selected OpenAPI libraries behind
repository-owned types and owns confined references, strict linting,
deterministic bundling, freshness, semantic comparison, and response
validation. Repository-owned tooling is Go-only; former Node and Redocly checks
remain only as historical cutover evidence in the accepted decision record and
completed plans.

## Runtime flows

### Refresh and bundle

`go run ./cmd/opendart-tool sync` invokes `internal/guide`, which fetches only
the trusted OpenDART guide surface, normalizes the discovered catalog, and
renders managed files into a staging directory. Go-owned catalog,
confined-reference, and strict lint checks validate that tree in process before
publication.

Publication replaces the managed entries through a sequence of filesystem
renames and attempts rollback when publication fails. It is not an atomic
directory swap because the output also contains unmanaged configuration and
release artifacts. Existing managed output is replaced only when its ownership
marker is valid, and that marker remains present throughout publication so an
interrupted run remains owned and repairable by the next refresh. A successful
refresh removes the prior portable bundle, which must then be regenerated
explicitly from the committed multi-file description.

### Verify and release

CI runs `go vet ./...`, `go test -race ./...`, and
`go run ./cmd/opendart-tool verify --repository-root .`. The verifier checks
catalog invariants, strict lint for the multi-file source and bundle,
the committed auditor-evidence manifest's strict sanitized schema,
the complete live-conformance inventory's offline coverage and request budget,
release/workflow policy, and byte-for-byte Go bundle freshness. It does not
read a credential, rewrite a committed artifact, or contact OpenDART.

`.github/workflows/verify.yml` runs that gate for pull requests, reusable
workflow calls, and manual dispatches with read-only repository permission.
On `main`, `.github/workflows/release-please.yml` first calls the same gate, then
allows Release Please to manage the version, changelog, tag, and draft release.
It attaches the bundle and checksum before publishing the immutable release.

### Focused live probe

`varlock run -- go run ./cmd/opendart-tool probe-multi-company` validates and
injects `OPENDART_API_KEY` from an ignored local override according to the
committed `.env.schema`. The Go command still receives the credential only from
its process environment. It tests the two documented multi-company operations
across JSON and XML, using the canonical comma-separated encoding and a
repeated-key control. Requests are sequential and have no retry. Responses are
bounded, parsed, validated against the committed OpenAPI representation, and
discarded. The probe emits a sanitized JSON observation, does not change the
specification, and has no scheduled GitHub workflow.

`varlock run -- go run ./cmd/opendart-tool probe-auditor-evidence` uses the same
credential boundary and a separate fixed, bounded request matrix. It records
only allowlisted request coordinates, response summaries, archive hashes, and
semantic assertions needed by the external-auditor guide. Raw bodies,
authenticated URLs, and arbitrary headers or error text are excluded. Its
reviewed output is durable empirical evidence, not an input to canonical
specification generation. JSON responses pass the committed OpenAPI validator.
Document responses are accepted only after a positive ZIP signature and then
parsed under probe-specific entry and expansion bounds without extracting
source-controlled member paths; this empirical adapter also accommodates the
observed non-contract media type and CP949-compatible content.

### General live conformance

`opendart-tool live-conformance --preflight-only` loads the canonical OpenAPI
document and proves exact primary-case coverage, trusted routing, valid
requests, typed assertions, fixed discovery partitions, pagination closure,
and the derived request ceiling without reading `OPENDART_API_KEY`. The normal
command performs that same preflight before credential access, resolves rare
event coordinates through bounded reusable discovery, then executes every
physical operation once. JSON, XML, and ZIP bodies are bounded, validated,
semantically checked, and discarded; only the strict versioned report remains.
Observed download media and Korean archive encodings are normalized only after
positive bounded ZIP validation.

`.github/workflows/live-conformance.yml` is manual-only, requires the canonical
repository's `main` ref, and exposes `OPENDART_API_KEY` only to the live command
inside the declared protected environment. It has read-only repository access
and uploads only the report file. The separate
`.github/workflows/live-conformance-notify.yml` runs from the trusted
default-branch workflow definition after a producer completes. It checks out
the exact trusted producer revision, has no environment or OpenDART secret,
and gives issue-write permission only to the isolated notifier. The notifier
strictly decodes the bounded report; missing, malformed, or inconsistent
artifacts become a fixed failure derived only from Actions metadata. Failures
update one marker-owned issue, recovery is recorded once, and automation never
changes issue state. The protected environment and credential remain
unconfigured, and the workflow has not been dispatched or scheduled.

`internal/liveprobe` confines the live-only HTTP policy shared by credentialed
repository tools.
On 2026-07-18, the live OpenDART origin required a TLS 1.2 RSA key-exchange
suite that modern Go does not enable by default. The probe client adds only the
required AES-GCM suite to Go's secure suite set; this compatibility path lacks
forward secrecy and must not be reused by released SDK or general application
transport. Re-test the origin with Go's default transport whenever the dated
live evidence is refreshed or the Go toolchain changes, and remove the
exception as soon as the default handshake succeeds. Ambient HTTP proxies are
disabled because authenticated query parameters must go only to the fixed
OpenDART origin.

## Code map

- Start with `openapi/openapi.yaml` for the canonical multi-file contract.
  `openapi/paths/`, `openapi/schemas/`, and `openapi/components/` contain its
  generated fragments; `openapi/.opendart-spec-output` marks the managed tree.
- `openapi/generated/openapi.bundle.yaml` is the portable release interface.
- Start with `cmd/opendart-tool/main.go` for the repository command surface.
  `internal/openapi` owns third-party OpenAPI types, confined reference loading,
  semantic comparison, strict lint, deterministic bundle artifacts, and
  response validation.
  `internal/guide` owns trusted acquisition, normalization, deterministic
  generation, staged validation, guarded publication, and rollback.
  `internal/verification` coordinates the offline repository gate, while
  `internal/releaseguard` owns release and workflow policy.
- `internal/multicompanyprobe` owns the fixed credentialed probe, including its
  request, assertion, pacing, response-bound, and sanitized-report policy.
- `internal/auditorprobe` owns the fixed external-auditor evidence matrix,
  bounded disclosure/document inspection, and sanitized evidence schema.
- `internal/liveconformance` owns the canonical primary-case registry, bounded
  discovery, fail-closed execution, semantic adapters, and notifier-safe report.
- `internal/livenotifier` owns strict report consumption, fixed workflow
  failure fallback, GitHub issue deduplication, and recovery recording.
- `internal/liveprobe` owns the shared one-attempt HTTP transport and its
  upstream-confined TLS compatibility exception.
- `.github/workflows/verify.yml` is the credential-free repository gate.
  `.github/workflows/live-conformance.yml` is the manual protected producer;
  `.github/workflows/live-conformance-notify.yml` is its credential-isolated
  default-branch notifier.
  `.github/workflows/release-please.yml`, `release-please-config.json`, and
  `.release-please-manifest.json` own release automation.
- `.env.schema` is the committed Varlock contract for local credentialed
  commands. Secret values remain in ignored, locally encrypted overrides.
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
- Third-party OpenAPI types do not cross `internal/openapi`; reference loading
  is local-only and physically confined to the selected specification tree.
- Focused probes receive their key only from `OPENDART_API_KEY`; the local
  workflow validates and injects it through Varlock, and probe output never
  contains the key, an authenticated URL, or an unrestricted response body.
- Release automation cannot publish until the read-only verification job
  succeeds.
- Non-default live workflow refs receive neither the protected API credential
  nor issue-writing authority. The notifier accepts only trusted default-branch
  producer metadata and never receives producer logs or arbitrary error text.
- No current automation modifies the specification from guide drift or live API
  observations. Specification changes remain reviewed repository changes.

## Evolution

[ADR 0001](docs/decisions/0001-go-repository-tooling.md) records the completed
migration of repository-owned tooling from Node.js to one internal Go CLI. The
[guide-drift](tasks/main/guide-drift.md) task defines future work not yet part
of the current runtime. The
[live-conformance](tasks/main/live-conformance.md) task tracks protected
workflow, notification, and supervised scheduling work beyond the current
manual command. The
[Rust SDK task](tasks/rust/public-rust-sdk.md)
proposes an explicit future product-boundary change while retaining Go as
private repository tooling; no SDK is part of the current system until that
decision and implementation are completed.
