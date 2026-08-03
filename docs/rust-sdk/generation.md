# Rust CLI projection generation

## Purpose

The Rust SDK is handwritten. Private Go tooling generates only two checked-in
CLI projections:

- the public command and discovery interface; and
- the private exhaustive adapter from that interface to handwritten SDK types.

Consumer builds use Cargo and never run Go, parse OpenAPI, or download source
inputs. Delivery state belongs in the
[Public Rust SDK plan](../../plans/rust/public-rust-sdk.md).

## Inputs and authority

Generation reads the canonical OpenAPI document through `internal/openapi` and
the reviewed product-language manifests under `sdk/rust/interface`. OpenAPI is
the sole authority for physical and logical identities, parameters,
representations, and source descriptions. The manifests bind those identities
to approved CLI vocabulary and private handwritten Rust symbols without
copying paths, schemas, statuses, media types, or HTTP behavior.

The CLI interface projection contains only reviewed command names, logical-ID
aliases, flags, source concepts, constraints, representations, physical
identities, descriptions, guide URLs, and coarse output shape. It contains no
Rust input, field, preparation-method, or response-wrapper names.

The private dispatch projection contains the Rust module, input, parameter,
preparation-method, and response-wrapper names required for exhaustive typed
wiring. It does not contain provider validation, request serialization,
response decoding, response schemas, authorization, or transport behavior.
Those remain handwritten SDK responsibilities.

## Owned outputs

The projections have separate ownership markers, schema versions, checksums,
and freshness identities:

```text
sdk/rust/crates/opendart-cli/src/generated/
├── interface/
│   ├── .opendart-cli-interface-generated
│   ├── catalog.rs
│   ├── command.rs
│   └── mod.rs
└── dispatch/
    ├── .opendart-cli-dispatch-generated
    ├── adapter.rs
    ├── dispatch_cases.json
    └── mod.rs
```

The interface identity deliberately excludes private Rust symbols. A Rust
rename may change the dispatch projection but cannot silently change public CLI
grammar or discovery. `dispatch_cases.json` is checked independently against
the reviewed manifest inventory so correlated generator omissions fail closed.

## Transaction and freshness

The repository command loads and validates all inputs, renders both trees into
staging directories, validates complete contents and ownership, and replaces no
accepted tree until both staged products are valid. Partial publication rolls
back. The command may replace only the two marked CLI subtrees.

Repository verification renders both projections in memory and compares their
complete contents with the committed trees without rewriting the working tree.
The generated files are reviewed source and may use compact generator-owned
formatting; Cargo still compiles, lints, tests, and documents their consumers.

Generation fails on incomplete or duplicate identity coverage, stale or orphan
interface mappings, CLI command/alias/flag collisions, ambiguous
representations, missing handwritten Rust bindings, nondeterministic output, or
invalid ownership. There is no fallback generic endpoint or SDK-generation
path.

## Boundaries

OpenAPI-derived tooling may also produce bundles, inventories, coverage checks,
and test scaffolding. It must not emit SDK request builders, validators,
validated values, wire types, response wrappers, or decoders. The repository
must not add `build.rs`, a specification-parsing proc macro, a runtime OpenAPI
parser, a serialized Rust schema manifest, or a network download.

A later language SDK may reuse canonical identities and independently authored
conformance evidence, but it owns its own idiomatic implementation and release
policy. A shared serialized SDK model is introduced only if a second real
conformer demonstrates that need.
