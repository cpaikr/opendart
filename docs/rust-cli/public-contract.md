# OpenDART CLI public contract

This document defines the approved shell, output, credential, and compatibility
interface for the implemented, unpublished `opendart` binary. See the
[implementation plan](../../plans/rust/public-opendart-cli.md) for status.

## Command grammar

The primary call shape is:

```text
opendart call <operation> [operation flags] --representation <json|xml>
```

`<operation>` accepts either the canonical kebab-case name derived from the
public SDK input type or its exact logical OpenDART ID. Discovery returns both.
Physical OpenAPI operation IDs are evidence in discovery and output, not call
aliases.

When a logical operation supports more than one structured representation,
`--representation` is required. If a future operation has exactly one
structured representation, the generator selects that sole representation and
does not expose a meaningless selector. ZIP-only operations do not accept a
representation flag and require `--output <path>`.

`--output` accepts a non-empty Unicode filesystem path. The value `-` is
rejected rather than interpreted as binary stdout, and the successful artifact
record returns the same path spelling supplied by the caller.

Generated parameter flags use kebab case from the SDK field name. Required SDK
constructor inputs are required flags; optional SDK builders are optional
flags. Scalar flags accept one value. A list input repeats the same flag once
per item; the SDK, not the CLI, owns its wire-level joining and encoding.
Unknown flags, duplicate scalar flags, positional spillover, missing values,
unsupported representations, empty SDK-required values, and explicit SDK
cardinality violations fail with exit `2` before credentials or network access.

The CLI does not accept generic key/value maps, structured request documents,
raw paths, arbitrary query parameters, or physical endpoint URLs.

## Agent discovery

These commands require no credential and perform no I/O beyond stdout:

```text
opendart
opendart operations list
opendart operations describe <operation>
```

With no arguments, the binary emits a compact `kind: home` document containing
the exact absolute executable path, a display path with the home directory
collapsed to `~`, a one-sentence description, structured argument prefixes for
listing, describing, and calling operations, the call-scoped execution flags, and
the name and scope of the credential environment variable. It does not choose
an authenticated endpoint or print the full operation inventory. Agents spawn
`executable.path` directly and append an `argv` array; they never have to parse
a shell command string or expand `~`.

```json
{
  "kind": "home",
  "executable": {
    "path": "/home/example/bin/opendart",
    "display": "~/bin/opendart"
  },
  "description": "Call OpenDART through the typed Rust SDK.",
  "commands": {
    "list": {"argv": ["operations", "list"]},
    "describe": {"argv": ["operations", "describe", "<operation>"]},
    "call": {"argv": ["call", "<operation>"]}
  },
  "authentication": {
    "environment": "OPENDART_API_KEY",
    "required_for": ["call"]
  },
  "call_flags": [
    {"name": "--connect-timeout-ms", "required": false, "shape": "positive_integer"},
    {"name": "--read-timeout-ms", "required": false, "shape": "positive_integer"},
    {"name": "--total-timeout-ms", "required": false, "shape": "positive_integer"},
    {"name": "--envelope-limit-bytes", "required": false, "shape": "positive_integer"}
  ]
}
```

`operations list` emits `kind: operations` and a compact record for each logical
operation:

- canonical name and logical ID;
- source group and API ID; and
- available representations.

The exact outer shape is `{"kind":"operations","operations":[...]}`.
Each record has string fields `name`, `logical_id`, `group`, and `api_id`, plus
a `representations` array containing `json`, `xml`, or `zip`. Records sort by
canonical name and then logical ID; representations use that fixed order.

`operations describe` emits `kind: operation` and is self-sufficient for
constructing a valid call. It adds operation-specific flags, requiredness, list
shape, explicit constraints, credential requirements, call execution flags,
representation-specific invocation templates, physical IDs and SDK response
types, response field structure, ZIP destination requirements, and the official
guide URL. Presentation text may be improved without changing these generated
facts.

The exact outer shape is `{"kind":"operation","operation":{...}}`.
The operation has `name`, `logical_id`, `group`, `api_id`, `guide_url`,
`description`, `invocation`, `execution_flags`, `flags`, and `representations`.
`invocation` contains `argv_prefix: ["call", "<canonical-name>"]` and
`required_env: ["OPENDART_API_KEY"]`. `execution_flags` repeats the complete home
records so the detail document stands alone.

