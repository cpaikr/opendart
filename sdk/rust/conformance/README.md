# Rust-native conformance contract

This directory owns the independent coverage contract for the handwritten
Rust OpenDART conformer.

- `obligations.toml` gives each of the 167 canonical physical operations one
  reviewed obligation set. Structured JSON/XML operations require preparation,
  response binding, and decoding; ZIP operations require preparation,
  exact-byte streaming, and XML alternate-status handling.
- `cutover-guards.toml` records four active handwritten-cutover checks. The
  checker rejects partial activation and enforces no generated SDK exports or directory, no
  generator provenance, no public implementation selector, and no second
  structured execution result.
- The private tests in `crates/opendart/src/conformance.rs` execute the
  representative JSON, XML, and ZIP cases through the same request,
  interpretation, and streaming seams used by the public SDK. Deterministic
  faulty adapters prove rejection of path, parameter, encoding, validation,
  response-binding, media, XML-root, shape, and source-retention errors.
- The six interface manifests independently review every root and nested
  array-item response view and every schema-backed accessor name. The checker
  permits one logical mapping for JSON/XML only after their normalized schema
  graphs agree and rejects unsupported container shapes before implementation
  can scale from an ambiguous interface.

OpenAPI supplies inventory and wire authority, but it does not generate these
expectations or expected results. The retained fictional bodies under
`openapi/fixtures/v1` supply independent cross-boundary evidence with explicit
provenance, SHA-256 digest, and byte size.

Run the combined offline gate from the repository root:

```sh
./scripts/verify rust-conformance
```

The handwritten SDK is the only product conformer. Its conformance harness and
cutover guards remain private and introduce no implementation selector or
compatibility surface.
