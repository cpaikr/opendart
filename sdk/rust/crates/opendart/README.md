# opendart

`opendart` is a first-party Rust protocol SDK for the OpenDART API. It prepares
deterministic requests and preserves source-response evidence without taking
ownership of retry, quota, collection, persistence, or domain policy.

Current crates.io availability is tracked in the
[Public Rust SDK task](https://github.com/cpaikr/opendart/blob/main/tasks/rust/public-rust-sdk.md). Use a registry
dependency only after the intended version and its public artifacts have been
verified. The dependency examples below describe the stable `0.1` line.

This README documents the current generated implementation. The accepted
handwritten-only replacement is recorded in
[ADR 0004](../../../../docs/decisions/0004-handwritten-rust-sdk-conformer.md)
and does not become the public crate contract until its implementation cutover.

The checked-in `operations` module is generator-owned but supported public API.
Operation names and behavior participate in SemVer; generated file layout does
not.

## Ordinary native client

Default features include the native `client-reqwest` adapter:

```toml
[dependencies]
opendart = "0.1"
```

```no_run
use std::time::Duration;

# #[cfg(feature = "client-reqwest")]
use opendart::{operations::Company, ApiKey, Client, SourceReply};

# #[cfg(feature = "client-reqwest")]
# async fn example() -> Result<(), Box<dyn std::error::Error>> {
let request = Company::new("00126380").prepare_json()?;
let client = Client::builder(ApiKey::new("example-key")?)
    .connect_timeout(Duration::from_secs(5))
    .read_timeout(Duration::from_secs(20))
    .total_timeout(Duration::from_secs(30))
    .build()?;

match client.execute(&request).await?.reply {
    SourceReply::Success(company) => {
        println!("company name evidence: {:?}", company.corp_name)
    }
    SourceReply::Status(status) => println!("OpenDART status: {}", status.code),
    _ => unreachable!("future reply variants remain representable"),
}
# Ok(())
# }
```

Preparation performs no I/O. It validates required and non-empty inputs plus
canonical string lengths, supported formats, allowed values, decimal ranges,
list cardinality, and query serialization. Rules that exist only in narrative
prose are not inferred.

Each `prepare_json` or `prepare_xml` call binds its
representation-specific generated success type. `Client::execute` returns
recognized status-only envelopes—including `000`, `013`, documented errors,
and unknown future strings—as `SourceReply::Status`. The SDK does not classify
`013` as successful empty data or mark any source status retryable.

Use `Client::execute_raw(&request)` when you need the complete normalized
`SourceValue` success envelope instead of the generated success type.

XML evidence becomes authoritative only after complete bounded UTF-8 XML 1.0
validation. DTDs, custom entities, and external resolution are rejected. Valid
declarations, comments, processing instructions, namespaces, mixed content,
CDATA, and character references remain supported within the SDK's depth and
per-element attribute bounds. Literal XML line endings normalize according to
XML 1.0, while numeric character references retain their referenced character.

ZIP operations use `Client::execute_binary`. The result distinguishes a
supported positive ZIP signature, a completely validated bounded XML status
envelope, and an unrecognized replaying byte stream. Classification never
discards inspected prefix bytes.

On WebAssembly, `client-reqwest` is accepted but inert: native `Client` types
and their transport/runtime dependencies are absent. Prepared requests and wire
inspection remain available for a caller-owned WebAssembly adapter.

## Optional JSON serialization

Enable `serde-json` when a typed consumer needs to encode complete response
evidence directly with `serde_json`:

```toml
[dependencies]
opendart = { version = "0.1", features = ["serde-json"] }
serde_json = "1"
```

Generated response objects and shared response, status, metadata, and
`SourceValue` types then implement `serde::Serialize`. Source numbers are
validated on construction, and direct `serde_json` text encoding preserves
their exact lexemes, including arbitrary-size integers, decimals, and
exponents. Passing them through `serde_json::Value` or another numeric model
first is outside that guarantee.

Credentials, prepared and authorized requests, clients, and body streams remain
non-serializable.

## Caller-owned transport

Disable default features when the application owns HTTP execution:

```toml
[dependencies]
opendart = { version = "0.1", default-features = false }
```

```rust
use opendart::{operations::Company, ApiKey};

let request = Company::new("00126380").prepare_xml()?;
assert_eq!(request.relative_path(), "/api/company.xml");

let key = ApiKey::new("example-key")?;
let authorized = request.authorize(&key);
authorized.with_exposed_relative_uri(|relative_uri| {
    // Pass the URI directly into a caller-owned one-shot adapter. It contains
    // the credential and must not be logged, persisted, or returned in errors.
    assert!(relative_uri.starts_with("/api/company.xml?"));
});

# Ok::<(), Box<dyn std::error::Error>>(())
```

After a bounded caller-owned read, the prepared request interprets the complete
HTTP outcome through the same contract as the official client:

```rust
use opendart::{operations::Company, SourceReply, WireInspector};

let inspector = WireInspector::new(64 * 1024).expect("nonzero limit");
let api_key = opendart::ApiKey::new("example-key")?;
let prepared = Company::new("00126380").prepare_json()?;
let reply = prepared.interpret_response(
    &inspector,
    &api_key,
    200,
    br#"{"status":"013","message":"no data"}"#,
)?;
assert!(matches!(reply, SourceReply::Status(_)));

# Ok::<(), Box<dyn std::error::Error>>(())
```

Every non-2xx status returns `ResponseInterpretError::HttpStatus`, with
normalized bounded body evidence when it can be recognized safely and does not
contain the active credential or its encoded forms. A
success-shaped body under HTTP 500 is never decoded into the generated success
type. Callers remain responsible for bounding body collection before passing
the slice to this defensive interpreter.

The official client never retries, follows redirects, uses ambient proxies, or
automatically decodes response content. Callers requiring different transport
policy use prepared requests rather than injecting an arbitrary client.

`source_provenance()` identifies the selected semantic specification source,
canonical bundle checksum, generator schema, and SDK projection checksum.
Packaged archives also contain Cargo's `.cargo_vcs_info.json` for the exact Git
revision.

The complete supported behavior is documented in the
[public contract](https://github.com/cpaikr/opendart/blob/main/docs/rust-sdk/public-contract.md) and
[transport and safety](https://github.com/cpaikr/opendart/blob/main/docs/rust-sdk/transport-and-safety.md)
guides.
