# Harden SDK wire trust boundaries

## Outcome

Make the SDK's two untrusted wire boundaries fail closed. Sanitized response
metadata must never retain a credential or an encoded credential marker, and an
XML body must become authoritative `Success` or `Status` evidence only after the
entire document satisfies the supported XML grammar. XML values must also
follow XML 1.0 line-ending normalization without changing character references.

## Current state

- Commit `010e47d` makes response-metadata retention fail closed across three
  bounded percent-decoding passes. Each stage is searched for the secret and
  credential marker; malformed escapes, decoded control or non-text bytes, and
  unresolved escapes at the limit cause omission. Unit, transport, and CLI
  coverage includes mixed, fully encoded, nested, lowercase, and malformed
  values. The CodeRabbit follow-up also treats post-decoding `+` bytes as
  ambiguous literal-plus or form-space bytes, closing the partially decoded
  form-encoding case without rewriting retained metadata.
- `WireInspector` uses the event stream from `quick-xml` as if successful
  tokenization proved XML 1.0 well-formedness. Focused review probes found
  accepted illegal names, `<` inside attribute values, invalid comments,
  `]]>` in text, and misplaced or repeated declarations.
- A malformed body such as
  `<result x="<"><status>013</status></result>` can currently become an
  authoritative `SourceReply::Status`. The binary classifier can then discard
  the replayable body instead of returning it as unrecognized evidence.
- `Event::Text` and `Event::CData` retain literal CR and CRLF input. XML 1.0
  requires literal line endings to normalize to LF, while a numeric character
  reference such as `&#13;` must remain a carriage return.
- The authority spike found no maintained parser that satisfies the complete
  contract. `rxml 0.14.0` is streaming and security-oriented but rejects
  processing instructions, `standalone="no"`, and some otherwise accepted XML;
  it has no declared MSRV and includes unsafe implementation code.
  `roxmltree 0.21.1` is safe and conformant for most of the corpus but builds a
  DOM and deliberately does not validate declaration values. `xml 1.3.0` is
  safe, streaming, dependency-free, and MSRV-compatible, but accepts a
  targetless processing instruction, processes internal DTD content before it
  can be rejected, and does not normalize XML values.
- The current `quick-xml` dependency uses its empty default feature set, so the
  implemented conversion path is already UTF-8-only. Making that boundary
  explicit would clarify rather than narrow working behavior. The client
  requires a nonzero envelope bound and permits a caller-selected value; a DOM
  authority therefore adds input-bounded transient memory to a path that
  already materializes the complete `SourceValue`.
- The accepted design uses `roxmltree 0.21.1` with default features disabled as
  the maintained safe-Rust full-document authority and retains `quick-xml` as
  the source-faithful converter. A narrow preflight validates declaration
  values, rejects reserved processing-instruction targets and DTDs, and
  enforces depth and per-element attributes before the tree parser runs. Both
  passes consume the same unchanged bounded bytes.
- Public and internal corpora now cover the reproduced malformed classes,
  supported declarations and document miscellany, namespaces, long tokens,
  literal line endings, and character references. SDK and CLI binary paths
  replay malformed status-like bodies byte-for-byte as unrecognized evidence;
  structured CLI execution reports a malformed envelope.

## Design constraints

- Treat `ResponseMetadata` as a safety boundary, not a best-effort redaction
  layer. Ambiguous or malformed encoding in an otherwise allowlisted value is a
  reason to omit that header, never a reason to retain it.
- Keep credential matching bounded in input length and normalization depth.
  Search every observed normalization stage for both the secret and the
  credential parameter name. Do not allocate or decode without a fixed upper
  bound derived from the already-bounded header value.
- Do not publish a partial XML validator assembled from isolated checks around
  the current event reader. Select a maintained parser or validator that proves
  the complete supported grammar, including names, attributes, comments,
  declarations, CDATA terminators, root placement, and document ordering.
- Continue to reject DTDs and entity declarations. External or custom entity
  expansion must remain impossible.
- Prefer one authoritative XML pass. If the selected dependency cannot both
  validate and supply the source-faithful value events, a validation pass plus
  the existing conversion pass is acceptable only when both consume the same
  bounded byte slice and the duplication is documented.
- Preserve the inspector's body, nesting, and node limits, unknown-field
  retention, mixed-content behavior, expected-root checks, and open status
  model.
- Normalize literal CRLF and CR before values enter `SourceValue`. Apply the
  rule to text and CDATA, but not to a decoded character reference.
