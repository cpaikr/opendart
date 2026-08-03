# Public Rust SDK

Status: `deferred`

Planning scope: `rust`

## Outcome

Publish `opendart` as the first-party Rust SDK for the canonical OpenAPI
contract.

## Current state

| Area | State |
| --- | --- |
| Product design | [ADR 0004](../../docs/decisions/0004-handwritten-rust-sdk-conformer.md) accepts one handwritten Rust conformer, Rust-native conformance, idiomatic product naming, source-shaped response serialization, and a presentation-only CLI projection while preserving ADR 0002's SDK product boundary |
| Implementation | The handwritten-only SDK, Rust-native conformance, source-backed wrappers, safe client, and CLI integration are implemented; SDK generation and generated SDK source are absent |
| Delivery | `0.1.0-beta.1` is superseded; its publication stopped before `cargo publish` |
| Public availability | No accepted registry artifact, SDK tag, or published SDK release has been verified |
| Scheduling | Implementation predecessors are complete; release recovery remains deferred until explicitly started with renewed external-state review |

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

- The OpenAPI document remains the sole protocol authority. The planned Rust
  implementation is a handwritten conformer; OpenAPI-derived generation may
  support bundles, CLI breadth, coverage checks, or test scaffolding but must
  not emit SDK request builders, validators, wire types, or response decoders.
- The SDK owns typed protocol access, response validation, and transport
  behavior. Application policy remains outside the SDK.
- Normal conformance and verification are credential-free.
- Registry acceptance plus the matching published tag and GitHub release are
  publication evidence. A manifest version or draft alone is not.

## Remaining delivery work

1. Review the resulting breaking handwritten package contract and renewed
   offline qualification evidence, then let Release Please
   propose a new beta candidate from the handwritten-only SDK revision.
2. Revalidate the new draft, version, candidate commit, environment, registry,
   and tag state. Stop on any mismatch.
3. Verify the accepted beta artifact, revoke bootstrap authority, and complete
   the selected trusted-publishing cutover.
4. Publish and verify a non-prerelease `0.1.0` through a reviewed,
   release-eligible commit.
5. Use that verified non-prerelease SDK as the publication gate for the public
   CLI.

## Next action

Take no registry or release action while this task is deferred. When release
work resumes, renew external state and qualification evidence from the complete
handwritten-only package. The superseded beta.1 draft is not an authorized
recovery source.

## Canonical references

- [Rust roadmap](../../ROADMAP_RUST.md)
- [ADR 0004: Use one handwritten Rust SDK conformer](../../docs/decisions/0004-handwritten-rust-sdk-conformer.md)
- [Handwritten SDK contract and conformance](../../plans/rust/handwritten-sdk-contract-and-conformance.md)
- [CLI presentation projection](../../plans/rust/cli-presentation-projection.md)
- [Handwritten SDK implementation and cutover](../../plans/rust/handwritten-sdk-conformer.md)
- [SDK public contract](../../docs/rust-sdk/public-contract.md)
- [CLI projection generation](../../docs/rust-sdk/generation.md)
- [Transport and safety](../../docs/rust-sdk/transport-and-safety.md)
- [SDK verification and release](../../docs/rust-sdk/verification-and-release.md)
- [Repository release policy](../../RELEASING.md)
- [CLI delivery plan](../../plans/rust/public-opendart-cli.md)
