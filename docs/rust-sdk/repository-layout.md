# Rust SDK repository layout

## Purpose

This document defines where the public Rust SDK lives and which repository
component owns each part of its implementation. Delivery state and next actions
belong in the [Public Rust SDK plan](../../plans/rust/public-rust-sdk.md).

The topology below is current. [ADR 0004](../decisions/0004-handwritten-rust-sdk-conformer.md)
accepts a handwritten-only SDK layout and a retained CLI interface projection;
the combined implementation plan will update this page when that layout exists.

## Topology

The implemented boundary is:

```text
opendart/
├── openapi/                         canonical OpenAPI sources and portable bundle
├── cmd/opendart-tool/               private Go repository CLI
├── internal/
│   ├── openapi/                     private OpenAPI 3.2 boundary
│   └── sdkgen/
│       ├── model/                   repository-owned normalized SDK model
│       └── rust/                    deterministic SDK and CLI renderers
├── sdk/rust/
│   ├── Cargo.toml                   isolated Cargo workspace
│   ├── Cargo.lock                   committed verification lock
│   ├── rust-toolchain.toml          pinned repository toolchain
│   └── crates/
│       ├── opendart/
│       │   ├── src/generated/       generator-owned Rust projection
│       │   ├── src/request/         handwritten request and authorization kernel
│       │   ├── src/wire/            handwritten source-envelope model
│       │   ├── src/client.rs        optional native HTTP client
│       │   └── src/provenance.rs    release-selected source identity
│       └── opendart-cli/
│           ├── src/generated/       generator-owned command breadth and dispatch
│           └── src/*.rs             handwritten process and output behavior
└── docs/
    ├── rust-sdk/                    durable SDK contracts
    └── rust-cli/                    durable CLI contracts
```

`cmd/opendart-tool` remains private maintainer tooling. The user-facing CLI is
the separate `opendart-cli` Cargo package because its release, credential,
configuration, output, and compatibility obligations differ from repository
tooling. Its boundaries are documented in the
[CLI architecture](../rust-cli/architecture.md).

## Ownership

| Component | Owns | Does not own |
| --- | --- | --- |
| `openapi/` | Source-backed API paths, parameters, schemas, and extensions | Rust naming or runtime policy |
| `internal/openapi` | Confined loading and the repository-owned SDK surface | Public Go APIs |
| `internal/sdkgen` | Normalization, validation, deterministic SDK and CLI projections | Consumer runtime behavior |
| `opendart` | Request preparation, authorization, wire evidence, and the optional safe client | Collection, retry, quota, persistence, or domain policy |
| `opendart-cli` | Command grammar, credential access, process output, and artifact behavior | A second endpoint inventory |

`openapi/openapi.yaml` and its confined references are the canonical contract.
`openapi/generated/openapi.bundle.yaml` is the portable specification artifact;
neither generated Rust tree is an alternate source of API facts.

The generator is a subcommand of `opendart-tool`. Do not add another Rust,
Python, Node, shell, build-script, or proc-macro generator. The complete
projection and ownership rules are defined in
[Rust SDK generation](generation.md).

## Cargo package and feature boundary

The isolated workspace contains the `opendart` SDK and binary-only
`opendart-cli` packages. Keep the SDK as one package until an independently
consumed module requires its own compatibility or release policy; internal
module boundaries are sufficient today.

The SDK feature boundary is:

- With default features disabled, request construction, authorization,
  operation identity, wire inspection, and generated response types remain
  available without an HTTP runtime.
- `client-reqwest` enables the optional native client. Its dependencies are
  target-specific, and the public client module is absent on WebAssembly.
  Prepared requests remain available there for caller-owned execution.
- `serde-json` adds direct JSON serialization for response evidence. It does
  not make credentials, requests, clients, or streams serializable.
- Generated code contains no conditional transport behavior.

The package manifest is the source of truth for the exact feature-to-dependency
mapping. Verification requires the no-default-features graph to remain free of
native HTTP, TLS, proxy, DNS, and async-runtime packages.

The CLI uses an exact-version local SDK dependency with the client and JSON
features. Updating that pin is a CLI compatibility decision and does not by
itself release the independently versioned CLI.

## Generated and handwritten source

Each Rust product has one independently marked generated subtree. Handwritten
modules expose the supported interface and hide generated layout where
possible. Only the private generator may replace those owned trees; its
transaction and freshness guarantees belong in
[Rust SDK generation](generation.md).

Do not edit `src/generated` by hand. Correct the canonical contract, normalized
model, renderer, or handwritten runtime seam and regenerate.

`src/provenance.rs` is intentionally handwritten. It records the release-selected
specification source, canonical bundle checksum, generator schema, and SDK
projection checksum. Cargo supplies `.cargo_vcs_info.json` for the exact package
revision. Provenance and release verification are described in
[Rust SDK verification and release](verification-and-release.md).

## Future language packages

A later language SDK should consume the repository-owned normalized model and
language-neutral request vectors, then own idiomatic runtime, errors, package
metadata, and release policy in its ecosystem. It must not parse generated Rust
or copy the operation inventory.

Do not stabilize a serialized SDK manifest until a second implemented emitter
demonstrates that the format is useful. The in-memory Go model remains private
repository infrastructure.
