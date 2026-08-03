# ADR 0004: Use one handwritten Rust SDK conformer

- Status: accepted; implementation pending
- Date: 2026-08-03
- Amends: [ADR 0002](0002-public-rust-sdk.md) and
  [ADR 0003](0003-agent-first-opendart-cli.md)

## Context

ADR 0002 selected a repository-owned normalized model and generated Rust
request and response code. That implementation is reproducible and complete,
but the generator, generated validators, generated wire types, and generated
decoders form the conformer itself. Correlated model and renderer mistakes can
therefore survive freshness checks, and the public Rust interface inherits
generator-shaped names rather than deliberate product language.

The repository has no verified published `opendart` crate, SDK tag, or accepted
SDK release. The interrupted `0.1.0-beta.1` draft stopped before registry
publication and remains a fail-closed recovery tombstone. There is no supported
external Rust compatibility contract that requires the generated interface or
a transition layer.

Rust is the only planned SDK conformer. A versioned cross-language request and
response projection would introduce another contract-shaped artifact without a
second consumer. The existing small fictional evidence corpus is useful at
cross-boundary protocol seams, but an exhaustive shared fixture model would
compete with OpenAPI as wire authority.

The CLI currently derives command names and flags from Rust SDK identifiers and
publishes fields such as `sdk_field` and `response_type` in discovery. Those
details couple CLI compatibility to Rust refactoring even though the CLI should
own its presentation language.

## Decision

### One conformer and one wire authority

OpenAPI remains the sole authority for operation inventory, paths, methods,
parameters, authentication shape, schemas, statuses, media types, and provider
failure envelopes. The public Rust SDK will contain one handwritten conformer
for that contract.

The repository will remove generated SDK request builders, validators, wire
types, response decoders, and their renderer after cutover. It will not retain
a public or private permanent generated/handwritten selector, differential
conformer, compatibility alias layer, runtime OpenAPI parser, schema manifest,
build script, or proc-macro replacement.

`PreparedRequest<T>` remains the pure, immutable, credential-free execution
boundary. The safe native client and caller-owned transports consume it. No
public operation registry, operation trait, or transport trait is introduced.

### Deliberate Rust interface

Rust modules, inputs, methods, wrappers, and accessors use reviewed idiomatic
product names rather than preserving generator spelling. Required values are
constructor inputs; optional values use consuming builders or equally explicit
project-owned setters. Reusable validated values exist only for recurring
invariants proven identical across operations.

Every operation and representation retains its exact physical and logical
OpenAPI identity as protocol evidence. Those identities do not force Rust names
and do not justify aliases for the unpublished generated interface.

JSON and XML remain distinct physical representations. Binary operations keep
their separate streaming and replay lifecycle. Provider statuses and future
values remain open source evidence, not SDK retry, successful-empty, quota,
collection, persistence, or domain policy.

### Source-complete response wrappers

Every structured physical representation has an opaque project-owned response
wrapper. Successful construction validates established required structure and
retains the complete normalized `SourceValue`, including unknown fields, exact
source numbers, and future provider values. Typed accessors are views over that
evidence and do not create a second authoritative response model.

Under the optional SDK JSON feature, wrappers serialize by delegating to their
retained source representation. They do not derive serialization from typed
accessor fields. Once all structured wrappers expose complete evidence, the
parallel `Client::execute_raw` path is removed so there is one structured
execution result.

### Rust-native conformance

The complete physical operation inventory is derived from OpenAPI for coverage,
but preparation and decoding expectations are reviewed Rust-native tests. Each
physical operation has one canonical identity and as many cases as its
parameter rules and decoder branches require.

The existing bounded fictional evidence corpus remains for protocol families
such as JSON, XML, ZIP, provider status, HTTP status, malformed input, unknown
fields, and credential reflection. New independent bodies are added only where
a cross-boundary branch lacks evidence. Expected outcomes are not generated
from OpenAPI or captured from unapproved live responses.

