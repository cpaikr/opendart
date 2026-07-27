# Rust SDK verification and release

Planning source: [Public Rust SDK](../../tasks/rust/public-rust-sdk.md).

## Verification boundary

Rust verification extends the existing credential-free repository gate without
weakening specification, workflow-permission, or immutable-release checks.
The workflow may fetch locked registry dependencies; every build, test,
generation, documentation, and package command then runs offline.

### Repository gate

The Go verifier requires:

- complete canonical-to-generated physical-operation coverage;
- stable logical-operation pairing, response routing, and request vectors;
- deterministic generated source and ownership markers;
- approved Cargo package metadata, tracked package inventory, and source
  provenance matching the committed canonical bundle;
- aligned crate and workspace-lock versions; and
- the exact approved CI and Release Please policy.

It does not replace Cargo compilation or tests and never contacts OpenDART.

### Cargo gates

The pinned stable toolchain formats handwritten Rust and runs all-features and
no-default-features Clippy and tests, rustdoc with warnings denied, the reqwest
feature-unification fixture, and `cargo package`. The declared MSRV independently
runs all-features and no-default-features checks plus locked metadata.
Generated Rust uses generator-owned compact formatting; offline freshness is
its exact formatting gate, while every Cargo compile gate still includes it.

The SDK and CLI package file lists must exactly match
`sdk/rust/package-files.txt` and `sdk/rust/opendart-cli-package-files.txt`.
They require their reviewed generated source, public tests and output fixtures,
normalized manifests, workspace lock, and Cargo-generated provenance while
excluding generator source, repository-private inputs, credentials, and local
artifacts. Workspace packaging verifies the unpublished local dependency
relationship without granting publication authority.

The default client integration suite uses loopback-only fixtures. It proves the
one-interaction contract under dependency feature unification: no automatic
retry, redirect, ambient proxy, alternate TLS backend, alternate DNS resolver,
or response-content decoding. No OpenDART credential is used.

The CLI process suite additionally covers exact structured output fixtures,
malformed and incomplete responses, timeout controls, binary publication, and
broken stdout. Live CLI smoke coverage is opt-in and reads the credential only
when both `OPENDART_LIVE_TESTS=1` and `OPENDART_API_KEY` are present; it makes
only reviewed read-only calls and asserts structure rather than business data.
The protected workflow compiles both live runners before exposing the key; the
credential-bearing step executes only those reviewed binaries, so dependency
build scripts never inherit the credential.

The no-default-features graph must remain free of `reqwest`, Tokio, Hyper, TLS,
proxy, DNS, and streaming-runtime dependencies.

## Product versions and provenance

The repository has independent release components:

1. The canonical bundle uses root `vX.Y.Z` tags.
2. The `opendart` crate uses `opendart-vX.Y.Z` tags and Rust API SemVer.
3. The `opendart-cli` crate uses `opendart-cli-vX.Y.Z` tags and CLI contract
   SemVer.

They may release from the same commit but do not share a version by rule. A
crate package identifies:

- its Cargo version;
- its exact Git source revision through `.cargo_vcs_info.json`;
- the selected semantic specification source release, when applicable;
- the independently selected canonical bundle SHA-256;
- the generator schema; and
- the deterministic SDK projection SHA-256.

Generated freshness uses the SDK projection. A specification change outside
that projection does not rewrite or release the crate. The release guard proves
that the selected source tag exists and contains the canonical specification
inputs; generic verification does not freeze the current source tree to that
older tag. Full bundle provenance changes only when a crate release deliberately
selects a new generated artifact; that bundle need not be byte-identical to the
bundle originally generated at the source tag.

## Rust compatibility

Below `1.0.0`, compatible fixes are patches, compatible public additions are
minors, and breaking changes are majors. At and after `1.0.0`, use standard
SemVer. Examples:

| Change | Rust impact |
| --- | --- |
| Documentation or private refactor with identical output | None |
| Internal client fix with unchanged public contract | Patch |
| New operation, optional input, or open status constant | Minor |
| Required input, serialization change, public rename/removal, or narrowed wire type | Major |
| MSRV or guaranteed transport behavior change | Compatibility review |

Generated changes are not automatically compatible. Review public Rust API and
request behavior, not only the source OpenAPI label.

## Release Please boundary

The specification component is rooted at `openapi/generated`, so repository and
Rust commits cannot create a specification release. Separate Rust-aware
components are rooted at `sdk/rust/crates/opendart` and
`sdk/rust/crates/opendart-cli`; each owns its `Cargo.toml`, `CHANGELOG.md`,
component-qualified tags, and matching workspace-lock entry. The SDK component
also owns the matching `opendart` entry in the isolated reqwest compatibility
lock and updates the CLI's marked exact local SDK pin without bumping the CLI
version or changelog. The lockfile selectors use Release Please's tagged TOML
scalar values (`name.value`) so array-of-table package matches update the
intended entry instead of silently matching nothing; the release guard pins
that behavior for both components and both SDK locks.

