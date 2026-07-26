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

CLI package ownership, local accepted-artifact comparison, and the mandatory
pause before publication are documented in the
[CLI verification and release guide](../../docs/rust-cli/verification-and-release.md).

The no-default-features normal dependency graph must not contain `reqwest`,
Tokio, Hyper, TLS, proxy, DNS, or streaming-runtime dependencies. The default
`client-reqwest` feature is native-target-only; the transport-independent core
remains the portable public surface.

The package contains Cargo's `.cargo_vcs_info.json` for exact source revision
and exposes `source_provenance()` for the crate version, semantic specification
source release, independently selected canonical bundle checksum, generator
schema, and SDK projection checksum.