Mutation controls must prove that the gate detects incorrect paths,
serialization, requiredness, validation, media routing, XML roots, response
shapes, source retention, and failure categories. The generated client may be
used temporarily to investigate a disagreement, but it is never an acceptance
oracle. OpenAPI and independently authored evidence resolve disagreements.

A shared language-neutral projection will be considered only if a second real
language conformer creates a concrete need.

### CLI presentation boundary

The CLI owns explicit command names, flags, descriptions, discovery fields,
process envelopes, exit codes, and binary artifact policy. Public command names
are not mechanically derived from Rust identifiers. Stable logical IDs remain
machine-call aliases for those reviewed command names. Discovery retains
logical and physical OpenAPI identities, source parameter concepts,
constraints, and coarse response shape, but removes Rust implementation fields
such as `sdk_field` and `response_type`.

An OpenAPI-derived CLI interface projection may continue to generate the
command catalog, parser breadth, discovery, and exhaustive private typed call
wiring. That projection retains its own checksum and freshness verification.
It must not own provider parameter validation, request serialization, response
decoding, or response schemas. Rust symbol names may occur only inside private
dispatch wiring.

Structured CLI output serializes the SDK wrapper's retained source evidence
inside the CLI-owned process envelope. The CLI does not re-decode response
bodies or maintain endpoint-specific response mirrors.

### Provenance and migration

Runtime SDK provenance retains the crate version, applicable specification
source release, and canonical OpenAPI bundle checksum. Generator schema and SDK
projection identity are removed. Exact Git revision remains Cargo package and
release evidence through `.cargo_vcs_info.json`; no runtime Git-revision or
undefined contract-profile field is introduced.

The generated implementation may remain the sole public path while the
handwritten implementation is developed privately. Implementation, SDK export
switch, CLI adaptation, SDK generator deletion, active-documentation changes,
and offline package qualification form one product-state transition. The work
may use multiple reviewable commits, but a dual-conformer state is not a
completed or supported milestone.

Until that transition lands, current SDK and CLI architecture, contract,
generation, and README pages continue to describe the generated implementation
and link to this ADR as the accepted target. They change to handwritten-only
language atomically with the implementation cutover.

ADR 0002's first-party SDK product, pure request boundary, safe native client,
security, compatibility, packaging, and independent release decisions remain.
ADR 0003's agent-first CLI, compact JSON process contract, binary artifact
handling, source-first status behavior, independent release component, and SDK-
before-CLI publication order remain.

## Consequences

- OpenAPI is the only complete wire authority, and the final Rust product has
  one handwritten conformance path.
- Rust and CLI compatibility reflect reviewed product language instead of
  incidental generator layout.
- Complete source evidence and convenient typed access coexist without a raw
  parallel structured API or lossy CLI conversion.
- Conformance breadth comes from OpenAPI-derived inventory plus Rust-native
  cases, independent fictional evidence, mutations, real transport tests, and
  bounded live evidence rather than one all-purpose projection.
- CLI generation remains valuable for exhaustive presentation and integration
  wiring but cannot become another HTTP conformer.
- Removing the generated public interface is intentionally breaking, but no
  published SDK contract requires aliases or a staged compatibility mode.

## Alternatives considered

- Retaining the generated conformer permanently would preserve correlated
  failures and violate the one-handwritten-client boundary.
- Keeping both conformers behind a feature or runtime selector would double the
  supported behavior and make later removal a compatibility event.
- Building a shared language-neutral conformance projection now would create
  durable machinery without a second language consumer.
- Preserving generated names or aliases would make accidental generator output
  part of the first public compatibility contract.
- Moving response conversion into the CLI would duplicate SDK schemas and make
  CLI releases responsible for provider decoding.

## Delivery

- [Contract and Rust-native conformance
  plan](../../plans/rust/handwritten-sdk-contract-and-conformance.md)
- [Handwritten implementation and cutover
  plan](../../plans/rust/handwritten-sdk-conformer.md)
- [Public Rust SDK delivery plan](../../plans/rust/public-rust-sdk.md)
- [Public CLI delivery plan](../../plans/rust/public-opendart-cli.md)
