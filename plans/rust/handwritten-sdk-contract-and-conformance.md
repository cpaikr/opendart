# Establish the handwritten SDK contract and Rust-native conformance gate

## Outcome

The repository has an accepted, testable contract for its only Rust OpenDART
conformer before handwritten operation implementation scales. OpenAPI remains
the sole wire authority, Rust and CLI product interfaces are intentionally
named, and an offline Rust-native gate can reject implementation errors without
treating the generated client or the conformer's construction path as an
oracle.

## Current state

- [ADR 0004](../../docs/decisions/0004-handwritten-rust-sdk-conformer.md)
  accepts the handwritten-only target and amends the generated-conformer parts
  of ADRs 0002 and 0003. The generated implementation remains the current
  product path until the single implementation-and-cutover plan completes.
- The generated SDK covers every physical operation and is deterministic, but
  its normalized model and renderer create correlated-failure risk.
  Reproducibility does not prove protocol conformance.
- Handwritten request, authorization, wire-inspection, response-evidence, and
  safe-client modules already establish the intended pure and secure seams.
- The independent fictional evidence corpus has two request cases and seven
  response cases. It proves important cross-boundary protocol families and is
  retained, but it is not an exhaustive parallel wire contract.
- No accepted SDK registry artifact, tag, or release constrains the replacement
  interface. The interrupted `0.1.0-beta.1` draft remains a release tombstone,
  not a compatibility source.

## Plan contract

This plan must finish the following design work before handwritten operations
are scaled across the inventory:

- OpenAPI owns operation inventory, paths, methods, parameter serialization,
  authentication shape, schemas, statuses, media types, and failure envelopes.
- Rust request preparation and response decoding are handwritten conforming
  implementations. Rust code may repeat wire facts but does not own them.
- OpenAPI-derived generation may produce bundles, operation inventories,
  coverage checks, Rust test scaffolding, and CLI interface projection. It must
  not produce SDK request builders, validators, wire types, or decoders.
- `PreparedRequest<T>` remains pure, immutable, credential-free, and usable by
  the safe client and caller-owned transports. No public operation or transport
  trait is introduced.
- JSON and XML remain distinct physical evidence. Provider statuses remain
  open source evidence rather than retry, successful-empty, or business policy.
  Binary requests retain a separate streaming lifecycle.
- Recurring proven input invariants may use project-owned validated values.
  Operation-specific and cross-field validation remains in preparation.
- Structured responses are representation-specific opaque wrappers. Each
  wrapper retains complete normalized `SourceValue`, exposes typed accessors,
  and serializes through the retained source representation rather than a
  second field model.
- Rust operation names and CLI command names are reviewed as product language.
  Use concise semantic English even when it diverges from abbreviated OpenDART
  stems. Rust and CLI names do not inherit generator spelling or need to match
  each other mechanically. Exact physical and logical OpenAPI identities remain
  available as protocol evidence; stable logical IDs remain CLI machine-call
  aliases, and compatibility aliases are not added for the unpublished
  generated interface.
- CLI discovery exposes CLI and source concepts, not Rust implementation names.
  Generated CLI typed dispatch may refer to Rust symbols privately.

## Work

### 1. Fix the public interface

- Review the complete logical operation inventory in DS001 through DS006
  batches. Approve each batch's idiomatic Rust module, input, method, response,
  and accessor names before that family is handwritten. Use DS001 to establish
  the semantic-English convention, then apply it consistently through DS006.
- Approve explicit CLI-owned command and flag names independently of Rust field
  spelling. Keep stable logical IDs as machine-call aliases for the reviewed
  command names rather than coupling those names to Rust symbols.
- Define required constructors, optional builders, validated reusable values,
  representation selection, wrapper granularity, source-shaped serialization,
  and the sanitized error taxonomy.
- Record the approved batches in `sdk/rust/interface/ds001.toml` through
  `ds006.toml`. These manifests contain only canonical physical and logical
  identities, canonical OpenAPI references needed to bind source concepts, and
  reviewed Rust and CLI product names. They do not copy paths, parameter or
  schema definitions, statuses, media types, or other wire facts.
- After all six batches are approved, run one global consistency gate derived
  from OpenAPI. It must prove complete physical and logical identity coverage,
  one mapping per logical product interface, unique names within each Rust and
  CLI namespace, valid representation-specific names, and the absence of orphan
  mappings before operation implementation scales. JSON and XML physical
  entries may share their approved logical product interface.

### 2. Define Rust-native conformance

- Give every canonical physical operation exactly one stable OpenAPI identity
  in a complete machine-checked case-obligation inventory. Every operation has
  a preparation obligation; structured operations also have response-binding
  and decoder obligations, while binary operations have streaming and
  alternate-status obligations.
- In this plan, make one representative JSON/XML operation and the ZIP operation
  executable through the Rust-native gate. The implementation-and-cutover plan
  makes every remaining operation-specific obligation executable.
- For executable cases, test public preparation output directly: method,
  relative path, deterministic query serialization, authentication requirement,
  representation, response binding, and stable validation category.
