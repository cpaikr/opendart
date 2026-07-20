# Public agent-first OpenDART CLI

## Outcome

Publish an `opendart-cli` crate whose `opendart` binary exposes every logical
Rust SDK operation through a generated, non-interactive, agent-first shell
interface. Calls use SDK request and response types directly, emit complete
typed compact JSON envelopes, and preserve binary bodies at explicit paths
without duplicating endpoint contracts or adding a second output notation.

## Current state

- [ADR 0003](../../docs/decisions/0003-agent-first-opendart-cli.md)
  accepts the separate public CLI, shared-generator, typed-response,
  agent-output, credential, and independent-release decisions.
- The [target architecture](../../docs/rust-cli/architecture.md) and
  [public contract](../../docs/rust-cli/public-contract.md) define the target
  module seams and user-visible behavior. No CLI crate exists yet.
- The `opendart` SDK already provides complete generated logical operations,
  representation-specific typed responses, one-attempt client execution,
  sanitized metadata, replaying binary streams, and optional source-faithful
  JSON serialization.
- The private Go model already owns logical/physical identity, Rust names,
  parameters, constraints, representations, guide identity, and response
  shapes. It does not yet provide CLI presentation metadata or independently
  checksummed SDK and CLI projections.
- The Rust workspace currently contains only `opendart`. Its generated mapping
  is private and test-only, so CLI completeness and dispatch must be generated
  from the private model rather than discovered from the SDK at runtime.
- Registry availability checks are dated evidence, not reservations. Recheck
  both package names immediately before any authorized publication.
- The output decision has been narrowed to compact JSON only. SDK-owned direct
  `serde_json` encoding preserves exact `SourceValue` number lexemes through
  `RawValue`; TOON compatibility, dependencies, flags, and numeric-domain
  restrictions are out of scope.
- Review has resolved the exact SDK-pin workflow, self-sufficient discovery,
  SDK-owned JSON-number contract, finite-default artifact policy, and initial
  source-install host matrix. No product decision remains open before
  implementation.
- The unfinished [public SDK task](../../tasks/rust/public-rust-sdk.md) still
  owns SDK crates.io authority and verification. It must resume before the CLI
  can publish.

## Decisions

- Package `opendart-cli`, binary `opendart`, same isolated Cargo workspace,
  binary product with no supported Rust library interface.
- One generated command per SDK logical operation; canonical kebab-case SDK
  name plus exact logical-ID alias. Physical IDs remain evidence.
- Operation-specific flags only. Repeated list flags become SDK `Vec<String>`;
  the SDK owns final query serialization.
- Require representation selection only when a logical operation has multiple
  structured choices. A sole representation is implicit; ZIP operations require
  an explicit no-clobber output path.
- Generate command construction, discovery, and typed dispatch from the same
  normalized model as the SDK. No runtime OpenAPI parse or hand-maintained
  endpoint registry.
- Add optional SDK-owned `serde-json` serialization. Do not create CLI mirror
  response types or serialize credential-bearing/executable SDK values.
- Use compact JSON as the only stdout format. Emit one complete stable envelope,
  natural `SourceValue` values, structured stdout errors, safe diagnostic-only
  stderr, and exits `0`/`1`/`2`.
- Only `OPENDART_API_KEY` supplies credentials. Expose explicit SDK timeout and
  envelope-limit overrides, with no config file, retry, proxy, origin,
  pagination, extraction, or truncation.
- Version SDK and CLI independently. Implement the real CLI consumer before the
  first SDK publication; publish and verify the SDK before the CLI.
- Initial distribution is crates.io source only. The
  [prebuilt-release task](../../tasks/rust/opendart-cli-prebuilt-releases.md)
  owns later GitHub artifact hardening.
- Give every binary call a finite CLI-owned default byte budget and allow a
  positive `--artifact-limit-bytes` override. Count streamed SDK body bytes and
  clean up without publishing when the inclusive limit is exceeded. The
  default is 512 MiB (`536870912` bytes).
- Support crates.io source installation on Linux, macOS, and Windows. Verify
  source-install behavior natively on all three while leaving prebuilt targets,
  libc, minimum OS versions, signing, and archive policy to the separate
  hardening task.

## Ordered work

### 1. Add the SDK JSON serialization seam

- Begin with a compatibility spike over the repository's scalar and response
  corpus using direct `serde_json` text encoding. Prove that `RawValue` preserves
  exact valid `SourceValue` number lexemes without passing through
  `serde_json::Value`, fixed-width integers, or floating point. Cover `-0`,
  fractional trailing zeros, exponent case and sign, huge integers and
  exponents, and invalid forms before adding a supported SDK serialization API.
- Add an optional direct `serde` dependency to `opendart`. Name the public
  feature `serde-json`, activate `serde_json/raw_value`, and
  gate every public serialization implementation on that feature; existing
  parser dependencies may already include Serde transitively, but dependency
  presence must not make serialization part of the default interface.
