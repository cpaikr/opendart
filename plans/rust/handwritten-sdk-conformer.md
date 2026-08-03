# Implement and cut over the handwritten Rust SDK conformer

## Outcome

Every supported OpenDART operation has an idiomatic handwritten Rust request
and response implementation behind the existing pure protocol boundary. The
handwritten conformer is the only SDK product path, the CLI consumes it without
owning HTTP behavior, generated SDK source and machinery are absent, and the
packages pass complete offline qualification without publication.

Implementation may use private incremental phases, but this plan has one
product-state completion: the generated public conformer is replaced and
deleted. A repository state containing two conformers is never a separately
completed or supported milestone.

## Entry conditions

- The [contract and conformance plan](handwritten-sdk-contract-and-conformance.md)
  has approved the exact Rust and CLI product interface, physical/logical
  identity mapping, Rust-native coverage inventory, independent evidence, and
  mutation controls.
- The [CLI presentation projection plan](cli-presentation-projection.md) has
  implemented reviewed CLI-owned grammar and discovery while isolating current
  generated-SDK symbols inside private exhaustive dispatch.
- Current SDK contract, architecture, generation, layout, and README documents
  still describe the generated conformer. CLI documents describe the reviewed
  presentation projection backed by that current conformer. ADR 0004 describes
  the accepted target until this plan changes implementation and active SDK and
  integration documentation together.
- The CLI's private generated dispatch still constructs SDK types directly, so
  retargeting that adapter, switching SDK exports, and deleting SDK generation
  belong to this same plan.

## Final implementation shape

```text
opendart
├── values/                 reusable validated project-owned inputs
├── protocol/               private preparation and decode mechanics
├── operations/             handwritten inputs, wrappers, and conformers
├── request/                PreparedRequest and authorization capability
├── wire/                   bounded source inspection and evidence
└── client/                 safe native transport

opendart-cli
├── generated interface/    commands and discovery
├── generated adapter/      private exhaustive typed wiring
└── handwritten runtime/    process, credential, output, and artifact policy
```

Each SDK operation module colocates its logical input, physical preparation
methods, representation-specific wrappers, private decoder, and focused tests.
Shared helpers own only mechanics that are genuinely identical. There is no
Rust schema manifest or macro input that becomes a parallel wire authority.

The CLI interface projection owns grammar and discovery. Its separate private
dispatch adapter owns exhaustive typed wiring. Neither owns provider parameter
validation, request serialization, response decoding, or response schemas.

## Work

### 1. Deepen shared handwritten protocol modules

- Move repeated query construction, validation composition, response-object
  traversal, additional-field retention, source-shaped serialization, and
  error-path construction behind narrow private helpers.
- Preserve deterministic serialization, bounded iterator consumption, complete
  JSON/XML validation, exact source-number retention, open statuses, and
  credential-safe evidence.
- Add a private one-exchange HTTP seam only if both the production reqwest
  adapter and a scripted test adapter exercise real orchestration variation.
  Do not publish it or replace loopback transport tests with mocks.

### 2. Introduce validated input values

- Add project-owned types only for recurring invariants proven identical across
  operations, such as corporation codes, compact dates, business years, report
  codes, and bounded page values.
- Keep provider-evolvable output values open. Do not turn observed strings into
  closed result enums.
- Map construction failures to sanitized project-owned input errors without
  retaining rejected values.

### 3. Handwrite operation preparation

- Implement every DS001 through DS006 logical operation using the reviewed
  idiomatic Rust names while preserving exact physical and logical identities.
- Make required inputs constructor requirements and optional inputs consuming
  builders or equally explicit project-owned setters.
- Validate requiredness, length, format, allowed values, decimal ranges, list
  cardinality, cross-field rules, and serialization as established in OpenAPI.
- Bind each prepared request to its handwritten response contract without
  credentials or I/O.
- Complete one protocol family at a time on the implementation branch. Do not
  expose a public selector, alias, or compatibility path.

### 4. Handwrite response contracts

- Implement one opaque response wrapper for each supported structured physical
  representation, retaining the complete normalized source object.
- Expose typed accessors only for schema facts supported by OpenAPI; preserve
  uncertain scalars and additive fields through the source-evidence accessor.
- Decode established required structure eagerly so successful construction
  proves the wrapper invariant. Return path-aware sanitized failures.
- Implement the optional SDK JSON serialization contract by delegating wrapper
  serialization to retained `SourceValue`; do not derive a second field-shaped
  representation.
- Keep JSON and XML wrappers separate even where documented fields coincide.
  Preserve ZIP classification, replayed prefix bytes, and alternate XML status
  handling as a separate lifecycle.

### 5. Qualify the handwritten conformer

- Exercise every handwritten preparation and interpretation path through the
  Rust-native gate established by the preceding plan.
- Run retained fictional provider evidence and mutation controls against only
  the handwritten implementation.
- Use generated-client comparisons only to investigate disagreements. OpenAPI
  and independent evidence decide the result.
- Run Rust-specific property, adversarial, real loopback transport, feature,
  target, MSRV, WebAssembly, documentation, and package tests through the same
  public seams consumers use.

