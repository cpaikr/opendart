# Harden prebuilt OpenDART CLI releases

## Outcome

Distribute reviewed `opendart` binaries through GitHub Releases with an
evidence-backed target policy, portable OpenDART TLS behavior, verifiable
provenance, and clean installation checks. Prebuilt artifacts complement the
crates.io source package without weakening its release isolation.

## Current state

- ADR 0003 selects crates.io as the initial CLI distribution and deliberately
  defers prebuilt artifacts to this task.
- The CLI is implemented and package-ready but not yet published. Its native
  Linux, macOS, and Windows source-install gates pass, but no registry-installed
  consumer evidence exists yet.
- Source-install verification does not by itself promise a prebuilt target,
  architecture, libc, minimum OS, or archive format; those choices still need
  evidence from actual user and agent environments.
- The SDK currently selects native TLS because the fixed OpenDART origin's
  compatibility requirements are not satisfied by the selected Rustls path.
  Prebuilt Linux portability therefore cannot be assumed from a successful
  GitHub-hosted build.
- The SDK source-package pipeline is implemented in its pending setup change.
  CLI source publication and every binary artifact upload remain unauthorized.
- The source-package plans now define an automated, component-isolated
  publication and draft-finalization pattern. This task may reuse its exact
  Release Please identity and interrupted-run invariants, but binary builders,
  signing, provenance, and upload authority remain a separate workflow and
  environment with no crates.io credential.

## Scope

- Select supported operating-system, architecture, libc, and minimum-platform
  targets from actual user and agent environments.
- Prove TLS, DNS, certificate-store, timeout, streaming, and exact-byte behavior
  on every advertised target. Do not change TLS backends merely to simplify
  packaging without renewing the SDK compatibility gate.
- Build from the immutable CLI component tag and reviewed source revision with
  pinned toolchains and locked dependencies.
- Publish deterministic archives, SHA-256 checksum manifests, software bills of
  materials, signed attestations, and build provenance. Document any remaining
  source of non-reproducibility explicitly.
- Keep artifact upload, signing identity, and release finalization in a
  least-privilege job that consumes verified outputs and supports interrupted
  release recovery without overwriting mismatched artifacts.
- Verify each archive in a clean matching environment: unpack, inspect version,
  run keyless discovery, and validate dynamic-library requirements. Keep any
  explicitly authorized smoke call in a separate, non-blocking post-release
  path so archive publication never receives the OpenDART credential or depends
  on volatile upstream health.
- Evaluate `cargo-binstall`, an installer, and package-manager metadata only
  after the direct archives have a stable naming and provenance contract.

## Validation

- Target-matrix fixtures demonstrate OpenDART connectivity and exact binary
  streaming without retry, redirect, proxy, or automatic decoding drift.
- Rebuilding the same tag either reproduces artifacts or produces documented,
  reviewable differences.
- Checksums, attestations, SBOMs, archive contents, executable version, Git tag,
  and source revision agree before a release becomes final.
- Pull requests and ordinary branch builds have no signing or release-upload
  authority.

## Next action

After the crates.io-installed CLI has real consumers, collect their target and
installation requirements and run a compatibility spike for the highest-value
Linux, macOS, and Windows targets before selecting a release matrix.
