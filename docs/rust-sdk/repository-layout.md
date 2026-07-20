# Rust SDK Repository Layout

Planning source: [Public Rust SDK](../../tasks/rust/public-rust-sdk.md).

## Purpose

Place the public Rust SDK beside the canonical specification while preserving
clear boundaries between private repository tooling, generated SDK artifacts,
future language packages, and a possible user-facing CLI.

## Current structure

The implemented Rust paths are:

```text
opendart/
├── openapi/                         canonical source and portable bundle
├── cmd/
│   └── opendart-tool/               private Go repository CLI
├── internal/
│   ├── guide/                       guide acquisition and normalization
│   ├── openapi/                     private OpenAPI 3.2 boundary
│   └── sdkgen/
│       ├── model/                   language-neutral, repository-owned model
│       └── rust/                    deterministic Rust renderer
├── sdk/
│   └── rust/
│       ├── Cargo.toml               isolated Cargo workspace
│       ├── Cargo.lock               committed repository verification lock
│       ├── rust-toolchain.toml       repository toolchain pin
│       └── crates/
│           └── opendart/
│               ├── Cargo.toml
│               ├── README.md
│               ├── src/
│               │   ├── lib.rs
│               │   ├── request/     handwritten request kernel
│               │   ├── wire/        handwritten envelope and opaque values
│               │   ├── client.rs    optional reqwest convenience client
│               │   ├── provenance.rs reviewed source snapshot identity
│               │   └── generated/   reviewed generated Rust
│               └── tests/           public API and HTTP integration tests
└── docs/
    └── rust-sdk/                     stable design and maintainer contracts
```

Possible later additions:

```text
sdk/python/                           Python SDK package
sdk/typescript/                       TypeScript SDK targeting Node
sdk/rust/crates/opendart-cli/         Rust user CLI, if selected
```

The user-facing CLI must not be added to `cmd/opendart-tool`. The Go command is
trusted repository infrastructure with maintainer-oriented commands and a
private compatibility contract. A public CLI has different release,
configuration, credential, output, and compatibility obligations. It may
depend on the public Rust crate, but it remains a separate package.

## Why design documents live under `docs/rust-sdk`

Implementation state and the next action belong in the
[Public Rust SDK task](../../tasks/rust/public-rust-sdk.md). The supporting
design documents remain under `docs/rust-sdk` because their package boundaries,
contracts, and safety constraints carry forward into maintenance.
Installed-crate usage lives in the crate README.

## Package boundary

The workspace contains one package named `opendart`. Do not split it into
`opendart-core`, `opendart-types`, and transport crates:
the transport-independent modules can remain deep internal/public module
boundaries until independent consumers or release policies require separate
packages.

The feature model is:

```toml
[features]
default = ["client-reqwest"]
client-reqwest = ["dep:bytes", "dep:futures-core", "dep:reqwest", "dep:tokio"]
```

The feature boundary has these invariants:

- Request construction, authorization types, operation identity, and wire
  inspection compile with `default-features = false`.
- Tokio, `reqwest`, TLS, proxy, and HTTP streaming dependencies are reachable
  only through the optional convenience-client feature.
- The initial convenience client is native-target only because the required
  `reqwest` retry controls are not available on every WebAssembly target. Keep
  the transport-independent core portable where its selected dependencies
  allow it, and document target support explicitly.
- Enabling the client feature does not replace or change the low-level API.
- Generated code does not contain conditional transport behavior.

The foundation change also extends the repository `.gitignore` for the
isolated workspace's `target/` output without ignoring `Cargo.lock` or generated
Rust source.

If the CLI is later implemented in Rust, it joins the same isolated Cargo
workspace as a second crate and depends on `opendart`. Python and TypeScript
packages retain their native build systems and do not become Cargo members.

## Ownership

### Canonical contract

`openapi/openapi.yaml` and its referenced files own source-backed API facts.
`openapi/generated/openapi.bundle.yaml` remains the portable specification
artifact. No SDK file becomes an alternate endpoint inventory.

### Private generator

The Go packages under `internal/sdkgen` own normalization and language
rendering. Their types are not public Go APIs. Third-party OpenAPI model types
remain confined to `internal/openapi`; SDK generation consumes repository-owned
values.

The generator command remains a subcommand of `opendart-tool`. Do not add a
second Rust, Python, Node, shell, or build-script generator.

### Public Rust crate

The Rust crate owns source-protocol mechanics:

- operation identity and representation;
- typed request inputs and deterministic serialization;
- credential-safe authorization;
- source status and message extraction;
- conservative typed responses, raw wire values, and unknown-field retention; and
- an optional ordinary-user HTTP client.

It does not own collection coordinates, request footprints, acquisition
identity, persistence, artifact publication, retries, quotas, closure, or
domain interpretation.

## Generated and handwritten source

Keep generated files in one owned subtree with a marker header. Handwritten
modules expose the supported public interface and hide generated layout where
possible. Public documentation links to meaningful operation types rather than
generated file paths.

The generator may replace only its validated owned subtree. It must stage,
validate, format, and compare output before publication, following the same
guarded-publication principles as canonical specification generation.

Do not put handwritten patches in `src/generated`. If generation is wrong,
change the canonical contract, SDK model, emitter, or handwritten runtime seam
and regenerate.

`src/provenance.rs` is handwritten release metadata, not generated contract
output. It records the crate version, applicable semantic specification source
release, independently selected canonical bundle checksum, generator schema,
and SDK projection checksum. The release guard proves that the canonical source
inputs match the named tag without treating the generated bundle as tag-identical.
Cargo adds `.cargo_vcs_info.json` to the package for the exact source revision.
Ordinary specification regeneration does not update release-selected
provenance when the SDK projection is unchanged.

## Future-language boundary

The reusable cross-language asset is the private normalized SDK model and its
contract fixtures, not generated Rust syntax. A later language emitter must:

- consume the same operation identities and serialization rules;
- pass the same language-neutral request vectors;
- preserve the same source uncertainty and status policy boundary;
- own idiomatic language-specific runtime and error types; and
- publish through an independent package and release stream.

Do not commit a stable public SDK-manifest format until a second implemented
language proves that a serialized manifest is useful. The first Rust emitter
may consume the model in memory.

## Acceptance criteria

- Private Go tooling, the public Rust SDK, and a possible public CLI have
  distinct directories, commands, and compatibility policies.
- The Rust core builds without `reqwest`, Tokio, TLS, or a network runtime.
- The default client is optional without duplicating operation definitions.
- Native-client and transport-independent target support is explicit rather
  than inferred from transitive dependencies.
- Generated output has one owned subtree and no handwritten modifications.
- No empty future-language or CLI packages are committed.
- Adding another language requires a new emitter and native package, not a
  translation of generated Rust.
