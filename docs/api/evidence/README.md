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

Review the JSON before adding a dated manifest. Offline repository verification
enforces the strict schema, fixed request matrix, pagination closure,
credential boundary, archive evidence, and document-match assertions.

Dated manifests are immutable once merged, and the filename is part of the
verifier contract. If a new observation supersedes the current one, add a new
dated file and update every repository reference and verifier fixture in the
same reviewed change, then run:

```sh
go run ./cmd/opendart-tool verify --repository-root .
```
