# Rust Roadmap

Owner: `rust`. Non-Rust planning is owned by `ROADMAP.md`; do not edit it or
its linked work items from this worktree.

This file owns Rust product priority and scheduling. Linked work items own
implementation status, blockers, and the next safe action. Placement here does
not authorize a registry, release, secret, or workflow side effect.

## Current

- No authorized Rust implementation is active. The
  [handwritten Rust SDK conformer](plans/rust/handwritten-sdk-conformer.md)
  completed in PR #76 after clean exhaustive offline qualification, final-head
  CI, and independent and Codex review with no actionable findings. Public SDK
  publication remains queued but has not started.

## Plans

1. [Decouple the CLI presentation projection from Rust SDK symbols](plans/rust/cli-presentation-projection.md)
   completed in PR #75. It established reviewed CLI-owned grammar, discovery,
   and isolated dispatch before the later conformer cutover retargeted that
   dispatch without changing the public projection.
2. [Implement and cut over the handwritten Rust SDK conformer](plans/rust/handwritten-sdk-conformer.md)
   completed in PR #76. It transitioned the generated public implementation to
   one handwritten product path, including private CLI dispatch retargeting,
   SDK generator deletion, active-documentation reconciliation, and offline
   package qualification.
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