- Generate feature-gated `Serialize` implementations for every structured
  response object, including deterministic flattening of additive fields.
- Define deliberate serialization for `SourceStatus`, `SourceValue`,
  `StatusEnvelope`, `SourceReply<T>`, `SourceResponse<T>`, response metadata,
  headers, and HTTP version. Use stable adjacent tags for reply variants, omit
  absent optional fields while preserving explicit null, and retain sanitized
  header values as exact byte arrays.
- Encode `SourceValue` as natural JSON values. Validate public number
  construction against the SDK-owned JSON grammar documented in the public
  contract, make `SourceValue::number` return
  `Result<SourceValue, InvalidSourceNumberError>`, and emit the exact retained
  lexeme through `RawValue`; reject invalid construction and any round-trip
  through `serde_json::Value`.
- Migrate the bounded JSON inspector and caller-owned fixtures to the fallible
  constructor. Parser-validated input may use a private checked invariant, but
  no public or generated path may recreate unchecked numeric construction.
- Prove at compile time that `ApiKey`, authorized and prepared requests,
  clients, and body streams remain non-serializable.
- Add SDK contract fixtures for success/status replies, unknown fields, scalar
  kinds, numeric boundaries and invalid caller-created number spellings, header
  order, and both feature-enabled and feature-disabled graphs.

Completion evidence: the exact-JSON feasibility gate passes, stable/MSRV
all-feature and no-default-feature gates pass, generated freshness is clean,
and serialization snapshots use only repository-owned public names and exact
source number spellings.

### 2. Deepen the generator into a Rust artifact set

- Extend the normalized model with deterministic CLI command names,
  descriptions required for discovery, and response type/field metadata.
- Give the internal semantic model, SDK projection, and CLI projection distinct
  schema identities and checksums. Generated SDK headers use only the SDK
  projection identity, so CLI-only model or presentation changes cannot rewrite
  SDK source or create SDK release eligibility.
- Keep SDK rendering and CLI rendering as internal adapters behind one Rust
  artifact-generation interface. Stage and validate both owned output trees
  before publishing either tree.
- Add a CLI renderer for the command catalog, compact list records, detailed
  operation records, typed SDK input construction, and exhaustive dispatch.
- Validate kebab-name and logical-ID collisions, parameter flag collisions,
  reserved globals, representation coverage, SDK method/type references, and
  complete logical-operation coverage.
- If a presentation overlay is needed, key it by logical ID and restrict it to
  descriptions and examples. Reject orphaned IDs and any attempt to change
  operation, parameter, representation, or response facts.
- Project operation and parameter descriptions from canonical OpenAPI before
  adding an overlay; an overlay must not become the default source of help facts.
- Extend `generate-sdk` and offline freshness verification rather than adding a
  second generator command or consumer build step.

Completion evidence: one model build deterministically reproduces both owned
trees, a CLI-only prose fixture leaves the SDK projection unchanged, and stale,
missing, extra, or half-published output fails closed.

### 3. Establish the binary crate and generated discovery

- Add `sdk/rust/crates/opendart-cli` as a workspace member with workspace
  edition, MSRV, license, repository, lint, and lockfile policy.
- Depend on `opendart` through a path-plus-exact-version dependency with its
  client and `serde-json` features. Every SDK version-bump change updates this
  local exact pin and the lockfile before merge, even when no CLI release is
  planned. Use `clap`'s builder interface for the generated command tree.
- Keep `main` thin and the package binary-only. Internal handwritten modules own
  app orchestration, output, credentials, client settings, and artifacts;
  generated code owns operation breadth.
- Use non-exiting argument parsing and translate usage failures into the stable
  output envelope. Construct and prepare the typed SDK request before reading
  the API key so SDK-backed input violations are also usage failures.
- Implement keyless structured home, `operations list`, and
  `operations describe`. Keep list records compact and make home and detail
  records expose spawnable executable paths, argument arrays, credential names,
  global controls, representation selection, ZIP output requirements, and
  additive response-shape behavior.
- Generate scalar and repeated-list flags directly from SDK inputs. Do not
  accept generic parameter documents, raw query pairs, or physical URLs.
- Add process-level discovery tests proving name/ID equivalence, strict unknown
  input, representation rules, deterministic ordering, and no credential read.
  A fixture consumer must construct an invocation through SDK preparation using
  discovery JSON alone, without parsing prose or invoking `--help`.

Completion evidence: every generated logical operation is discoverable by both
identifiers without network access, and the CLI builds on stable and MSRV.

### 4. Execute typed structured calls and encode outcomes

- Let generated dispatch build the SDK operation input and prepare the exact
  JSON or XML request before credential access. Treat `PrepareError` as a
  structured usage failure.
