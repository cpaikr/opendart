# Changelog

## [0.2.0](https://github.com/cpaikr/opendart/compare/v0.1.0...v0.2.0) (2026-08-02)


### ⚠ BREAKING CHANGES

* **openapi:** Request preparation and the CLI now reject values outside curated OpenDART formats, allowed sets, and ranges; CLI discovery and error envelopes expose the corresponding stable contract details.

### Features

* **openapi:** enforce canonical request constraints ([3a8bccb](https://github.com/cpaikr/opendart/commit/3a8bccb3c27c789049c16053848a10a507171d69))
* **tooling:** cut over offline verification to Go ([534742b](https://github.com/cpaikr/opendart/commit/534742ba1a63f608c8810ccc421a4963576ea5ba))


### Bug Fixes

* **generator:** preserve request constraint semantics ([901cfed](https://github.com/cpaikr/opendart/commit/901cfed722f437632d291ad9713b26324e836a6f))

## 0.1.0 (2026-07-17)


### Features

* **openapi:** strengthen OpenDART specification contract ([4e37464](https://github.com/cpaikr/opendart/commit/4e37464157f248242ca1277be66c0654cbe0096d))


### Bug Fixes

* **openapi:** address review feedback ([a644a4b](https://github.com/cpaikr/opendart/commit/a644a4ba017501348a784ca8c0b419ff2711f372))


### Documentation

* correct release ownership ([aad88df](https://github.com/cpaikr/opendart/commit/aad88df2866a1172080feeb7202c61e269ec9ba3))