### 6. Retarget the private CLI runtime adapter

- Preserve the already-reviewed CLI grammar, discovery, aliases, catalog, and
  interface-projection checksum without deriving any public change from Rust
  symbols.
- Regenerate or rewrite only the exhaustive private dispatch adapter so every
  call uses handwritten SDK constructors, preparation, wrappers, and failure
  categories. Rust symbols remain confined to that adapter and its independent
  freshness identity.
- Serialize SDK wrappers through their source-shaped serialization contract.
  Do not re-decode bodies or maintain endpoint-specific CLI response mirrors.
- Preserve keyless discovery, exit codes, process envelopes, binary artifact
  transactions, credential boundaries, and dispatch completeness.

### 7. Apply one final product-state transition

- Switch SDK public exports and CLI consumption to the reviewed handwritten
  modules in one coherent integration change.
- Remove `Client::execute_raw` once every structured wrapper exposes complete
  normalized evidence and all consumers use the single typed execution path.
- Delete generated SDK operations, responses, mapping, ownership markers,
  `#[rustfmt::skip]` wiring, renderer, SDK-specific artifact model, projection
  checksums, freshness paths, generation-only fixtures, and generator-only
  tests. Retain the independent fictional provider evidence.
- Delete the remaining SDK-specific parts of `internal/sdkgen/model` so only
  the CLI interface and private dispatch adapter projections remain. Confirm
  that neither contains provider validation, serialization, or decoding.
- Remove runtime generator schema and SDK projection identity. Runtime
  provenance retains crate version, specification source release, and canonical
  bundle checksum; Cargo `.cargo_vcs_info.json` retains exact Git revision.
- Do not introduce `contract-profile`, a build script, proc macro, runtime
  OpenAPI parser, serialized Rust schema manifest, or network download.

This transition may be implemented as a reviewable commit series. “Atomic”
means the merged product state never exposes a selectable or supported
dual-conformer architecture.

### 8. Reconcile active documentation and release inputs

- Update `ARCHITECTURE.md`, SDK contracts, verification guides, repository
  layout, READMEs, package docs, ADR status notes, and any CLI integration
  references to describe the handwritten-only implementation.
- Remove or replace the active SDK generation guide. Retain only CLI interface
  projection guidance where it remains true.
- Package SDK and CLI from clean source. Prove neither archive contains SDK
  generator implementation or generated SDK source.
- Renew CLI inventory and dispatch completeness against canonical operation
  identities and prove that Rust symbol renames do not alter discovery.
- Run a cross-module code review and resolve blocking findings before declaring
  the plan complete.

## Temporary migration obligation

The generated conformer may remain the sole public path while implementation is
in progress on an unmerged branch. Handwritten modules may remain private and
test-only during that work. This coexistence exits within this plan: it cannot
be merged as an end state, published, selected at runtime, or used as a reason
to preserve generator-shaped compatibility.

## Non-goals

- Do not change application retry, quota, collection, persistence, or business
  policy.
- Do not publish SDK or CLI crates, create tags or releases, configure secrets,
  or trigger release workflows.
- Do not redesign CLI distribution or prebuilt artifacts.
- Do not introduce a public transport abstraction, registry, operation trait,
  generated compatibility aliases, or fallback dispatch.
- Do not use live OpenDART responses as the merge oracle.

## Validation

- Every canonical physical operation has one handwritten preparation path and
  the required Rust-native cases for its parameter rules and decoder branches.
- Every structured representation has an opaque handwritten wrapper with
  complete normalized evidence, typed accessors, and source-shaped
  serialization.
- The handwritten conformer is the only SDK preparation and decoding path
  reachable from the crate or CLI; `execute_raw` and public path selection are
  absent.
- Repository search finds no generated SDK ownership marker, renderer,
  projection checksum, SDK freshness command, runtime generator provenance, or
  active SDK generation guidance.
- CLI discovery exposes no Rust field or response-type symbols. Its retained
  interface projection is complete, its private dispatch adapter is exhaustive,
  and neither contains HTTP semantics.
- Full Go and Rust verification, formatting, linting, documentation, MSRV,
  WebAssembly, feature-unification, adversarial transport, CLI process,
  package-content, clean-install, provenance, and release-guard checks pass
  without credentials.
- Release guards still reject the interrupted beta.1 recovery path, and no new
  registry, tag, release, credential, or workflow state exists.

## Completion criteria

- Source, packages, tests, and active documentation expose one handwritten SDK
  conformer and no generated compatibility path.
- The CLI consumes the conformer as a presentation client and retains only its
  bounded interface and private dispatch adapter projections.
- Independent evidence, mutation controls, real transport safety, clean package
  installation, and release guards all pass.
- The [public SDK delivery plan](public-rust-sdk.md) becomes the
  next queued Rust item but still requires an explicit start and renewed
  external-state review before release action.

## Next action

After the contract and CLI presentation plans complete, deepen the shared
private helpers and implement one representative JSON/XML operation plus the
ZIP operation through the Rust-native gate. Continue within this plan through
full inventory coverage, private CLI dispatch retargeting, SDK generator
deletion, active-documentation cutover, and offline package qualification.