Before their first releases, both Rust paths are intentionally absent from
`.release-please-manifest.json`. The repository guard admits an SDK manifest
entry only for the aligned first beta and continues to reject the CLI entry
until work 9 supplies its complete publication and recovery flow.

The setup workflow creates separate component proposals, finalizes
specification assets, and connects only the SDK path to the protected reusable
crate workflow. The CLI component remains configured for future ownership but
its own path is excluded from proposal eligibility until work 9. Path-qualified
outputs for one component never authorize another component's publication.

## Work 6: crates.io publication

Publication automation is active on `main` as ordinary reviewed code; landing
the setup did not merge a Rust Release Please PR or publish a crate. Branch
protection and the protected bootstrap environment are configured. The exact
SDK beta proposal must still regenerate with a matching workspace lock, pass
its exact-SHA gates, and be separately reviewed and confirmed. The setup
implements:

1. Separate Release Please PRs per manifest component. Set
   `bump-minor-pre-major` and `bump-patch-for-minor-pre-major` to `false` for
   both Rust components so their configured pre-1.0 behavior matches the
   documented patch/minor/major policy; retain the specification's independent
   settings. Configure only the SDK component with `versioning: prerelease`,
   `prerelease: true`, `prerelease-type: beta`, and temporary package-scoped
   `release-as: 0.1.0-beta.1`. The override prevents accumulated breaking
   bootstrap commits from implying `1.0.0-beta`; it does not pre-seed the
   released-version manifest. Combined PR #32 was closed without merging, so
   the setup push can create or update independent proposals on `main`.
   The release skill must still review and confirm the exact beta before merge.
2. The release guard and mutation tests enforce an exact allowlist for the
   orchestrator, reusable crate-release workflow,
   component outputs, job order, permissions, environment, action pins,
   commands, and recovery rules. Continue rejecting registry credentials in
   pull requests, generic verification, other components, and untrusted refs.
   Because a pull request created through `GITHUB_TOKEN` does not create a new
   `pull_request` workflow run, a separate dispatcher job has only
   `actions: write`, `contents: read`, and `pull-requests: read`. It dispatches
   Verify and full-race for every managed component PR's exact head SHA,
   requires each run to check out and attest that expected SHA, and
   redispatches after each update. GitHub also suppresses `workflow_run` events
   from these `GITHUB_TOKEN` dispatches, so a second trusted orchestrator job
   uses only `actions: read`, `contents: read`, `pull-requests: read`, and
   `statuses: write`. It waits for exactly one new GitHub Actions-created Verify
   and full-race run at the proposal SHA, then revalidates the proposal, current
   `main` ancestry, and exact generated-file scope. The required `verify` status
   succeeds only when both runs succeed. Verify and full-race remain
   credential-free. Extend the reporter job's allowlist and releaseguard
   invariant before enabling another component or changing the generated
   release-file set.
   After the setup reaches `main`, add the stable `Verify / verify` aggregate
   to branch protection. Keep
   full-race release-only; its exact-SHA result is folded into the proposal's
   required `verify` status. Do not make every ordinary PR run the full sweep.
3. The Release Please job consumes the exact
   `sdk/rust/crates/opendart--release_created`, `--tag_name`, `--version`, and
   `--sha` values. Before invoking Release Please, query the exact
   `opendart-vX.Y.Z` release identity. With the configured draft release and no
   forced tag creation, recover a draft only when its tag name and full-SHA
   `targetCommitish` equal the expected identity and reviewed candidate SHA and
   that SHA is an ancestor of current `main`. A recovered draft is equivalent
   to matching action outputs and keeps its candidate SHA even when a newer
   repaired workflow resumes it. For a published release, resolve the Git tag:
   equality with the current event SHA is an idempotent completion, while a
   verified ancestor is the normal previous release and permits Release Please
   to process later commits. A branch-shaped draft target, tag-only state,
   multiple match, non-ancestor, or other mismatch stops without side effects.
4. A component-parameterized Rust crate workflow is called only for that exact SDK
   output. The later CLI work may reuse the mechanics, but it must supply its
   own path, tag, environment, package inventory, and consumer checks.

### Automated stage order

After the maintainer reviews and merges the SDK Release Please PR, the workflow
owns the remaining work:

1. **Verify candidate without credentials.** Check out the exact output or
   recovered SHA with persisted credentials disabled, require the complete
   repository verification gate, reproduce the tracked SDK inventory, run
   `cargo package --locked` and `cargo publish --locked --dry-run`, and retain
   the candidate `.crate`, checksum, version, tag, revision, and inventory as
   immutable workflow evidence. Use attempt-qualified artifact names and record
   the upload digest so a rerun cannot overwrite or silently substitute earlier
   evidence.
2. **Acquire publication authority narrowly.** Enter the component-specific
   `crates-io-opendart` environment only after candidate verification. The job
   receives repository read authority and exactly one crates.io authentication
   path; it receives no GitHub release-write authority. Check out the same
   immutable revision and verify the candidate evidence digest before invoking
   Cargo. Recheck the package name, expected owner when one exists, and exact
   version immediately before using authority. An unexpected owner or name
   takeover stops for a product decision.
