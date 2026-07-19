# Rust SDK workspace

This isolated Cargo workspace contains the public `opendart` crate. Consumer
builds use only Cargo; the private Go generator and canonical OpenAPI inputs
remain repository tooling.

The repository pins stable Rust in `rust-toolchain.toml` and declares Rust
1.85.0 as the crate MSRV. Run the gates directly so CI and local verification
exercise the same Cargo contracts:

```sh
cargo fmt --manifest-path sdk/rust/Cargo.toml --all -- --check
cargo clippy --locked --manifest-path sdk/rust/Cargo.toml --workspace --all-targets --all-features -- -D warnings
cargo test --locked --manifest-path sdk/rust/Cargo.toml --workspace --all-features
cargo test --locked --manifest-path sdk/rust/Cargo.toml -p opendart --no-default-features
RUSTDOCFLAGS="-D warnings" cargo doc --locked --manifest-path sdk/rust/Cargo.toml --workspace --all-features --no-deps
cargo package --locked --manifest-path sdk/rust/crates/opendart/Cargo.toml
```

Run the MSRV boundary explicitly:

```sh
cargo +1.85.0 check --locked --manifest-path sdk/rust/Cargo.toml --workspace --all-targets --all-features
cargo +1.85.0 check --locked --manifest-path sdk/rust/Cargo.toml -p opendart --no-default-features
cargo +1.85.0 tree --locked --manifest-path sdk/rust/Cargo.toml -p opendart --no-default-features -e normal
```

The final dependency tree must not contain `reqwest`, Tokio, Hyper, TLS, proxy,
DNS, or streaming-runtime dependencies. The default `client-reqwest` feature is
native-target-only; the transport-independent core remains the supported
portable surface.
