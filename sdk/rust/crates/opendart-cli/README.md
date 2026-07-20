# opendart-cli

`opendart-cli` is a machine-readable OpenDART command-line client backed by the
typed `opendart` Rust SDK. It exposes the SDK operation catalog without adding
retry, pagination, collection, persistence, or domain policy.

The CLI package is prepared for a future crates.io release but is not published
by the repository's current workflows. From a reviewed source checkout, install
the locked package reproducibly with:

```sh
cargo install --locked --path sdk/rust/crates/opendart-cli
```

## Keyless discovery

Discovery never reads an API key or contacts OpenDART:

```sh
opendart operations list
opendart operations describe company
```

Every discovery response is one JSON document on standard output. It contains
canonical operation names, logical-ID aliases, required flags, supported
representations, output classes, and execution controls. Invalid invocations
also return structured JSON and a stable nonzero exit code.

## Calls

Calls require `OPENDART_API_KEY` and execute exactly once through the typed SDK.
Structured JSON and XML responses both write one compact JSON envelope to
standard output.
Binary operations require an explicit destination and publish the complete
artifact without overwriting an existing path.

The CLI preserves OpenDART source-status evidence and uses stable exit classes;
it does not reinterpret source statuses as retry, empty-success, or pagination
policy. CLI-owned fields, metadata, diagnostics, and errors never include the
API key. Complete source payloads and binary artifacts are untrusted evidence
and deliberately remain unredacted.

Use `opendart operations describe <name-or-logical-id>` as the source of truth
for a call's generated flags. Exact SDK and JSON encoder pins in the packaged
lockfile are part of the reviewed CLI behavior; changes to those pins require
CLI compatibility review.