Each operation flag has `name`, `sdk_field`, `description`, `required`,
`value_kind`, and `occurrence`. Scalar flags use `occurrence: "once"`; list
flags use `occurrence: "repeat"`. `min_items` and `max_items` appear only when
the SDK enforces them.

Each representation has `name`, `physical_id`, `response_type`,
`response_shape`, `selector_argv`, and `output`. `selector_argv` is
`["--representation", "json"]` or `["--representation", "xml"]` when
selection is required and is empty when the operation has one implicit
representation. Structured output is `{"kind":"stdout"}`. ZIP output is
`{"kind":"artifact","argument_argv":["--output","<path>"],"required":true,"existing_destination":"reject"}`.
The artifact record will also expose `limit_argument_argv` as
`["--artifact-limit-bytes","<positive-integer>"]`, `limit_required: false`,
and `default_limit_bytes: 536870912`.

Response shapes are recursive objects using `kind` values `object`, `array`,
`source_value`, `source_status`, or `binary`; object shapes have an
`additional_fields` Boolean and fields with `name`, `required`, `shape`, and an
optional `description`, while an array has `items`. `additional_fields: true`
means the SDK retains and serializes source fields not yet present in the
generated schema.

“Construct a valid call” means that a consumer using only the discovery JSON
can assemble an argument vector accepted by CLI parsing and SDK preparation
when its supplied values satisfy the advertised constraints. It does not mean
the credential is present or that OpenDART will accept a semantically valid
request at runtime.

Every subcommand also supports concise conventional `--help`; `--version`
reports the CLI package version. Help and version are the only intentionally
plain-text successful outputs.

## Call execution controls

All machine-readable stdout uses compact JSON. The CLI has no output-format
selector and does not expose TOON as an alternate contract.

Call-scoped optional controls expose existing SDK client builder settings plus
one CLI-owned artifact budget:

- `--connect-timeout-ms <positive integer>` — connection establishment;
- `--read-timeout-ms <positive integer>` — time between body reads;
- `--total-timeout-ms <positive integer>` — total request and body deadline;
- `--envelope-limit-bytes <positive integer>` — structured or alternate
  envelope limit; and
- `--artifact-limit-bytes <positive integer>` — advertised by ZIP output
  metadata instead of the common call flags; it overrides the finite default
  for a ZIP body and is rejected for structured representations. The default
  is 512 MiB (`536870912` bytes).

Omitting an SDK-owned control leaves that builder setting untouched so the SDK
remains its default source of truth. Omitting the artifact control uses the
CLI-owned finite default. Invalid integer syntax and zero fail as usage errors
before credentials are read. A value that parses but the SDK cannot represent
safely is a sanitized configuration failure.

There is no retry count, proxy, origin, decompression, pagination, extraction,
or persistent configuration control. The CLI sets a sanitized product/version
user-agent suffix internally.

## Credentials

Authenticated calls read only `OPENDART_API_KEY` from the inherited process
environment. An absent or empty value is a structured execution error.

The CLI has no API-key value flag, file flag, dotenv loading, config file,
keychain integration, or prompt. It passes the exact non-empty value into the
SDK `ApiKey` type. CLI-owned values, diagnostics, snapshots, paths, and errors
never include it, and SDK metadata remains sanitized. Complete typed source
payloads and exact binary artifacts are deliberately not redacted; they are
untrusted source evidence, so the secrecy guarantee does not claim that the
fixed upstream can never reflect a value it received. Argument parsing, typed
request preparation, and keyless discovery happen before the environment
variable is read. SDK client construction validates otherwise well-formed
timeout and limit values after key binding but before network access.

## Structured output envelope

Every completed machine-readable write is exactly one UTF-8 document on stdout.
The CLI finishes encoding before writing so an encoder failure can still become
a structured error. If stdout itself rejects or only partially accepts the
document, appending another document would corrupt the channel; the CLI exits
`1` and emits no replacement on stdout. An opt-in sanitized stderr diagnostic
is allowed for this channel failure.