- Test representative wrapper construction and source-shaped serialization
  through the public interpretation seam. Cover path-aware decode failures
  without projecting a language-neutral response model.
- Generate inventories or test skeletons from OpenAPI only to prove coverage.
  Keep expectations independently reviewed and Rust-owned.

### 3. Retain proportionate independent evidence

- Keep the existing fictional provider evidence for JSON, XML, ZIP,
  provider-status, HTTP-status, malformed, wrong-root, unknown-field,
  unknown-status, and credential-reflection boundaries.
- Add independently authored cases only where a protocol family, decoder
  branch, or safety boundary lacks evidence. Do not create one fixture per
  operation merely to mirror OpenAPI.
- Record provenance, digest, and bounded size for retained bodies. Do not
  generate expected results from OpenAPI or capture unapproved provider
  responses.
- If another language conformer is actually introduced, make a separate
  decision about extracting a shared projection. Do not prebuild it now.

### 4. Prove the gate can fail

- Add deterministic faulty conformer or test adapters that cross the public
  preparation and interpretation seams. Cover paths, parameter names and
  encoding, requiredness, allowed values, response field shapes, media routing,
  XML roots, source retention, and stable failure categories.
- Require each faulty behavior to fail the intended Rust-native expectation or
  inventory check for the intended reason. Corrupting an expectation may
  smoke-test the harness, but it does not satisfy mutation acceptance.
- Treat comparisons with the generated client as diagnostic evidence only.
  OpenAPI and independently authored evidence resolve disagreements; generated
  behavior never gates acceptance.
- Keep OpenAPI validation, Rust conformance, real transport verification, and
  bounded live conformance as separate layers so none certifies itself.

### 5. Integrate the offline gate

- Add one credential-free repository verification entry point for operation
  coverage, Rust-native cases, retained fictional evidence, and mutation
  controls.
- Prove the gate using the deterministic faulty adapters, then require only the
  handwritten conformer to pass it. Do not add a general mutation-testing
  dependency.
- Define the compile-time or public-contract checks that will forbid generated
  SDK exports, generator provenance, a public generated/handwritten selector,
  and a second structured execution result. Activate them with the cutover so
  this contract plan does not make the current implementation fail.

## Cross-scope live-probe handoff

The dev-owned `probe-multi-company` and `probe-auditor-evidence` commands are
observational tools, not evidence for this offline Rust-native gate or
prerequisites for the handwritten SDK cutover. Their existing request bounds,
response validation, and sanitization remain useful, but both currently read
`OPENDART_API_KEY` before proving their complete execution envelope offline.

Before either focused probe is run again, scheduled, or promoted into
release-gating qualification evidence, the dev scope must first identify which
of its observations remain unique beside the general live-conformance runner.
Remove duplicated observations from the focused probes in favor of that
runner. Give every retained focused probe a credential-free
`Preflight -> Plan -> Run` boundary that validates trusted routing, request
encoding, assertions, an absolute attempt ceiling, and the report contract
before reading the credential or constructing a credential-bearing request.
This Rust plan neither changes nor certifies those Go commands.

## Non-goals

- Do not implement the complete handwritten operation inventory in this plan.
- Do not redesign application retry, quota, collection, persistence, or
  business policy.
- Do not publish a crate, alter release authority, resolve credentials, or make
  an OpenDART request.
- Do not build an exhaustive cross-language fixture or projection harness.
- Do not make the current generated client pass the new gate as a completion
  condition.

## Validation

- The accepted ADR and interface mapping have no contradiction with OpenAPI
  authority, source-evidence semantics, or the credential boundary.
- Every canonical physical operation has one inventory identity, a preparation
  obligation, and every applicable structured-decoder or binary-lifecycle
  obligation. Operations may have multiple cases where rules or branches
  require them.
- The DS001 through DS006 manifests pass the OpenAPI-derived global coverage,
  namespace-aware uniqueness, representation, alias, and orphan checks.
  Representative JSON, XML, and ZIP cases execute through the public seams;
  full per-operation execution remains an entry condition of the
  implementation-and-cutover plan.
- Retained fixture provenance, digests, sizes, and schemas validate offline.
- Each faulty adapter is observed failing for its intended reason; an
  expectation-only failure is not accepted as mutation evidence.
- Existing Go, Rust SDK, CLI, package, and release-guard verification remains
  green and credential-free.

## Completion criteria

- Exact Rust and CLI product names have been approved in six DS-family batches,
  and the complete physical/logical identity mapping passes one global
  consistency gate before operation implementation scales.
- The complete case-obligation inventory is machine-checked, and representative
  JSON, XML, and ZIP cases prove Rust-native conformance can accept or reject
  preparation and decoding without generator output or a second wire model.
- The retained independent corpus covers cross-boundary protocol families, and
  the gate fails on protected semantic mutations.
- Current public behavior and publication state remain unchanged.

## Next action

Define the name-only interface manifest format, author and approve the DS001
batch, repeat through DS006, and pass the global consistency gate. Then complete
the case-obligation inventory and executable representative JSON, XML, and ZIP
cases with deterministic faulty-adapter controls before scaling implementation.
