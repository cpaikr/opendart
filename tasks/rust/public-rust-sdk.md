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
| Implementation | The transport-independent response contract is being revised before a new candidate |
| Delivery | `0.1.0-beta.1` is superseded; its publication stopped before `cargo publish` |
| Public availability | No accepted registry artifact, SDK tag, or published SDK release has been verified |
| Scheduling | Release recovery is deferred by [the Rust roadmap](../../ROADMAP_RUST.md) |

The merged Release Please proposal advanced the SDK manifest, changelog, lock
files, and release manifest to `0.1.0-beta.1`. Those repository changes are
proposal state, not evidence that the crate was published.

The interrupted draft targets candidate commit
`157d78aa62bace4b00df6677bc3372baf88b9281`. Candidate verification passed,
then publication stopped before the registry accepted an artifact. That exact
candidate is now a fail-closed recovery tombstone: automation verifies that it
remains an untagged draft at the recorded commit and does not recover it. A new
Release Please proposal must produce the replacement beta candidate.

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

1. Complete and review the breaking response-contract refactor, then let
   Release Please propose a new beta candidate.
2. Revalidate the new draft, version, candidate commit, environment, registry,
   and tag state. Stop on any mismatch.
3. Verify the accepted beta artifact, revoke bootstrap authority, and complete
   the selected trusted-publishing cutover.
4. Publish and verify a non-prerelease `0.1.0` through a reviewed,
   release-eligible commit.
5. Use that verified non-prerelease SDK as the publication gate for the public
   CLI.

## Next action

Take no registry or release action while this task is deferred. Complete the
replacement contract and review the resulting Release Please beta proposal;
the superseded beta.1 draft is not an authorized recovery source.

## Canonical references

- [Rust roadmap](../../ROADMAP_RUST.md)
- [SDK public contract](../../docs/rust-sdk/public-contract.md)
- [Generation pipeline](../../docs/rust-sdk/generation.md)
- [Transport and safety](../../docs/rust-sdk/transport-and-safety.md)
- [SDK verification and release](../../docs/rust-sdk/verification-and-release.md)
- [Repository release policy](../../RELEASING.md)
- [CLI delivery plan](../../plans/rust/public-opendart-cli.md)
