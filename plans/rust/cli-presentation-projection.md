# Decouple the CLI presentation projection from Rust SDK symbols

Status: complete

## Delivery state

- Delivery completed in [PR #75](https://github.com/cpaikr/opendart/pull/75),
  merged into `rust` as `176f440` after required CI and review resolution.
- The interface projection covers all 85 logical commands, and the private
  dispatch projection covers all 167 physical cases with independent freshness
  identities.
- Public discovery is Rust-symbol-free. The later handwritten cutover has
  retained the projection and retargeted only its private dispatch adapter.

## Outcome

The `opendart` CLI exposes the reviewed CLI-owned command grammar and discovery
contract for every logical OpenDART operation without publishing Rust SDK
implementation names. Its generated interface projection remains complete and
fresh against OpenAPI and the approved interface manifests, while a separately
checked private dispatch adapter remains exhaustive. This plan delivered that
adapter against the then-current generated SDK; the later conformer cutover
retargets the adapter without changing the interface projection.

## Entry conditions at plan start

- The [contract and conformance plan](handwritten-sdk-contract-and-conformance.md)
  has approved all six DS-family interface manifests and their global identity,
  naming, representation, alias, and orphan checks.
- The generated SDK remains the only public preparation and decoding path. This
  plan does not start, select, or expose a handwritten conformer.
- The current CLI derives public names and discovery fields from Rust SDK
  symbols. Those dependencies are transitional rather than compatibility
  requirements because neither Rust package has an accepted registry release.

## Completed work

### 1. Give the CLI projection its own inputs

- Generate CLI grammar and discovery from canonical OpenAPI facts plus the
  reviewed CLI names in `sdk/rust/interface/ds001.toml` through `ds006.toml`.
- Keep physical and logical identities, source parameter concepts, constraints,
  representation availability, and coarse response shape as protocol evidence.
- Separate the CLI interface projection from the private dispatch adapter
  projection. Give each deterministic inputs, its own checksum, and fail-closed
  freshness verification; exclude Rust symbols from the interface checksum.
- Make the interface projection fail closed on incomplete, duplicate,
  ambiguous, stale, or orphan mappings and the adapter fail closed on missing
  or extra typed call wiring.
- Do not copy provider validation, serialization, response schemas, or decoder
  behavior into a CLI-owned wire model.

### 2. Implement CLI-owned grammar and discovery

- Use concise semantic English for commands and flags even when it diverges
  from abbreviated OpenDART stems. Preserve exact logical IDs as stable
  machine-call aliases and physical IDs as discovery evidence.
- Generate the complete parser and catalog from the reviewed CLI vocabulary,
  independently of Rust module, field, input, method, or response-wrapper names.
- Remove `sdk_field` and `response_type` from public discovery. Describe source
  parameter concepts, accepted CLI shapes, constraints, representations, and
  output behavior without leaking private dispatch symbols.
- Preserve keyless discovery, deterministic ordering, process envelopes, exit
  codes, execution controls, and binary artifact behavior.

### 3. Isolate current SDK dispatch

- Retain exhaustive generated typed wiring to the current generated SDK only
  inside the private dispatch adapter projection. Rust symbols may occur there
  and nowhere in the CLI interface projection or public output.
- Prove that Rust symbol changes cannot alter command names, flags, aliases, or
  discovery. Keep every source operation reachable through exactly one reviewed
  CLI product interface.
- Do not introduce a public adapter, generated compatibility alias, generic
  key/value call path, or a second endpoint inventory.

### 4. Qualify and document the presentation module

- Update root and CLI architecture, the CLI public contract and README,
  generation guidance, and package expectations to describe the implemented
  CLI-owned projection while identifying the generated SDK as the current
  runtime conformer.
- Add projection freshness, namespace uniqueness, alias, discovery-schema,
  parser, dispatch-completeness, process, package, and clean-install coverage.
- Run a cross-module review and resolve blocking findings before completion.

## Non-goals

- Do not implement handwritten SDK operations, switch SDK exports, retarget
  dispatch to handwritten types, or delete SDK generation.
- Do not move validation, request serialization, response decoding, HTTP
  behavior, source-shaped wrapper serialization, or provider policy into the
  CLI.
- Do not publish crates, create tags or releases, configure credentials, alter
  distribution, or contact OpenDART.

## Validation

- Every logical operation has one reviewed command, complete reviewed flags,
  stable logical-ID aliasing, physical representation evidence, and exhaustive
  private dispatch to the current generated SDK.
- Public CLI grammar and discovery contain no Rust SDK field, input, method, or
  response-type names and remain unchanged when only private Rust symbols are
  renamed.
- Interface-projection freshness rejects missing, duplicate, ambiguous, stale,
  and orphan mappings without becoming a wire authority. Dispatch-adapter
  freshness independently rejects incomplete typed wiring.
- Existing Go, Rust SDK, CLI process, package, clean-install, provenance, and
  release-guard verification remains green and credential-free.

## Completion criteria

- CLI grammar and discovery are implemented as a self-contained presentation
  module over reviewed product names and protocol identities.
- The current generated SDK remains the only runtime conformer behind private
  exhaustive dispatch; the interface and dispatch adapter projections have
  independent freshness identities, and public SDK behavior and publication
  state are unchanged.
- The [handwritten conformer plan](handwritten-sdk-conformer.md) can replace that
  private dispatch adapter without redesigning or renaming the CLI interface.

## Next action

None in this completed boundary. The
[handwritten conformer plan](handwritten-sdk-conformer.md) owns the subsequent
dispatch retargeting and SDK cutover.
