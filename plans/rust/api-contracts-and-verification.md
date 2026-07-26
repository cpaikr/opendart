# Finish Rust API contracts and verification coverage

## Outcome

Bound generated list inputs before collection, document every public fallible
Rust API at the point of use, and make the README's advertised offline gate
match the CI workflow exactly.

## Current state

- Generated bounded string-array inputs retain at most the canonical maximum
  plus one sentinel. Valid inputs preserve serialization; oversized and
  infinite iterators return control and fail preparation with exact existing
  cardinality metadata.
- Every generated `prepare_*` method has operation-specific `# Errors`
  documentation derived from validation facts. The handwritten fallible API
  inventory is documented, and `clippy::missing_errors_doc` is enforced by the
  warnings-denied Clippy gate.
- The Rust README contains the exact structured and binary
  `opendart_compat` loopback commands, explains the repository-only cfg, and
  states that both commands are offline and credential-free.

## Design decisions

- Preserve the existing fluent `IntoIterator` builder surface. Collect at most
  the canonical maximum plus one sentinel element, then let the existing
  preparation check return `InvalidCardinality`.
- Document that invalid oversized builder state retains only enough elements to
  prove overflow. Do not promise full getter fidelity for input that cannot
  produce a valid request.
- Keep the maximum generator-owned. Do not hard-code endpoint-specific bounds
  in handwritten Rust.
- Generate operation-specific `# Errors` text from the same requiredness,
  cardinality, format, range, and representation facts used by preparation.
- Document public errors and panics, not implementation mechanics. Avoid
  boilerplate that merely says a method returns an error.
- Treat the CI workflow as execution truth and the README as its runnable local
  mirror. Commands that rely on `cfg(opendart_compat)` must show that cfg
  explicitly.

## Implementation plan

### Bound generated iterator consumption

- Update `internal/sdkgen/rust/render.go` so every bounded string-array setter
  consumes no more than `maximum + 1` items.
- Use straightforward iterator control such as `take(maximum + 1).collect()`;
  keep overflow handling visible and avoid a new public collection type.
- Compute the sentinel bound with checked arithmetic in generator validation.
  Reject an unrepresentable maximum rather than emitting wrapping Rust.
- Keep unbounded array parameters unchanged unless the canonical model gives
  them a limit.
- Update generated method docs to state bounded consumption and invalid
  oversized retention behavior.
- Regenerate the owned SDK tree and verify no unrelated generated drift.

### Prove the iterator boundary

- Add a counting iterator fixture and assert a valid input consumes exactly its
  available elements.
- Pass more than the maximum and assert consumption stops after the sentinel,
  the getter exposes only the bounded retained prefix, and every `prepare_*`
  representation returns the existing `InvalidCardinality` details.
- Add an infinite iterator fixture and prove the builder returns after bounded
  consumption.
- Exercise every generated parameter that has a canonical maximum through
  generator-driven or representative-plus-generator-validation coverage so a
  newly bounded list cannot omit the rule.
- Confirm exact valid maximum inputs retain their current query serialization.

### Add useful public `# Errors` documentation

- Inventory public `Result`-returning functions across manual, generated, and
  feature-gated APIs.
- Add concise manual sections for credential validation, client construction
  and execution, source-number construction, wire inspection, and any public
  builder finalization path found by the inventory.
- Extend the Rust renderer so each `prepare_*` method describes the actual
  `PrepareError` conditions supported by that operation and representation.
- Share small renderer helpers for common prose only where the same contract
  facts apply. Do not create a documentation template language.
- Add or extend generator golden tests to assert representative required,
  optional-constrained, bounded-list, and representation-specific error docs.
- Run rustdoc with warnings denied for all features and inspect representative
  generated docs for readable output.

### Synchronize the advertised verification gate

- Add the exact structured and binary `RUSTFLAGS="--cfg opendart_compat"`
  loopback commands to `sdk/rust/README.md` in the same relative order as CI.
- State why the cfg is required and that the tests remain credential-free.
- Compare the remainder of the README command block with
  `.github/workflows/verify.yml`; correct only real execution omissions and
  avoid copying CI setup boilerplate into user documentation.
- If command drift recurs, add a lightweight repository verification check over
  required semantic commands rather than duplicating the whole workflow as a
  brittle text snapshot.

## Validation

- Generator/model tests and generated-source freshness pass.
- Public contract tests cover exact-maximum serialization, finite counting,
  maximum-plus-one retained state, infinite iterators, both bounded generated
  parameters, and every representation's exact cardinality error.
- Warnings-denied all-feature Clippy and rustdoc pass.
- Both exact README compatibility loopback commands pass offline without
  credentials.
- The complete pinned stable, MSRV, feature, package, and clean-install gate is
  pending after independent review.

## Completion criteria

- No canonically bounded generated iterator can consume or retain more than the
  maximum plus one element.
- Oversized and infinite inputs return control and produce the established
  preparation error without changing valid request serialization.
- Every public fallible API explains its meaningful failure conditions,
  including generated operation-specific preparation failures.
- The README and CI agree on the credential-free compatibility commands, and
  all advertised gates pass.

## Next action

Run independent code review, address any actionable findings, then execute the
complete credential-free pre-push gate and deliver the API-contract PR.
