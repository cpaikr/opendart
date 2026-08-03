# Reviewed Rust and CLI product names

The six `dsNNN.toml` manifests are the approved product-language boundary for
the Rust SDK and CLI. They bind canonical OpenAPI physical and logical
identities to deliberate Rust module, input, preparation, response, command,
alias, field, and flag names.

These files do not own wire behavior. Paths, methods, requiredness,
serialization, authentication, schemas, statuses, and media types remain
authoritative only in `openapi/openapi.yaml` and its references. The manifests
therefore record each OpenAPI parameter name solely to bind a source concept to
its reviewed Rust and CLI names.

The approved implementation convention is:

- each logical product has one input type; all required values are constructor
  arguments and optional values use consuming `with_*` builders;
- a reusable validated value is introduced only when OpenAPI proves the same
  invariant recurs across operations; operation-specific and cross-field rules
  remain in `prepare_*`;
- every structured physical representation has its own opaque response wrapper
  with `source`, `status`, `message`, `items`, and `field` views; serialization
  delegates to the one retained complete `SourceValue`;
- each structured logical operation records one root response view and one
  semantic Rust item-view type for every nested array-object path. Its
  `accessors` map binds each documented scalar or opaque source field to a
  reviewed typed method name once for the equivalent JSON/XML schemas;
- ZIP operations return the existing separate binary request/stream lifecycle;
  and
- preparation failures retain the stable `MissingInput`, `InvalidCardinality`,
  `InvalidLength`, `InvalidFormat`, `InvalidAllowedValue`, and
  `InvalidDecimalRange` categories. Interpretation keeps HTTP, envelope,
  source-status, XML-root, and path-aware shape failures distinct without
  retaining credentials or rejected bodies in diagnostics.

`internal/rustconformance` loads OpenAPI and all six manifests together. It
rejects missing or orphan identities, duplicate names in the applicable Rust
or CLI namespace, inconsistent generic or schema-backed accessors, invalid
representation names, unmapped source parameters, and logical-ID alias drift.
It also proves that a shared logical response mapping has equivalent normalized
JSON/XML schemas and fails closed if an OpenAPI response adds direct object
properties, non-object array items, or multiple array children to one view;
those shapes require a more explicit reviewed interface. The accepted
inventory is 85 logical operations, 167 physical operations, 171 structured
response views, and 1,900 schema-backed accessor mappings.

Run the complete credential-free gate from the repository root:

```sh
./scripts/verify rust-conformance
```
