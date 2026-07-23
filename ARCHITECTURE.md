# Repository architecture

## Product boundary

This repository maintains two public products derived from one source-backed
OpenDART contract:

- the portable OpenAPI bundle; and
- the `opendart` Rust protocol SDK.

The official OpenDART development guide is authoritative for documented
behavior. Authenticated observations remain separate evidence. Go packages and
`cmd/opendart-tool` are private repository tooling, not supported public APIs.

## System map

```text
OpenDART guide
    -> staged acquisition and normalization
    -> canonical multi-file OpenAPI 3.2
    -> deterministic bundle ----------------------> GitHub specification release
    -> repository-owned SDK model
    -> deterministic checked-in Rust -------------> opendart crate package

OpenDART guide
    -> manual trusted-main semantic comparison
    -> bounded sanitized report artifact
    -> isolated default-branch notifier
    -> one persistent drift issue

OpenDART API + local API key
    -> fixed, bounded probes
    -> sanitized empirical evidence

OpenDART API + protected environment key
    -> manual trusted-main live conformance
    -> bounded sanitized report artifact
    -> isolated default-branch notifier
    -> one persistent GitHub issue
```

Pull-request verification never refreshes from OpenDART and never receives an
API key. It validates the committed specification, generated artifacts, live
case inventory, Rust workspace, package contents, and release policy.

## Runtime flows

### Specification refresh

`go run ./cmd/opendart-tool sync` acquires the trusted guide surface through
`internal/guide`, renders a complete staging tree, and validates it before
replacing the owned canonical output. A successful refresh invalidates the old
bundle; `opendart-tool bundle` regenerates it explicitly.

`internal/openapi` confines third-party OpenAPI types and local reference
loading. It owns strict linting, deterministic bundling, semantic comparison,
response validation, and the repository-owned SDK input projection.

### Rust generation and use

`opendart-tool generate-sdk` passes the canonical contract through
`internal/sdkgen/model` and the Rust renderer. Generated source is reviewed and
committed beneath `sdk/rust/crates/opendart/src/generated`; consumer builds do
not run Go or parse OpenAPI.

The crate exposes generated operation types plus handwritten request,
authorization, wire-inspection, and provenance contracts. Its core performs no
I/O. The optional default `client-reqwest` feature adds one-attempt bounded HTTP
execution with redirects, retries, ambient proxies, and automatic response
decoding disabled. Applications retain persistence, quota, retry, collection,
and domain policy.

### Verify and package

`.github/workflows/verify.yml` runs read-only Go and Cargo gates. Dependency
fetching is explicit; compilation, tests, documentation, generation checks,
and package inspection then run offline. The verifier requires exact generated
operation coverage, complete live-case coverage and request budgets, a tracked
crate inventory, matching Cargo versions, source provenance, and approved
workflow and release configuration.

Cargo records the exact packaged source revision in `.cargo_vcs_info.json`.
The crate's `source_provenance()` additionally records the canonical bundle,
SDK projection, generator schema, and applicable specification release.

### Releases

Release Please owns independent components:

- root `vX.Y.Z` tags and `CHANGELOG.md` for the OpenAPI bundle; and
- `opendart-vX.Y.Z` tags and the crate changelog/version for the Rust SDK.

Rust changes are excluded from root release eligibility. The Rust component is
configured to prepare a draft component release and keep the crate manifest and
workspace lock aligned. This repository does not yet authorize `cargo publish`;
registry publication and post-publication verification belong to work 6.

### Focused live probes

`probe-multi-company` and `probe-auditor-evidence` use a Varlock-injected local
`OPENDART_API_KEY`, fixed request matrices, bounded bodies, sequential attempts,
and sanitized output. They do not change the specification or SDK.

### General live conformance

`opendart-tool live-conformance --preflight-only` proves exact primary-case
coverage, trusted routing, valid requests, typed assertions, fixed discovery
partitions, pagination closure, and the derived request ceiling before reading
`OPENDART_API_KEY`. The normal command performs the same preflight, resolves
rare event coordinates through bounded discovery, and executes every physical
operation once. JSON, XML, and ZIP bodies are bounded, validated, semantically
checked, and discarded; only the strict versioned report remains.

`.github/workflows/live-conformance.yml` is manual-only, requires the canonical
repository's `main` ref, grants read-only repository access, exposes
`OPENDART_API_KEY` only to the live command inside the declared protected
environment, and uploads only the report file. The separate
`.github/workflows/live-conformance-notify.yml` runs from the trusted
default-branch workflow definition after a producer completes. It checks out
the exact trusted producer revision, has no environment or OpenDART secret,
and gives issue-write permission only to the isolated notifier. The notifier
strictly decodes the bounded report; missing, malformed, or inconsistent
artifacts become a fixed failure derived only from Actions metadata. Failures
update one marker-owned issue, recovery is recorded once, and automation never
closes the issue. The protected environment and credential remain
unconfigured, and the workflow has not been dispatched or scheduled.