3. **Publish at most once without exposing the token to build scripts.** If the
   exact version is absent, run one
   `cargo +1.97.1 publish --locked --no-verify` for `opendart` at the candidate
   SHA. The earlier credential-free `cargo +1.97.1 publish --locked --dry-run`
   is the build verification; `--no-verify` prevents Cargo from running that
   build again after the registry credential is present. Treat a Cargo timeout
   as indeterminate because index polling can time out after a successful
   upload; continue to reconciliation instead of retrying blindly.
4. **Reconcile the immutable registry result.** Poll with a bounded deadline,
   acquire the accepted checksum and `.crate` through repository-owned network
   logic, and invoke `opendart-tool verify-crate-artifact`. Require exact
   checksum, provenance, normalized/original manifests, tar contents, and
   reviewed inventory identity. If the version was already present, these same
   checks decide whether the run may resume.
5. **Prove the public consumer.** Build and test a clean temporary consumer
   against the exact registry version, inspect the crates.io source view, and
   wait for successful docs.rs output. Do not substitute the local path crate.
6. **Finalize separately.** Give a finalizer GitHub contents-write authority but
   no crates.io credential. It may publish only the matching draft whose tag,
   target SHA, version, component identity, and expected prerelease flag match
   the reviewed component outputs after every preceding stage has passed. The
   beta finalizer preserves `prerelease=true`;
   the stable finalizer requires `prerelease=false`. Both explicitly keep
   `latest=false` because repository-global Latest belongs to the independent
   specification convention. A mismatch or timeout leaves the draft unpublished
   for investigation, and the release guard rejects the specification
   finalizer's `--latest` command on a Rust path.

   The reusable workflow admits exactly one of these finalization forms after
   validating the component tag and all prior evidence:

   ```sh
   gh release edit "${tag}" --draft=false --prerelease=true --latest=false
   gh release edit "${tag}" --draft=false --prerelease=false --latest=false
   ```

   The first form is beta-only and the second is stable-only; the guard rejects
   either form when it disagrees with the reviewed Release Please proposal.

### First-release bootstrap and steady state

crates.io trusted publishing cannot be configured until the first `opendart`
version exists. The `crates-io-opendart` GitHub environment is configured with
a required reviewer, no protection-rule bypass, `main`-only deployments, the
exact `OPENDART_CRATES_IO_OWNER`, and one short-lived API token capable only of
creating the new crate as environment secret `CARGO_REGISTRY_TOKEN`. Enable
prevent-self-review when an
independent maintainer is available; otherwise record that the same maintainer
approved both the exact Release Please proposal and the one-time environment
deployment. The repository guard separately restricts access to the canonical
workflow. Do not store the token as a repository-wide secret or use it outside
the publication job.

The prerelease still follows the full automated stage order; the one-time
environment approval is the only step added to the normal Release Please PR
review and merge. After the accepted prerelease passes registry, consumer, and
docs.rs verification:

1. delete the bootstrap secret and revoke its token;
2. configure the crates.io trusted publisher for the exact repository,
   workflow, and `crates-io-opendart` environment;
3. replace the bootstrap authentication step with a commit-pinned
   `rust-lang/crates-io-auth-action` step and grant only `contents: read` plus
   `id-token: write` to the publication job;
4. remove the environment reviewer so later publication is automatic after the
   explicit Release Please PR merge, while retaining branch/workflow scoping;
5. keep `versioning: prerelease` and `prerelease-type: beta`, set
   `prerelease: false`, and change package-scoped `release-as` from
   `0.1.0-beta.1` to `0.1.0` so Release Please has an explicit stable-promotion
   version input. In the same reviewed delivery, add a release-eligible
   Conventional Commit under `sdk/rust/crates/opendart` that records the
   verified beta and stable support contract. Release Please skips a component
   with no user-facing path-scoped commits, so an out-of-path
   OIDC/configuration change alone cannot trigger the promotion; do not
   substitute an empty version-bump commit; and
6. rerun the no-side-effect configuration and recovery tests before allowing
   that exact stable proposal. After the stable release succeeds, remove
   `release-as`, `versioning`, `prerelease`, and `prerelease-type` so later
   releases use the default strategy.

Do not retain token fallback logic after OIDC is enabled. The two temporary
package-scoped `release-as` values are the sole planned forced-version exception
and must be removed after stable. The beta and stable versions must still be
presented and explicitly confirmed under the repository release policy before
their respective merges.

Because crates.io versions are immutable and `cargo publish` cannot accept a
prebuilt `.crate`, pre-publication review and post-publication verification are
separate required gates. Pull requests, branch builds, generator runs, and
specification-only releases must never receive registry authority.

## Adoption boundary

A strict collector uses the crate with default features disabled for generated
operation inputs, deterministic preparation and authorization, and bounded wire
inspection after its own artifact boundary. It retains HTTP execution evidence,
exact-byte storage, attempt lifecycle, retry scheduling, quotas, collection
closure, successful-empty classification, domain conversion, and persistence.

Production adoption waits for a verified crates.io version. A local path
dependency is suitable only for integration development before that registry
release.
