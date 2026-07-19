# opendart

`opendart` is the first-party Rust protocol SDK for the OpenDART API. Its core
prepares deterministic requests and preserves source response evidence without
performing network I/O or imposing retry, collection, persistence, or domain
policy.

This repository is building the crate in ordered, reviewed slices. The current
generated surface exposes the complete canonical operation inventory through a
transport-independent contract. Its optional HTTP client executes those
prepared requests through one bounded, no-retry, no-redirect adapter.

```rust
use opendart::{ApiKey, Representation, operations::Company};

let operation = Company::new("00126380");
let prepared = operation.prepare(Representation::Json)?;
assert_eq!(prepared.relative_path(), "/api/company.json");

let key = ApiKey::new("example-key")?;
let authorized = prepared.authorize(&key);

// A strict caller exposes the authenticated relative URI only inside its own
// one-shot adapter boundary.
authorized.with_exposed_relative_uri(|relative_uri| {
    assert!(relative_uri.starts_with("/api/company.json?"));
});

# Ok::<(), Box<dyn std::error::Error>>(())
```

With the default `client-reqwest` feature, the same request can cross the
safe-default adapter without exposing a credential-bearing URL:

```no_run
# #[cfg(feature = "client-reqwest")]
use std::time::Duration;
# #[cfg(feature = "client-reqwest")]
use opendart::{ApiKey, Client, Representation, operations::Company};

# #[cfg(feature = "client-reqwest")]
# async fn example() -> Result<(), Box<dyn std::error::Error>> {
let prepared = Company::new("00126380").prepare(Representation::Json)?;
let client = Client::builder(ApiKey::new("example-key")?)
    .connect_timeout(Duration::from_secs(5))
    .read_timeout(Duration::from_secs(20))
    .total_timeout(Duration::from_secs(30))
    .build()?;
let response = client.execute(&prepared).await?;
assert_eq!(response.metadata.status(), 200);
# Ok(())
# }
```

ZIP operations use `Client::execute_binary` and return a fallible replaying
stream. Disable default features to use only request construction,
authorization, operation identity, and bounded wire inspection without an HTTP
client or async runtime in the normal dependency graph.
