# Rust SDK Generation

Planning source: [Public Rust SDK](../../tasks/rust/public-rust-sdk.md).

## Purpose

Generate complete, deterministic request and conservative wire code from the
canonical OpenAPI 3.2 contract without coupling consumers to a generic OpenAPI
generator or hand-writing one method per physical endpoint.

## Inputs and source of truth

The generator reads `openapi/openapi.yaml` and its confined references through
the existing private Go OpenAPI boundary. The portable bundle may be used for
comparison and provenance, but it does not become a second inventory.

The SDK model may use:

- path, HTTP method, parameters, security, responses, and media types;
- explicit schemas and constraints;
- `operationId` and stable `x-opendart.logicalOperationId`;
- guide group and API identity required for grouping or diagnostics;
- source evidence that changes safe serialization or response routing; and
- stable naming inputs.

It must not turn descriptive prose, observations, or application-specific
collection analysis into runtime validation or success policy.

## Repository-owned SDK model

Add a narrow repository-owned projection at the `internal/openapi` boundary.
Third-party OpenAPI model types remain private. `internal/sdkgen/model` validates
and normalizes that projection into concepts shared by language emitters:

- `LogicalOperation` and `PhysicalOperation`;
- request parameters with wire name, requiredness, shape, style, explode, and
  explicit constraints;
- authentication scheme and trusted relative target;
- supported request/response representations;
- source status-envelope location;
- conservative object, array, and opaque scalar shapes;
- descriptions selected for generated public documentation; and
- stable naming inputs.

The model is an internal Go API. Initially it can be passed in memory. Do not
publish or stabilize a serialized intermediate format until another language
has a concrete need.

## Physical and logical operations

Every canonical OpenAPI path/method with one `operationId` maps to one physical
callable operation. Its primary request/normal-success representation follows
the documented path and response contract. An alternate response media type
does not create another callable operation: in particular, an XML source-error
body from an operation whose normal success is ZIP remains an outcome of that
same physical operation. JSON and XML paths with distinct operation IDs remain
separate physical operations even when they share a logical identity.

Operations carrying the same stable logical identity may share one public
request input type and representation selector when their parameter contracts
are equivalent.

Generation fails when paired physical operations disagree on a field that the
shared public type would hide. It must not silently select one variant.

Naming rules must be deterministic and validated for collisions across:

- logical and physical operation types;
- parameters after Rust case conversion and reserved-word escaping;
- generated response shapes;
- source group modules; and
- future additions that normalize to an existing public name.

Persist a generated mapping from public Rust name to OpenAPI operation ID and
logical operation ID so compatibility review can distinguish source renames
from semantic changes.

## Generated request behavior

Generate:

- request input structs and constructors;
- required and optional parameter handling;
- explicit array/cardinality constraints;
- exact path and query encoding;
- physical/logical operation markers;
- representation support and response media metadata;
- authentication requirement; and
- safe source URLs and provenance for documentation.

Reject before I/O:

- missing required inputs;
- an unsupported representation;
- wrong scalar/list shape;
- empty lists or cardinality outside an explicit schema constraint;
- invalid target construction; and
- generator/model states that cannot preserve documented serialization.

Do not add validators for narrative string lengths, date formats, enum prose,
defaults, or ranges absent from the canonical schema. Preserve the explicit
multi-company `style: form`, `explode: false` comma serialization as a generated
contract test.

## Generated response behavior

Generate only what the canonical evidence supports:

- object and array structure;
- documented property names and descriptions;
- explicitly typed containers and scalars;
- optionality where it is actually known; and
- unknown-field retention where additional properties are allowed or source
  evolution is plausible.

Emit a distinct public `#[non_exhaustive]` response type for each structured
physical representation. Object properties are public for reading; additive
properties remain private and are exposed through `additional_field` and
`additional_fields`. Arrays become `Vec<T>`, source-established strings become
`String`, `OpenDartStatus` becomes `SourceStatus`, and unresolved scalars remain
`SourceValue`. Requiredness is preserved rather than inferred. Generated,
path-aware decoders are private and are bound into `PreparedRequest<T>` by the
corresponding `prepare_json` or `prepare_xml` method.

Represent an undocumented scalar with `SourceValue` or an equivalent
source-faithful form. Do not guess `String`, numeric, date, or enum types from
examples or descriptions. Strong domain conversions belong in consumer code or
an explicitly evidence-backed future layer.

The canonical `OpenDartStatus` schema is a deliberate forward-compatibility
exception to ordinary closed-enum generation. Map references to it onto the
open `SourceStatus` wrapper: documented enum members become associated constants
or predicates, while unknown future strings remain representable.

Status-envelope parsing is handwritten because it spans operations and must
recognize source errors independently of HTTP status. ZIP success bodies remain
streaming/binary; empirically supported XML error bodies route to bounded
prefix inspection and the XML envelope inspector. Prefix inspection is
handwritten shared runtime behavior and must replay every consumed byte for a
ZIP or unrecognized result.

## Emitter and command

Add one direct repository command, with final naming fixed during
implementation, equivalent to:

