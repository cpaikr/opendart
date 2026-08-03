# Rust Roadmap

Owner: `rust`. Non-Rust planning is owned by `ROADMAP.md`; do not edit it or
its linked work items from this worktree.

This file owns Rust product priority and scheduling. Linked work items own
implementation status, blockers, and the next safe action. Placement here does
not authorize a registry, release, secret, or workflow side effect.

## Current

- [Decouple the CLI presentation projection from Rust SDK symbols](plans/rust/cli-presentation-projection.md)
  is active after the contract and conformance gate completed review and merged
  in PR #74. The generated SDK remains the sole runtime conformer while this
  result separates public CLI grammar and discovery from private typed dispatch.

## Plans

1. [Decouple the CLI presentation projection from Rust SDK symbols](plans/rust/cli-presentation-projection.md)
   implements the reviewed CLI-owned grammar and discovery contract while the
   generated SDK remains the sole runtime conformer behind private dispatch.
2. [Implement and cut over the handwritten Rust SDK conformer](plans/rust/handwritten-sdk-conformer.md)
   owns the complete transition from the generated public implementation to one
   handwritten product path, including private CLI dispatch retargeting, SDK
   generator deletion, active-documentation reconciliation, and offline package
   qualification.
3. [Publish and adopt the public Rust SDK](plans/rust/public-rust-sdk.md) remains
   the enabling delivery item after the handwritten package passes its complete
   release gate.
4. [Publish and adopt the public agent-first OpenDART CLI](plans/rust/public-opendart-cli.md)
   remains blocked on a verified non-prerelease `opendart` registry release.

The list is the intended work order. Capturing these plans does not start them
or authorize registry, release, secret, or workflow effects. The interrupted
beta.1 candidate remains a fail-closed tombstone and cannot be recovered. A new
SDK release candidate may be proposed only after the handwritten-only product
state is complete and reviewed.

## Tasks

- [Harden prebuilt OpenDART CLI releases](tasks/rust/opendart-cli-prebuilt-releases.md)
