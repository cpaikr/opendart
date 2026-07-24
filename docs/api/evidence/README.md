# API Evidence Manifests

Files here are reviewed, sanitized observations produced by fixed credentialed
probes. They contain allowlisted request coordinates, response hashes and
summaries, and bounded semantic assertions. They do not contain credentials,
authenticated URLs, raw response bodies, or unrestricted headers.

Regenerate the auditor manifest from the repository root with:

```sh
./scripts/with-opendart-env -- \
  go run ./cmd/opendart-tool probe-auditor-evidence
```

Review the JSON before replacing the dated manifest. Offline repository
verification enforces the strict schema, fixed request matrix, pagination
closure, credential boundary, archive evidence, and document-match assertions.
