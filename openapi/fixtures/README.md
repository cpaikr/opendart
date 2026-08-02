# Contract fixtures

`v1/manifest.json` is a small, independently maintained corpus for protocol
families that generated breadth tests cannot prove by themselves. It is not a
provider-response archive and must not contain credentials or authenticated
URLs.

Each response-case manifest entry records its body payload's provenance and
digest; body files contain only protocol response payloads. `synthetic` means
the body was handwritten to exercise a documented or historically observed
contract fact; it must never be described as a captured OpenDART response. Add
a case only when it represents a new request or response family, not another
equivalent endpoint.
