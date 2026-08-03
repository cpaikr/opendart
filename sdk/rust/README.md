# Rust SDK workspace

This isolated Cargo workspace contains:

- `opendart`, the public protocol SDK; and
- `opendart-cli`, the independently versioned binary-only public CLI.

Consumer builds use Cargo and the checked-in Rust source. The SDK implementation
is handwritten. Private Go tooling and canonical OpenAPI inputs remain
repository-only and generate only the CLI interface and private dispatch
projections described by
[ADR 0004](../../docs/decisions/0004-handwritten-rust-sdk-conformer.md).

## Verification

Run commands from the repository root. For a focused SDK edit loop:

```sh
./scripts/verify fast rust -p opendart
```

Run the required Linux Rust pull-request contract with:

```sh
./scripts/verify rust
```

This installs the pinned stable and MSRV toolchains, fetches locked dependencies,
then runs offline formatting, Clippy, feature combinations, tests, rustdoc,
WebAssembly and dependency-graph checks, reqwest compatibility, package
inventories, workspace packaging, and a clean CLI installation.

The faithful pre-push contract composes that Rust mode with the required Linux
Go contract:

```sh
./scripts/verify pre-push
```

`scripts/verify` is the command source shared by local development and Linux
GitHub Actions. Native macOS and Windows artifact checks remain CI-owned.

To reproduce the repository-only CLI loopback seams directly:

```sh
RUSTFLAGS="--cfg opendart_compat" cargo +1.97.1 test --locked --offline --manifest-path sdk/rust/Cargo.toml -p opendart-cli --test structured_loopback
RUSTFLAGS="--cfg opendart_compat" cargo +1.97.1 test --locked --offline --manifest-path sdk/rust/Cargo.toml -p opendart-cli --test binary_loopback
```

`opendart_compat` enables private loopback origins and deterministic failure
seams that are unavailable in consumer builds. These commands are
credential-free and make no live OpenDART request.

## Workspace boundaries

The workspace manifests and `rust-toolchain.toml` are authoritative for the
toolchain, MSRV, features, target-specific dependencies, and exact local package
pins.

The SDK's transport-independent surface remains available with default features
disabled. The default native client is absent on WebAssembly even when its
feature is selected. The CLI uses an exact local SDK and JSON encoder pin;
changing either behavior-defining pin requires CLI compatibility review.

The package inventories require Cargo's `.cargo_vcs_info.json` for the exact
source revision. The SDK also exposes `source_provenance()` for the semantic
specification source and canonical bundle checksum.

## Documentation

- [Repository layout](../../docs/rust-sdk/repository-layout.md)
- [SDK public contract](../../docs/rust-sdk/public-contract.md)
- [CLI projection generation](../../docs/rust-sdk/generation.md)
- [Transport and safety](../../docs/rust-sdk/transport-and-safety.md)
- [SDK verification and release](../../docs/rust-sdk/verification-and-release.md)
- [CLI architecture](../../docs/rust-cli/architecture.md)
- [CLI public contract](../../docs/rust-cli/public-contract.md)
- [CLI verification and release](../../docs/rust-cli/verification-and-release.md)

Current SDK delivery state is tracked in the
[Public Rust SDK plan](../../plans/rust/public-rust-sdk.md).
