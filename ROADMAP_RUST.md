# Rust Roadmap

Owner: `rust`. Non-Rust planning is owned by `ROADMAP.md`; do not edit it or
its linked work items from this worktree.

This file owns Rust product priority and scheduling. Linked work items own
implementation status, blockers, and the next safe action. Placement here does
not authorize a registry, release, secret, or workflow side effect.

## Current

_None. Rust delivery is deferred._

## Plans

1. [Publish and adopt the public agent-first OpenDART CLI](plans/rust/public-opendart-cli.md)
   is the next product outcome. Its publication work is blocked on a verified
   non-prerelease `opendart` registry release.
2. [Publish and adopt the public Rust SDK](tasks/rust/public-rust-sdk.md) is the
   enabling delivery item, but its release recovery is also deferred.

The list expresses product priority, not executable dependency order. The SDK
must be published and verified before CLI publication can start. The existing
SDK release workflow remains technically capable of inspecting an interrupted
release on a `main` push; roadmap deferral alone does not disable it. Resolve
that operational mismatch before the next `main` delivery.

## Tasks

- [Harden prebuilt OpenDART CLI releases](tasks/rust/opendart-cli-prebuilt-releases.md)