- Resolve only `OPENDART_API_KEY`; pass it directly to `ApiKey` after request
  preparation succeeds.
- Add `--connect-timeout-ms`, `--read-timeout-ms`, `--total-timeout-ms`, and
  `--envelope-limit-bytes`. Call SDK builder setters only for supplied values so
  omitted settings inherit SDK defaults. The SDK validates otherwise well-formed
  settings after key binding and before any network attempt.
- A handwritten generic executor accepts the prepared request, calls
  `Client::execute` for `T: Serialize`, and emits the complete typed response.
- Define CLI-owned operation and error wrappers while embedding the SDK response
  without endpoint-specific conversion.
- Encode directly to compact JSON with the SDK-supported `serde_json` path. Do
  not pass the response through `serde_json::Value` or another intermediate
  data model.
- Buffer the single encoded document before writing it. Convert encoder failures
  into the JSON error envelope; if stdout itself fails, exit `1` without
  appending a corrupt replacement document.
- Pin the reviewed `serde_json` behavior in the manifest and `Cargo.lock`, then
  run a compatibility corpus including Unicode, controls, nested/mixed arrays,
  unknown fields, key ordering, and exact arbitrary-precision number spellings.
- Classify only SDK `Success` as exit `0`; preserve every `Status` response and
  exit `1`. Map configuration, client, decode, and output failures into stable
  sanitized CLI error codes. Parsing and preparation failures exit `2`.
- Keep stdout to one structured document and stderr to opt-in safe diagnostics.

Completion evidence: typed JSON and XML loopback responses produce the exact
documented JSON envelopes, source statuses remain complete and nonzero, and
credential sentinels are absent from every captured channel and error.

### 5. Preserve binary replies as artifacts

- Expose the 512 MiB (`536870912` byte) default and positive override through
  discovery, validation, errors, and process-level tests.
- Let generated binary dispatch construct and prepare the SDK request before
  credential access. Treat `PrepareError` as a structured usage failure.
- Require a non-empty Unicode `--output` only for ZIP operations, reject `-`,
  and reject an existing destination before calling OpenDART.
- Create the temporary file in the destination directory, stream every
  `Archive` or `Unrecognized` chunk once, enforce the inclusive effective byte
  limit before each write, flush safely, and publish with no-clobber semantics.
- Emit the final path, byte count, classification, and sanitized SDK response
  metadata. Preserve the caller's path spelling in the artifact reference.
  `Archive` exits `0`; `Unrecognized` and `Status` exit `1`.
- Finish encoding the artifact report before its commit point. Then publish with
  no clobber and write the prepared report to stdout; if that stdout write fails,
  leave the complete published artifact in place and exit `1`.
- For a source `Status`, remove the owned temporary file and publish no final
  destination. Do the same on stream, deadline, or write failure; never expose a
  partial final artifact.
- Refuse extraction and do not infer success from file extension or
  `Content-Type`.
- Test split signatures, empty archives, unrecognized bodies, alternate XML
  statuses, incomplete streams, destination races, symlinks/existing paths,
  cleanup, exact-default and override boundaries, post-publication stdout
  failure, and platform-specific no-clobber behavior.

Completion evidence: fixture hashes prove exact archive and unrecognized bytes,
and no failure path overwrites or publishes a partial destination.

### 6. Close verification and live confidence

- Extend repository verification and package inventories for the second crate,
  both generated trees, both feature graphs, and CLI output fixtures.
- Reuse or deepen the repository-only `opendart_compat` origin seam for loopback
  CLI tests. A CLI built only under `cfg(opendart_compat)` reads the test-only
  `OPENDART_COMPAT_ORIGIN` environment variable and calls the existing public
  compatibility bridge; ordinary builds neither read nor recognize that name
  and expose no origin override.
- Cover parsing, preparation, source success/status, malformed bodies, timeout
  controls, structured errors, broken output, and binary streaming at the
  process interface.
- Add an opt-in read-only smoke suite requiring both `OPENDART_API_KEY` and
  `OPENDART_LIVE_TESTS=1`. Absence of either gate skips before reading or using
  the credential.
- Keep live assertions structural: operation identity, representation,
  successful sanitization, and decodable reply class. Do not snapshot volatile
  business data or treat live observations as generated input.
- Run ordinary Go, stable/MSRV Cargo, all-feature/no-default-feature, lint,
  rustdoc, generation, package, and offline verification without a key.

Completion evidence: the complete credential-free gate passes offline after
dependency fetch, and an explicitly authorized local smoke run proves both a
structured and binary path without retaining source bodies or secrets.

### 7. Prepare independent packaging and release ownership

- Add CLI README, changelog, license inclusion, docs, exact package inventory,
  and clean `cargo install --locked --path` verification on Linux, macOS, and
  Windows.