The CLI encodes the SDK-owned JSON serialization contract directly without an
endpoint-specific conversion. Object and array order is deterministic where
the underlying SDK contract preserves order.

A structured call has this JSON shape:

```json
{
  "kind": "response",
  "operation": {
    "name": "company",
    "logical_id": "DS001-2019002",
    "physical_id": "get_company_json",
    "representation": "json"
  },
  "response": {
    "metadata": {
      "status": 200,
      "version": "http/1.1",
      "headers": []
    },
    "reply": {
      "kind": "success",
      "value": {}
    }
  }
}
```

The operation wrapper is CLI-owned. `response` is the complete typed SDK
`SourceResponse`; its metadata remains sanitized by the SDK. `SourceReply` uses
stable adjacent tagging:

- `{"kind":"success","value":...}` for the generated response type; and
- `{"kind":"status","value":...}` for the complete `StatusEnvelope`.

Generated response fields retain their source names. Additive fields flatten
into the object rather than appearing under a CLI-only extension bucket. An
absent SDK `Option` is omitted; `Some(SourceValue::null())` remains an explicit
null. `SourceStatus` is its exact source string. Known HTTP versions use stable
lowercase protocol strings such as `http/1.1`; an `Other` version preserves the
SDK's stored string. Each sanitized response header is an object with `name`
and `value`, where `value` is the exact byte array so non-UTF-8 evidence is not
lost.

`SourceValue` renders as null, Boolean, number, string, array, or object. A
number uses the exact JSON lexeme retained by the SDK, including arbitrary-size
integer, decimal, and exponent forms accepted by the SDK's JSON grammar. The
encoder must not narrow it through a Rust integer or floating-point type,
canonicalize its spelling, quote it, or round it. The SDK's fallible public
constructor rejects a spelling outside its documented JSON-number grammar, so
an invalid number cannot reach CLI encoding. `output_encode` remains a
fail-closed invariant error if a supported encoder cannot emit an SDK-valid
lexeme.

The CLI does not truncate fields, select a default subset, calculate aggregates,
or reinterpret source pagination. An agent that needs less data uses the
operation's source parameters; an agent that needs an artifact redirects
stdout explicitly.

## Source statuses and errors

A decoded SDK status is still a source response, not a CLI error document. It
uses `kind: response`, preserves metadata and the full status evidence, and
exits `1`. This applies to every status-only reply, including `000`, `013`,
other known codes, and unknown future strings. Only `SourceReply::Success`
exits `0` for a structured API call.

Failures that do not produce a typed source reply use a CLI error envelope:

```json
{
  "kind": "error",
  "error": {
    "code": "missing_api_key",
    "message": "OPENDART_API_KEY is required",
    "help": ["Set OPENDART_API_KEY and retry the same command"]
  }
}
```

`code` is a stable machine identifier. `message` is sanitized and actionable.
`help` is omitted when no safe deterministic next action exists. Optional error
response `metadata` appears beside `error` only when the SDK observed it.
Other optional context uses repository-owned values only; raw `clap`, `reqwest`,
filesystem, or parser errors never cross the interface. Every encodable usage
or execution error uses JSON. Failure of stdout itself is the sole case in which
the process cannot return this envelope on stdout.

The initial stable error-code inventory is:

- usage: `invalid_invocation`, `invalid_request`;
- credential and client setup: `missing_api_key`,
  `invalid_client_configuration`, `client_initialization`;
- transport: `transport_timeout`, `transport_connection`, `transport_body`,
  `transport_protocol`, `transport_other`;
- structured response: `body_limit`, `malformed_envelope`, `response_decode`;
  and
- local output and invariants: `executable_resolution`, `output_encode`,
  `destination_exists`, `artifact_limit`, `artifact_io`,
  `sdk_contract_mismatch`.

Binary stream transport failures use the matching `transport_*` code. New codes
may be additive; changing the meaning or exit class of an existing code is a
breaking CLI change.

## Binary replies

A ZIP operation requires a destination path that does not exist. The CLI writes
into a temporary file in the destination directory, streams every SDK body
chunk once, and publishes without overwriting another path.

