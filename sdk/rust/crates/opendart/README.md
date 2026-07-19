# opendart

`opendart` is a first-party Rust protocol SDK for the OpenDART API. It prepares
deterministic requests and preserves source-response evidence without taking
ownership of retry, quota, collection, persistence, or domain policy.

The checked-in `operations` module is generator-owned but is supported public
API. Its operation names and behavior participate in SemVer; its file layout
does not. Other generated routing and wire metadata remain private.

## Ordinary client

Default features include the native `client-reqwest` adapter:

```toml
[dependencies]
opendart = "0.1"
```

```no_run
use std::time::Duration;

# #[cfg(feature = "client-reqwest")]
use opendart::{operations::Company, ApiKey, Client, Representation, SourceReply};

# #[cfg(feature = "client-reqwest")]
# async fn example() -> Result<(), Box<dyn std::error::Error>> {
let request = Company::new("00126380").prepare(Representation::Json)?;
let client = Client::builder(ApiKey::new("example-key")?)
    .connect_timeout(Duration::from_secs(5))
    .read_timeout(Duration::from_secs(20))
    .total_timeout(Duration::from_secs(30))
    .build()?;

match client.execute(&request).await?.reply {
    SourceReply::Success(value) => println!("{value:?}"),
    SourceReply::Status(status) => println!("OpenDART status: {}", status.code),
    _ => unreachable!("future reply variants remain representable"),
}
# Ok(())
# }
```

`Client::execute` handles bounded JSON or XML envelopes. It returns every
recognized status envelope—including `000`, `013`, documented error values,
and unknown future strings—as `SourceReply::Status`. The SDK does not decide
that `013` is an empty success and does not mark any source status retryable.

ZIP operations use `Client::execute_binary`. The result distinguishes a
positive ZIP signature, a bounded alternate XML status envelope, and an
unrecognized replaying byte stream without losing inspected prefix bytes.

## Advanced transport ownership

Disable default features when the application owns HTTP execution:

```toml
[dependencies]
opendart = { version = "0.1", default-features = false }
```

```rust
use opendart::{operations::Company, ApiKey, Representation};

let request = Company::new("00126380").prepare(Representation::Xml)?;
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

After the caller's bounded read, `WireInspector::inspect_json` and
`WireInspector::inspect_xml` classify source envelopes while retaining unknown
fields and scalar forms:

```rust
use opendart::{SourceReply, WireInspector};

let inspector = WireInspector::new(64 * 1024).expect("nonzero limit");
let reply = inspector.inspect_json(br#"{"status":"013","message":"no data"}"#)?;
assert!(matches!(reply, SourceReply::Status(_)));

# Ok::<(), Box<dyn std::error::Error>>(())
```

The SDK never retries, follows redirects, reads ambient proxies, or
automatically decodes response content. `source_provenance()` identifies the
reviewed specification and generator inputs; packaged archives also include
Cargo's `.cargo_vcs_info.json` for the exact source revision.