- Stay in safe Rust and retain Rust 1.85.0 support. Any new dependency must have
  a maintained release, a compatible MSRV and license, a narrow feature graph,
  and no network or ambient filesystem behavior.

## Implementation plan

### Lock the failures down with public-boundary tests

- Add response-metadata tests that serve an allowlisted header containing:
  fully encoded and mixed encoded `crtfc_key`, lowercase hex, an encoded API
  key, a marker and key separated by ordinary delimiters, nested percent
  encoding, and malformed percent escapes.
- Assert on the public `ResponseMetadata::headers()` result and on serialized
  CLI output where a loopback response already exposes metadata. Do not test
  only a private string helper.
- Add an adversarial XML corpus to the public inspector contract covering every
  reproduced malformed class. Each body must return `EnvelopeError::Malformed`
  rather than `Success` or `Status`.
- Feed the same malformed status bodies through binary classification and
  assert `BinaryReply::Unrecognized` replays every original byte.
- Add XML value tests for lone CR, CRLF, CR, LF, CDATA line endings, and
  `&#13;`. Assert the exact resulting `SourceValue` strings.

### Make response metadata normalization fail closed

- Extract one private predicate whose name states the invariant, such as
  `header_value_is_safe`.
- Check the original bytes first, then perform bounded percent-decoding in
  explicit passes. Search after each pass for the raw secret and
  case-insensitive `crtfc_key`.
- Stop after a small fixed number of passes. Drop the header if another valid
  escape remains at the limit, if a `%` sequence is malformed, or if decoding
  expands into non-text bytes that the policy cannot classify confidently.
- Keep the allowlist check separate and first. Do not generalize metadata
  retention or expose the normalization helper publicly.
- Use table-driven unit tests for normalization mechanics and keep at least one
  transport-level test as the invariant proof.

### Select the strict XML authority

- Build a focused compatibility spike over the adversarial and valid fixture
  corpus before changing production code.
- Compare candidate maintained parsers or validators against the required XML
  1.0 constructs, declaration and comment rules, DTD rejection, line-ending
  semantics, byte-slice input, allocation behavior, MSRV, and feature graph.
- Record the selected dependency and parsing shape in this plan's current state
  when the spike resolves the decision. Remove the spike if it is not itself a
  durable test.
- If no dependency satisfies the contract without a broad new surface, stop
  and narrow the accepted XML grammar explicitly in
  `docs/rust-sdk/public-contract.md`; do not silently retain the permissive
  behavior.

### Rebuild XML inspection around validated events

- Make successful full-document validation a prerequisite for constructing the
  root value or extracting status.
- Keep declaration, processing-instruction, comment, root, and trailing-content
  transitions visible in ordinary control flow. Avoid a generic callback or
  state-machine framework for the small fixed grammar.
- Reuse the existing `XmlFrame`/`XmlContent` model where it remains clear.
  Replace only the token authority and normalization points needed by the
  contract.
- Normalize literal line endings at the event boundary so all downstream
  status, message, text, attribute, and mixed-content logic observes one
  canonical representation.
- Preserve reference decoding as a separate path so `&#13;` is not normalized
  as literal source text.

### Close SDK and CLI integration coverage

- Extend structured JSON/XML and binary loopback fixtures with malformed XML
  status-like bodies and confirm the public CLI reports decode or unrecognized
  evidence without losing bytes or exposing credentials.
- Update public SDK documentation only where the accepted XML grammar or
  metadata omission policy becomes more explicit.
- Refresh `Cargo.lock` and reviewed package inventories if the parser choice
  changes packaged files or dependencies.

## Validation

- Run focused `opendart` unit and public-contract tests after each boundary is
  changed.
- Run both structured and binary `opendart_compat` loopback suites so the
  classifier and CLI behavior are exercised, not merely compiled.
- Run the pinned stable format, Clippy, all-feature, no-default-feature,
  rustdoc, package, and offline repository verification gates from
  `sdk/rust/README.md`.
- Run the MSRV checks and inspect the no-default dependency tree after any
  dependency change.
- Confirm no test reads `.env.local` or requires `OPENDART_API_KEY`.

## Completion criteria

- No supported encoding of the credential marker or current secret survives in
  retained response metadata.
- Every adversarial malformed XML body fails inspection, and binary inspection
  replays it byte-for-byte as unrecognized evidence.
- Valid XML behavior, source uncertainty, size limits, and exact character
  reference behavior remain intact.
- Literal XML line endings are normalized exactly once.
- The complete offline Rust and repository gate passes on stable and MSRV.

## Next action

Complete. PR #46 merged into `rust` after its required Linux, macOS, and
Windows verification passed.