- Add an independent Release Please component, version/changelog/tag identity,
  workspace-lock update, and release-guard rules. CLI changes must not trigger a
  specification or SDK release when their projections are unchanged.
- Exact-pin the reviewed SDK and JSON encoder behavior in the CLI manifest,
  package the reviewed lockfile, document `cargo install --locked` as the
  reproducible path, and require explicit CLI compatibility review before a
  behavior-defining pin moves. An SDK version-bump PR updates the CLI's local
  path-and-version pin for workspace resolution but does not itself release the
  CLI.
- Prepare crates.io package and post-publication verification logic without
  granting registry authority in pull requests or generic verification.
- At the release gate, pause this plan and resume the public SDK task. Publish
  and verify the SDK artifact before returning here to publish the dependent
  CLI package.
- Recheck `opendart` and `opendart-cli` registry ownership immediately before
  authorization. A name conflict requires a new explicit product decision.

Completion evidence: release dry runs reproduce reviewed package contents,
component outputs are isolated, interrupted-release recovery is specified, and
no workflow can publish without the dedicated authority change.

### 8. Publish and adopt the source distribution

- After the SDK task verifies a registry version containing the JSON
  serialization contract, confirm that the already-aligned CLI exact pin
  matches that verified version;
  do not defer the manifest update until after SDK publication.
- Through a separately authorized release, publish `opendart-cli`, download the
  accepted crate, compare checksum/manifest/contents with the candidate, inspect
  docs.rs, and install it in a clean environment.
- Run keyless discovery and an explicitly authorized smoke call from the
  registry-installed binary before finalizing the matching GitHub component
  release.
- Record actual consumer feedback before expanding aliases, input modes, output
  transformations, or distribution channels.
- Leave prebuilt archives, installers, and package-manager publication to the
  separate hardening task.

Completion evidence: clean Linux, macOS, and Windows environments install the
reviewed crates.io package, discovery matches the canonical model, and release
provenance points to the immutable reviewed revision.

## Cross-plan constraints

- The CLI plan may change the SDK only to establish the selected serialization
  consumer and repository-only test seam. The SDK task retains registry
  publication and collector-adoption ownership.
- The canonical OpenAPI contract and private normalized model remain endpoint
  sources of truth. CLI observations never update either automatically.
- CLI generation and ordinary verification are offline and credential-free.
- Source-status exit policy belongs to the CLI only; the SDK continues to
  return status evidence without application meaning.
- Initial crates.io release work must not absorb the prebuilt-binary task. The
  TLS and provenance questions differ and require their own compatibility gate.

## Overall acceptance criteria

- Every SDK logical operation has exactly one generated CLI command, exact
  logical-ID alias, correct parameter contract, and complete representation
  coverage.
- Structured dispatch uses the SDK's public typed preparation and execution
  path; no endpoint-specific request or response mirror exists in the CLI.
- The SDK `serde-json` feature is optional, stable, source-faithful, and
  incapable of serializing credential-bearing or executable SDK request/client
  values.
- Discovery is complete, deterministic, keyless, and sufficiently structured
  for an agent to construct a valid call without scraping human help.
- Compact JSON carries the complete typed value. Exact arbitrary-precision
  `SourceValue` number lexemes survive direct text encoding; invalid source-number
  construction and any numeric narrowing fail release validation.
- Output channels, error codes, source-status exits, and ZIP artifacts match the
  public contract under process-level tests.
- No command retries, follows redirects, reads ambient proxy configuration,
  chooses an origin, paginates, truncates, extracts, or stores credentials.
- Normal verification never reads an API key or contacts OpenDART. Live calls
  require the explicit second gate and retain no credential or volatile body.
- SDK, CLI, and specification release eligibility, versions, tags, changelogs,
  package contents, and publication authority remain independent.

## Progress

- Work 1 is merged on `rust`: the SDK feature graph, fallible JSON-number
  boundary, generated/shared serialization, exact-number snapshots, and
  non-serialization assertions are present.
- Works 2 and 3 are implemented in the current delivery slice. One normalized
  build produces separately identified SDK and CLI projections, validates both
  owned trees before publication, and verifies both offline. CLI-only prose is
  covered against SDK checksum and byte drift.
- The binary-only `opendart-cli` workspace crate uses the exact local SDK pin,
  generated clap command construction, exhaustive typed preparation dispatch,
  and keyless machine-readable discovery. Process tests cover every canonical
  name and logical-ID alias, strict usage, representation rules, help/version,
  and a discovery-only invocation consumer.
- Works 4–7 remain. Authenticated execution has not begun, and no publication
  or prebuilt-release work is authorized by this plan state.

## Next action

After works 2 and 3 pass review and merge, implement Work 4 as the next slice:
execute prepared structured requests through the SDK, apply the approved client
controls, and encode typed success, source status, and sanitized failure
outcomes. Keep ZIP artifact handling in Work 5.
