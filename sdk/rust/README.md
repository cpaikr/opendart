# Rust SDK workspace

This isolated Cargo workspace contains the public `opendart` SDK crate and the
binary-only `opendart-cli` crate. Consumer builds use Cargo only; the private Go
generator and canonical OpenAPI inputs remain repository tooling.

The stable gate is pinned to Rust 1.97.1 and the crate declares Rust 1.85.0 as
its MSRV. From the repository root, run a focused package test during the edit
loop:

```sh
./scripts/verify fast rust -p opendart
```

Run the exact Linux Rust pull-request contract, including locked dependency
fetch, stable and MSRV toolchains, formatting, Clippy, all feature
combinations, documentation, compatibility, dependency-graph, package-content,
and clean-install checks, with:

```sh
./scripts/verify rust
```

To reproduce the two CLI compatibility contracts directly from the repository
root, run:

```sh
RUSTFLAGS="--cfg opendart_compat" cargo +1.97.1 test --locked --offline --manifest-path sdk/rust/Cargo.toml -p opendart-cli --test structured_loopback
RUSTFLAGS="--cfg opendart_compat" cargo +1.97.1 test --locked --offline --manifest-path sdk/rust/Cargo.toml -p opendart-cli --test binary_loopback
```

The `opendart_compat` configuration activates repository-only loopback origins
and deterministic failure seams that are unavailable in ordinary consumer
builds. Both commands are credential-free and make no live OpenDART requests.

The faithful cross-language pre-push gate composes that exact Rust mode with
the required Linux Go contract:

```sh
./scripts/verify pre-push
```

`scripts/verify` is the command source shared by local development and the
Linux GitHub Actions jobs. Native macOS and Windows artifact checks remain
CI-owned.

The `opendart-cli` package includes its reviewed lockfile and exact local SDK
and JSON encoder pins. Moving either behavior-defining pin requires an explicit
CLI compatibility review; an SDK version update does not itself release the
independently versioned CLI.

The CLI live smoke test is skipped unless `OPENDART_LIVE_TESTS=1` and
`OPENDART_API_KEY` are both present. It performs only the reviewed read-only
structured and binary calls and keeps assertions structural.

SDK publication automation, first-release bootstrap, and the current stop gate
are documented in the
[SDK verification and release guide](../../docs/rust-sdk/verification-and-release.md).
CLI ownership and its dependent publication gate are documented separately in
the [CLI verification and release guide](../../docs/rust-cli/verification-and-release.md).

The no-default-features normal dependency graph must not contain `reqwest`,
Tokio, Hyper, TLS, proxy, DNS, or streaming-runtime dependencies. The default
`client-reqwest` feature is inert on WebAssembly and activates only
native-target dependencies; WebAssembly callers use prepared requests with
their own executor. Both default and no-default SDK configurations are checked
with Rust 1.97.1 for `wasm32-unknown-unknown`, including a dependency-tree
assertion that native client/runtime packages remain absent.

The package contains Cargo's `.cargo_vcs_info.json` for exact source revision
and exposes `source_provenance()` for the crate version, semantic specification
source release, independently selected canonical bundle checksum, generator
schema, and SDK projection checksum.
