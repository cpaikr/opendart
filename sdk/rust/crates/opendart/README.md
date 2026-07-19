# opendart

`opendart` is the first-party Rust protocol SDK for the OpenDART API. Its core
prepares deterministic requests and preserves source response evidence without
performing network I/O or imposing retry, collection, persistence, or domain
policy.

This repository is building the crate in ordered, reviewed slices. The current
generated surface exposes the complete canonical operation inventory through a
transport-independent contract; the optional HTTP client is added separately.

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

The default `client-reqwest` feature is reserved for the safe-default
convenience client. Disable default features to use only request construction,
authorization, operation identity, and wire evidence types.