`internal/liveprobe` confines the one-attempt HTTP policy shared by credentialed
repository tools. Its dated TLS compatibility exception lacks forward secrecy
and must not be reused by the released SDK transport. Re-test the origin with
Go's default transport whenever the dated evidence is refreshed or the Go
toolchain changes, and remove the exception when the default handshake works.
Ambient proxies are disabled so authenticated queries reach only the fixed
OpenDART origin.

### Public guide drift

`opendart-tool guide-drift` acquires the current public guide without a
credential, generates and structurally validates a temporary candidate, and
compares OpenAPI meaning after normalizing only snapshot metadata. It emits a
strict bounded report with `unchanged`, `changed`, or `error`; raw guide values
and exhaustive differences are never report fields.

`.github/workflows/guide-drift.yml` is manual-only and restricted to the
canonical repository's `main` ref with read-only contents permission. It
uploads only the attempt-scoped report. The separate default-branch
`workflow_run` notifier checks out the exact trusted producer revision and is
the only job with issue-write authority. Missing, oversized, malformed, or
conclusion-inconsistent artifacts become fixed failure state derived from
trusted Actions metadata. Changed and error outcomes update one drift issue;
only validated unchanged state records recovery once, and automation never
closes the issue. Offline release guards enforce these boundaries. Neither
workflow has been dispatched and no schedule is enabled.

## Code map

- `openapi/openapi.yaml` and its referenced fragments are the canonical source;
  `openapi/generated/openapi.bundle.yaml` is the portable bundle.
- `cmd/opendart-tool` is the private command surface.
- `internal/guide` owns guide acquisition and guarded generation.
- `internal/driftnotifier` owns strict drift-report consumption, fixed workflow
  failure fallback, issue deduplication, and recovery recording.
- `internal/openapi` confines OpenAPI dependencies and owns SDK projection.
- `internal/sdkgen/model` and `internal/sdkgen/rust` own deterministic SDK
  normalization and rendering.
- `sdk/rust` is the isolated Cargo workspace. The public crate lives under
  `sdk/rust/crates/opendart`; generated files have one owned subtree.
- `internal/verification` coordinates repository verification, while
  `internal/releaseguard` enforces workflow, package, provenance, and release
  policy.
- `internal/multicompanyprobe`, `internal/auditorprobe`, and
  `internal/liveprobe` own focused credentialed evidence collection.
- `internal/liveconformance` owns the canonical case registry, bounded
  discovery, fail-closed execution, semantic adapters, and sanitized report.
- `internal/livenotifier` owns strict report consumption, fixed workflow
  failure fallback, issue deduplication, and recovery recording.
- `.github/workflows/verify.yml` is the credential-free repository gate.
  Release Please configuration and `.github/workflows/release-please.yml` own
  component release preparation and specification asset publication. The two
  guide-drift workflows are the credential-free producer and isolated
  notifier; the two live-conformance workflows are the protected producer and
  isolated notifier.

## Invariants

- OpenAPI 3.2 remains canonical; generated SDK files never become an alternate
  endpoint inventory.
- Generated OpenAPI and Rust files change through their generators, not by
  hand, and verification requires byte-for-byte freshness.
- Guide facts, empirical observations, and executable policy remain separate.
- Offline verification makes no OpenDART request and requires no credential.
- Third-party OpenAPI types remain confined to `internal/openapi`; reference
  loading is local-only and confined to the selected specification tree.
- The SDK core can prepare and inspect every supported operation without an
  HTTP runtime. No SDK path silently retries or exposes a credential-bearing
  URL through safe diagnostics.
- Specification and crate versions, tags, changelogs, and release eligibility
  are independent. No current workflow has crates.io publication authority.
- Non-default live workflow refs receive neither the protected API credential
  nor issue-writing authority. The notifier accepts only trusted default-branch
  producer metadata and never receives producer logs or arbitrary error text.
- No automation modifies the specification from guide drift or live API
  observations. Specification changes remain reviewed repository changes.

## Decisions and evolution

[ADR 0001](docs/decisions/0001-go-repository-tooling.md) records the completed
migration to private Go repository tooling. [ADR 0002](docs/decisions/0002-public-rust-sdk.md)
accepts the first-party Rust SDK boundary. Current packaging and the remaining
publication/adoption work are tracked in the [public Rust SDK task](tasks/rust/public-rust-sdk.md).

[guide drift](tasks/main/guide-drift.md) owns credential-free acquisition and
semantic-comparison work. Drift-safe acquisition, the bounded command and
report, the manual producer, and the isolated notifier are implemented.
Supervised execution and scheduling remain authorization-gated. The
[live-conformance](tasks/main/live-conformance.md) task defers protected
environment setup, supervised execution, and weekly scheduling pending
explicit authorization; the runner, protected workflow definition, and
isolated notifier are implemented.
