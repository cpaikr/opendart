# Rust SDK workspace

This isolated Cargo workspace contains the public `opendart` SDK crate and the
binary-only `opendart-cli` crate. Consumer builds use Cargo only; the private Go
generator and canonical OpenAPI inputs remain repository tooling.

The stable gate is pinned to Rust 1.97.1 and the crate declares Rust 1.85.0 as
its MSRV. Fetch locked dependencies once, then run every build gate offline:

```sh
cargo +1.97.1 fetch --locked --manifest-path sdk/rust/Cargo.toml
cargo +1.97.1 fetch --locked --manifest-path sdk/rust/compat/reqwest-feature-unification/Cargo.toml

cargo +1.97.1 fmt --manifest-path sdk/rust/Cargo.toml --all -- --check
cargo +1.97.1 clippy --locked --offline --manifest-path sdk/rust/Cargo.toml --workspace --all-targets --all-features -- -D warnings
cargo +1.97.1 clippy --locked --offline --manifest-path sdk/rust/Cargo.toml -p opendart --all-targets --no-default-features -- -D warnings
cargo +1.97.1 test --locked --offline --manifest-path sdk/rust/Cargo.toml --workspace --all-features
cargo +1.97.1 test --locked --offline --manifest-path sdk/rust/Cargo.toml -p opendart --no-default-features
cargo +1.97.1 test --locked --offline --manifest-path sdk/rust/Cargo.toml -p opendart-cli --no-default-features
RUSTDOCFLAGS="-D warnings" cargo +1.97.1 doc --locked --offline --manifest-path sdk/rust/Cargo.toml --workspace --all-features --no-deps
RUSTFLAGS="--cfg opendart_compat" cargo +1.97.1 test --locked --offline --manifest-path sdk/rust/compat/reqwest-feature-unification/Cargo.toml
cargo +1.97.1 tree --locked --offline --manifest-path sdk/rust/Cargo.toml -p opendart --no-default-features -e normal --prefix none
```

Run the MSRV boundary independently:

```sh
cargo +1.85.0 check --locked --offline --manifest-path sdk/rust/Cargo.toml --workspace --all-targets --all-features
cargo +1.85.0 check --locked --offline --manifest-path sdk/rust/Cargo.toml -p opendart --no-default-features
cargo +1.85.0 check --locked --offline --manifest-path sdk/rust/Cargo.toml -p opendart-cli --all-targets --no-default-features
cargo +1.85.0 metadata --locked --offline --manifest-path sdk/rust/Cargo.toml --no-deps
```

Verify both reviewed package inventories exactly, then dry-run the dependent
workspace packages together:

```sh
sdk_package_files="$(mktemp)"
cli_package_files="$(mktemp)"
trap 'rm -f "${sdk_package_files}" "${cli_package_files}"' EXIT
cargo +1.97.1 package --locked --offline --manifest-path sdk/rust/crates/opendart/Cargo.toml --list > "${sdk_package_files}"
diff -u sdk/rust/package-files.txt "${sdk_package_files}"
cargo +1.97.1 package --locked --offline --manifest-path sdk/rust/crates/opendart-cli/Cargo.toml --list > "${cli_package_files}"
diff -u sdk/rust/opendart-cli-package-files.txt "${cli_package_files}"
cargo +1.97.1 package --workspace --locked --offline --manifest-path sdk/rust/Cargo.toml
```

The CLI live smoke test is skipped unless `OPENDART_LIVE_TESTS=1` and
`OPENDART_API_KEY` are both present. It performs only the reviewed read-only
structured and binary calls and keeps assertions structural.

The no-default-features normal dependency graph must not contain `reqwest`,
Tokio, Hyper, TLS, proxy, DNS, or streaming-runtime dependencies. The default
`client-reqwest` feature is native-target-only; the transport-independent core
remains the portable public surface.

The package contains Cargo's `.cargo_vcs_info.json` for exact source revision
and exposes `source_provenance()` for the crate version, semantic specification
source release, independently selected canonical bundle checksum, generator
schema, and SDK projection checksum.
