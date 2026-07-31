# Public Rust SDK

Status: `deferred`

Planning scope: `rust`

## Outcome

Publish `opendart` as the first-party Rust SDK for the canonical OpenAPI
contract.

## Current state

| Area | State |
| --- | --- |
| Product design | Accepted in [ADR 0002](../../docs/decisions/0002-public-rust-sdk.md) |
| Implementation | Package-ready; generation, transport, and verification are implemented |
| Delivery | The beta candidate was verified, but publication stopped before `cargo publish` |
| Public availability | No accepted registry artifact, SDK tag, or published SDK release was verified on 2026-07-31 |
| Scheduling | Release recovery is deferred by [the Rust roadmap](../../ROADMAP_RUST.md) |

The merged Release Please proposal advanced the SDK manifest, changelog, lock
files, and release manifest to `0.1.0-beta.1`. Those repository changes are
proposal state, not evidence that the crate was published.

The interrupted draft targets candidate commit
`157d78aa62bace4b00df6677bc3372baf88b9281`. Candidate verification passed,
then the publication job observed an empty registry token and stopped before
publishing. Static environment binding has since landed, but no successful
registry reconciliation has established publication.

Draft releases, duplicate proposals, environment configuration, and secret
availability are external and volatile. Revalidate all of them before acting;
do not infer current release state from this dated record.

## Boundaries

- The OpenAPI document remains the protocol authority. Generated Rust is
  checked in and must be regenerated rather than hand-edited.
- The SDK owns typed protocol access, response validation, and transport
  behavior. Application policy remains outside the SDK.
- Normal generation and verification are credential-free.
- Registry acceptance plus the matching published tag and GitHub release are
  publication evidence. A manifest version or draft alone is not.

## Remaining delivery work

1. Resolve the mismatch between roadmap deferral and the main-push workflow,
   which can still inspect and recover the interrupted SDK draft.
2. If recovery is explicitly resumed, revalidate the exact draft, version,
   candidate commit, environment, registry, and tag state. Stop on any
   mismatch.
3. Verify the accepted beta artifact, revoke bootstrap authority, and complete
   the selected trusted-publishing cutover.
4. Publish and verify a non-prerelease `0.1.0` through a reviewed,
   release-eligible commit.
5. Use that verified non-prerelease SDK as the publication gate for the public
   CLI.

## Next action

Take no registry or release action while this task is deferred. Before the
next delivery to `main`, maintainers must choose one of two implementation
paths:

- keep recovery deferred and remove or disable the workflow authority that can
  resume it; or
- explicitly resume recovery of the exact verified candidate under the
  repository release procedure.

That operational choice requires a non-documentation change and is not resolved
by roadmap wording alone.

## Canonical references

- [Rust roadmap](../../ROADMAP_RUST.md)
- [SDK public contract](../../docs/rust-sdk/public-contract.md)
- [Generation pipeline](../../docs/rust-sdk/generation.md)
- [Transport and safety](../../docs/rust-sdk/transport-and-safety.md)
- [SDK verification and release](../../docs/rust-sdk/verification-and-release.md)
- [Repository release policy](../../RELEASING.md)
- [CLI delivery plan](../../plans/rust/public-opendart-cli.md)
