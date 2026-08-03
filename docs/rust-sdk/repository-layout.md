# Rust SDK repository layout

## Purpose

This document defines ownership of the handwritten Rust SDK and the retained
CLI projections. Delivery state belongs in the
[Public Rust SDK plan](../../plans/rust/public-rust-sdk.md).

## Topology

```text
opendart/
├── openapi/                         canonical sources and portable bundle
├── cmd/opendart-tool/               private Go repository CLI
├── internal/
│   ├── openapi/                     confined OpenAPI 3.2 boundary
│   ├── rustinterface/               reviewed Rust and CLI name bindings
│   └── sdkgen/                      CLI projections only
├── sdk/rust/
│   ├── Cargo.toml                   isolated Cargo workspace
│   ├── Cargo.lock                   committed verification lock
│   └── crates/
│       ├── opendart/src/
│       │   ├── values/              private recurring input validation
│       │   ├── protocol/            private preparation and decode mechanics
│       │   ├── operations/          handwritten inputs, wrappers, and conformers
│       │   ├── request/             pure request and authorization capability
│       │   ├── wire/                bounded normalized source evidence
│       │   ├── client.rs            optional safe native HTTP client
│       │   └── provenance.rs        release-selected source identity
│       └── opendart-cli/src/
│           ├── generated/interface/ public grammar and discovery projection
│           ├── generated/dispatch/  private exhaustive typed adapter
│           └── *.rs                 handwritten process and artifact policy
└── docs/
    ├── rust-sdk/                    durable SDK contracts
    └── rust-cli/                    durable CLI contracts
```

There is no generated SDK source tree. Each logical SDK operation is
handwritten under its semantic family module and colocates its input,
representation-specific preparation, opaque response wrapper, private decoder,
and focused tests. Shared modules own only mechanics proven identical across
operations.

## Ownership

| Component | Owns | Does not own |
| --- | --- | --- |
| `openapi/` | Protocol paths, parameters, schemas, statuses, and media | Rust or CLI naming; runtime policy |
| `internal/openapi` | Confined source inspection and repository checks | Public Go APIs or runtime parsing |
| `internal/rustinterface` | Reviewed Rust and CLI product-name bindings | Wire facts |
| `internal/sdkgen` | CLI interface and private dispatch projections | SDK implementation or HTTP semantics |
| `opendart` | Handwritten preparation, validation, decoding, source evidence, and optional safe client | Retry, quota, collection, persistence, or domain policy |
| `opendart-cli` | Grammar, credentials, process output, and artifact behavior | A second protocol conformer |

`openapi/openapi.yaml` and its confined references are the canonical contract;
the portable bundle, handwritten Rust, and generated CLI trees are not
alternate operation inventories.

## Cargo and feature boundary

The isolated workspace contains the `opendart` SDK and binary-only
`opendart-cli` package. With default features disabled, request preparation,
authorization, operation identity, source-backed wrappers, and wire inspection
remain available without an HTTP runtime. Private validated values support
handwritten operation inputs without creating a second public input API.
`client-reqwest` enables the target-specific native client; the client remains
absent on WebAssembly. `serde-json` adds source-shaped JSON serialization for
structured evidence without making credentials, requests, clients, or streams
serializable.

The CLI exact-pins its local SDK dependency and reviewed JSON encoder behavior.
Verification requires no-default-feature and WebAssembly graphs to exclude
native HTTP, TLS, proxy, DNS, and async-runtime packages.

## Generated and handwritten source

SDK source is ordinary handwritten Rust and is formatted with rustfmt. Only the
two marked CLI subtrees are generator-owned; correct their canonical inputs or
private renderer and regenerate them together. Their independent transaction
and freshness contract is documented in
[CLI projection generation](generation.md).

`src/provenance.rs` records crate version, the selected semantic specification
source release, and canonical bundle checksum. Cargo supplies
`.cargo_vcs_info.json` for the exact packaged revision. Package inventories must
contain the handwritten modules and must exclude SDK generator implementation,
generated SDK source, repository-private inputs, and local artifacts.