```sh
go run ./cmd/opendart-tool generate-sdk \
  --language rust \
  --root openapi/openapi.yaml \
  --output sdk/rust/crates/opendart/src/generated
```

The command:

1. Loads and validates the complete canonical document.
2. Builds and validates the normalized SDK model.
3. Renders into a new staging directory.
4. Emits deterministic generator-owned Rust formatting.
5. Validates the complete staged output and ownership marker.
6. Replaces only the owned generated subtree with rollback on publication
   failure.

Do not invoke Cargo from Go merely to hide validation. Repository CI may run
Cargo after generation/freshness checks. Do not add Make, Just, npm, shell, or a
second CLI wrapper.

## Checked-in output and freshness

Generated files are reviewed source artifacts. Each includes:

- a do-not-edit marker;
- generator schema/version identity;
- the deterministic SDK-projection checksum; and
- stable formatting independent of local paths or timestamps.

The generated module is intentionally marked `#[rustfmt::skip]`. Generator
freshness, rather than the locally selected rustfmt version, enforces its
canonical compact formatting and avoids toolchain-only projection churn.
`cargo fmt --check` covers handwritten Rust; Cargo checks, Clippy, tests, and
rustdoc still compile and validate the generated subtree.

The SDK projection contains only normalized canonical inputs that affect the
generated Rust contract, its public documentation, or runtime behavior. A
description not emitted into SDK docs, provenance, or other specification
change outside that projection does not rewrite generated Rust. The generator
fails if it cannot classify whether an input affects the projection; it does
not silently exclude an unknown construct.

Full source provenance is release metadata rather than an SDK-model input or
generated-file freshness signal. A crate release records the exact Git revision,
the semantic specification source release when one exists, and the independently
selected canonical bundle checksum in the handwritten provenance module. The
release guard verifies that the named tag contains the canonical source inputs
without requiring the evolving working tree to equal that older tag; the tag
does not claim byte identity with the selected generated bundle. That metadata
advances only in a crate release PR, so an SDK-irrelevant specification change
neither touches the Rust component nor creates a crate release.

Extend `opendart-tool verify` to render the Rust output in a confined temporary
location and compare it byte for byte with the committed tree. Verification is
offline and cannot rewrite the working tree.

Consumer compilation reads only committed Rust. Do not use `build.rs`, a proc
macro that parses OpenAPI, a Git submodule, network download, or an environment
path back to the repository specification.

## Failure policy

Generation fails closed on:

- an unsupported OpenAPI construct affecting a public request or response;
- unresolved or non-confined references;
- missing or duplicate operation identity;
- public-name collision;
- incompatible physical variants sharing a logical identity;
- unsupported parameter serialization;
- ambiguous authentication placement;
- unrepresentable response shape; or
- nondeterministic output.

Do not fall back to a generic untyped request method without review. Add narrow
model and emitter support with fixtures, or explicitly exclude the operation
and fail the complete-coverage gate until resolved.

## Tests

### Model tests

- Representative JSON/XML logical pairs.
- ZIP success plus XML error representation.
- Exact physical-inventory checks for every ZIP operation with an alternate
  XML error media type, proving that the alternate outcome creates no extra
  callable operation.
- Multi-company comma serialization and cardinality.
- Open source-status mapping for documented and unknown status strings.
- Unknown scalar and additional-property preservation.
- Numeric/reserved schema names and Rust naming collisions.
- Missing, duplicate, and contradictory logical identities.
- Unsupported style, explode, security, or schema constructs.

### Generated request vectors

For every physical operation, derive at least one structurally valid request
case from explicit fixture inputs and assert:

- method and relative path;
- ordered or canonically compared query pairs;
- exact percent encoding and list serialization;
- credential omission before authorization;
- operation identities and representation; and
- no unexpected parameter, default, or inferred validator.

Do not hard-code a volatile endpoint total. Compare the set of generated
physical identities with the canonical inventory and require exact set
equality.

### Freshness and determinism

- Generate twice in distinct temporary roots and compare bytes.
- Verify no absolute paths, timestamps, credentials, or environment-dependent
  ordering appear.
- Detect handwritten or stale changes under the generated ownership marker.
- Format generated Rust and require a clean diff.
- Compile generated code with default and no-default features.

The ZIP fixtures cover byte-for-byte successful archive replay, a recognized
XML source error, misleading `Content-Type`, a truncated prefix, and an XML
candidate that exceeds the configured inspection bound.

## Future emitters

A Python or TypeScript emitter reuses the normalized model and language-neutral
request vectors. It owns idiomatic builders, async runtime integration, package
metadata, and errors for its ecosystem. It must not parse generated Rust or
copy an operation inventory into a second hand-maintained file.

Extract shared serialized fixtures only when a second emitter exists and can
prove the format. Until then, keep test data as repository implementation
detail rather than a promised public artifact.

## Acceptance criteria

- Canonical and generated physical-operation identity sets are equal.
- Generated request behavior is deterministic and contains every explicit
  serialization rule.
- Unknown source scalar types and fields remain representable.
- Unsupported constructs fail repository verification with operation and
  source locations.
- Generated files require no consumer-side generator or repository checkout.
- A future language emitter can consume the normalized model without depending
  on Rust syntax or public Go APIs.
