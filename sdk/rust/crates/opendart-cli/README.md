# opendart-cli

`opendart-cli` is a machine-readable OpenDART command-line client backed by the
typed `opendart` Rust SDK. It exposes the SDK operation catalog without adding
retry, pagination, collection, persistence, or domain policy.

The CLI package is prepared for a future crates.io release but is not published
by the repository's current workflows. From a reviewed source checkout, install
the locked package reproducibly with:

```sh
cargo +1.97.1 install --locked --path sdk/rust/crates/opendart-cli
```

Confirm the installed binary and inspect the operation-specific help:

```sh
opendart --version
opendart --help
opendart call company --help
```

Use the top-level `--version` for the CLI package identity. Use
`opendart call <operation> --help` for concise call syntax; the machine-readable
operation description remains the complete source of truth.

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

With `OPENDART_API_KEY` present in the inherited environment, an installed CLI
can make a structured call directly:

```sh
opendart call company \
  --corp-code 00126380 \
  --representation json
```

Download the corporate-code archive to a new path with:

```sh
opendart call corp-code --output /tmp/corp-code.zip
```

Choose a different path or remove the prior artifact before repeating the
example; the CLI deliberately refuses to overwrite it.

The CLI preserves OpenDART source-status evidence and uses stable exit classes;
it does not reinterpret source statuses as retry, empty-success, or pagination
policy. CLI-owned fields, metadata, diagnostics, and errors never include the
API key. Complete source payloads and binary artifacts are untrusted evidence
and deliberately remain unredacted.

Use `opendart operations describe <name-or-logical-id>` as the source of truth
for a call's generated flags. Exact SDK and JSON encoder pins in the packaged
lockfile are part of the reviewed CLI behavior; changes to those pins require
CLI compatibility review.