The structured reply preserves the SDK classification:

- `archive` writes the exact body, returns the final path and byte count, and
  exits `0`;
- `status` publishes no final file, retains the complete source status and
  metadata, and exits `1`; and
- `unrecognized` writes the exact replayed body, reports its path and byte
  count, and exits `1`.

Binary calls use the same outer `kind: response`, operation, and metadata shape
as structured calls, with `operation.representation` set to `zip`. The reply
mirrors `BinaryReply`, replacing only the consumed stream with an artifact
reference:

```json
{
  "kind": "archive",
  "value": {
    "path": "./corp-code.zip",
    "bytes": 1234
  }
}
```

`unrecognized` has the same artifact value shape. `status` instead places the
complete SDK `StatusEnvelope` in `value` and publishes no destination.

Every failure before publication—including transport or connection, timeout,
stream, filesystem, and no-clobber publication failures—exits `1`, reports any
already-safe response metadata, removes its owned temporary file, and never
publishes a partial destination. The CLI does not open or validate archive
entries.

Every binary call has a 512 MiB (`536870912` byte) default budget and accepts a
positive `--artifact-limit-bytes` override. The inclusive limit applies to
bytes yielded by the SDK body stream: exactly the limit succeeds, while the
first chunk that would exceed it fails before those bytes are written.
`Content-Length` may reject early but cannot replace counted streaming.
Overflow removes the owned temporary file, publishes no destination, exits
`1`, and uses stable error code `artifact_limit`.

Artifact publication is the binary commit point. The CLI first finishes encoding
the artifact report in memory, then publishes the complete no-clobber destination
and writes the prepared report to stdout. If that later stdout write fails, the
valid final artifact remains at the requested path and the process exits `1`;
removing an already published artifact would create a worse race for the caller.
A source `status` removes the temporary file and publishes no final destination.

## Channels and exit codes

- stdout: structured home, discovery, source responses, artifacts, errors, and
  actionable help fields;
- stderr: opt-in safe diagnostics only, never required agent information;
- exit `0`: home, discovery, help/version, typed success, or archive success;
- exit `1`: source status, transport/decode/configuration/write failure, or
  unrecognized binary content; and
- exit `2`: unknown or invalid invocation, including missing flags and
  SDK-backed request-preparation violations.

Progress text and dependency diagnostics never mix into stdout. Unknown flags
include the valid flags or a precise replacement hint so an agent can correct
the invocation without a second discovery call.

## Compatibility

The binary and its stdout schema follow CLI SemVer independently from the SDK.
Canonical command renames, required flag changes, output field removal or
meaning changes, JSON encoding changes, and exit reclassification are CLI
breaking changes. Additive operations and response fields follow the SDK's
compatibility classification but still require CLI generation and contract
review.

Any SDK response-serialization or reply-variant change triggers explicit CLI
review even when Cargo considers the dependency update compatible. Human help
prose and examples may change without a version impact when command facts remain
unchanged.

Each CLI release exact-pins its reviewed `opendart` version and packages the
reviewed lockfile. Every SDK version-bump change updates the same-workspace CLI
path-and-version pin and lockfile before merge, even when no CLI release is
planned; that alignment alone does not make the CLI release-eligible. Registry
packaging of the aligned CLI waits for that SDK version to be published and
verified. The reproducible source-install command is
`cargo install --locked opendart-cli --version x.y.z`; an unlocked install
may resolve newer compatible transitive dependencies and is not release-equivalent,
as described by Cargo's
[install lockfile behavior](https://doc.rust-lang.org/cargo/commands/cargo-install.html#dealing-with-the-lockfile).

The initial crates.io source-install support matrix is Linux, macOS, and
Windows. A CLI release must install its packaged crate with
`cargo install --locked` and verify version, keyless discovery, process and
error behavior, Unicode paths, TLS, and atomic no-clobber writes natively on
each operating-system family. This source support promise does not imply that
every Rust target, architecture, libc, or OS version is supported. Exact
prebuilt targets, minimum OS versions, archive formats, and signing policy
remain later decisions.
