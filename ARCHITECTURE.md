# Repository architecture

## Purpose and boundary

This repository publishes a source-backed OpenAPI description of OpenDART. The
released bundle is the product. Repository tooling exists only to acquire,
generate, validate, test, and release that specification; it is not a public
application, SDK, or reusable package.

The official development guide is authoritative for documented behavior.
Authenticated API observations are evidence about the live service and must
remain distinguishable from guide-sourced facts.

## System shape

```text
OpenDART guide -> source acquisition -> normalized catalog -> OpenAPI fragments
                                                          -> portable bundle

committed OpenAPI -> offline verification ----------------> release gate
OpenDART guide    -> semantic drift check ----------------> sanitized issue
OpenDART API      -> credentialed conformance check ------> sanitized issue
```

The drift and conformance paths are independent. Neither may edit the
specification, create a pull request, or publish a release.

## Major components

- `openapi/openapi.yaml` is the canonical multi-file entry point. Path and
  schema fragments below `openapi/` are generated, reviewed artifacts.
- `openapi/generated/openapi.bundle.yaml` is the portable release interface.
- `scripts/` contains the current Node.js extraction, validation, and probe
  implementation. It remains authoritative until each command is replaced and
  verified by the Go migration.
- `.github/workflows/verify.yml` is the credential-free repository gate.
  Release automation consumes that gate before publishing the bundle.
- `docs/plans/` separates the Go migration, public-guide drift detection, and
  credentialed live conformance into reviewable work streams.

The target tooling is one repository-internal Go module with one CLI and a small
set of deep internal packages. Workflow YAML should orchestrate commands and
security boundaries; parsing, comparison, validation, sanitization, and policy
belong in tested Go code.

## Dominant flows

### Specification refresh

The acquisition layer fetches only trusted guide pages and converts them into
a normalized catalog. The specification layer renders a complete staging tree,
validates ownership, references, catalog invariants, and OpenAPI semantics, and
then atomically publishes the generated fragments. Bundling is derived from the
committed multi-file description.

### Scheduled observations

The credential-free drift job regenerates a candidate with baseline snapshot
metadata and compares specification meaning. Page markup or formatting that
normalizes to the same OpenAPI contract is unchanged.

The credentialed conformance job executes every physical OpenAPI operation
using committed public inputs. It validates representation and endpoint
content, produces an allowlisted report, and discards raw responses. Separate
notification jobs create or update separate drift and conformance issues.

## Invariants

- OpenAPI 3.2 is canonical; compatibility artifacts must not weaken it.
- Generated OpenAPI files are changed through the generator, never by hand.
- Guide-sourced facts, empirical constraints, and executable test scenarios
  remain separate and traceable.
- A refresh publishes a complete validated catalog or leaves the prior catalog
  untouched.
- Offline verification performs no OpenDART network requests and needs no API
  key.
- The API key is available only to trusted default-branch conformance code and
  never to issue-writing jobs.
- Reports and issues contain only allowlisted evidence. They never contain the
  key, authenticated URLs, or unrestricted response bodies.
- Scheduled automation reports evidence only. Specification changes always
  require a reviewed human-authored change.

## Start here

- [`README.md`](README.md) describes the artifacts and current commands.
- [`docs/decisions/0001-go-repository-tooling.md`](docs/decisions/0001-go-repository-tooling.md)
  records the language and tooling decision.
- [`docs/plans/go-tooling-migration.md`](docs/plans/go-tooling-migration.md)
  owns the Node-to-Go migration.
- [`docs/plans/guide-drift.md`](docs/plans/guide-drift.md)
  owns public-guide semantic drift detection.
- [`docs/plans/live-conformance.md`](docs/plans/live-conformance.md) owns
  weekly live endpoint conformance.
