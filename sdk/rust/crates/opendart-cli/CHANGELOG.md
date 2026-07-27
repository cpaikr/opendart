# Changelog

## 0.1.0 (2026-07-27)


### ⚠ BREAKING CHANGES

* **openapi:** Request preparation and the CLI now reject values outside curated OpenDART formats, allowed sets, and ranges; CLI discovery and error envelopes expose the corresponding stable contract details.

### Features

* **cli:** make artifact publication transactional ([06dc2f6](https://github.com/cpaikr/opendart/commit/06dc2f61f6d1fd4087b0a9449e4d4e7c8af65561))
* **cli:** make artifact publication transactional ([c0908f9](https://github.com/cpaikr/opendart/commit/c0908f94cdfdcb29709294e92b189aa39616a292))
* **openapi:** enforce canonical request constraints ([3a8bccb](https://github.com/cpaikr/opendart/commit/3a8bccb3c27c789049c16053848a10a507171d69))
* **rust:** add generated CLI discovery ([0aeeb92](https://github.com/cpaikr/opendart/commit/0aeeb922f2c9b3ec2ef575c4604aa5878d5138e6))
* **rust:** add generated CLI discovery ([d1cb6e7](https://github.com/cpaikr/opendart/commit/d1cb6e75e0163e2f27a0e497c87212bfcb56f05c))
* **rust:** execute structured CLI calls ([a85b11f](https://github.com/cpaikr/opendart/commit/a85b11f13b6cd22570768c0cfee36219e93e5c10))
* **rust:** execute structured CLI calls ([2865420](https://github.com/cpaikr/opendart/commit/28654200fd30efd9a951ec1bd5772199a8a40243))
* **rust:** prepare CLI release ownership ([06862ec](https://github.com/cpaikr/opendart/commit/06862ec99c4573bd5083c6a413c979b2a5d7232d))
* **rust:** prepare CLI release ownership ([cccd894](https://github.com/cpaikr/opendart/commit/cccd894ad352f1ccfb507fd8fb19d9e5e50bac63))
* **rust:** prepare the CLI source package ([53c80ef](https://github.com/cpaikr/opendart/commit/53c80efccef994fb398db8298a2f3fda6f53d53c))
* **rust:** preserve binary CLI artifacts ([95115aa](https://github.com/cpaikr/opendart/commit/95115aa79b55bd61f49e7ee37ce048cf118f8d9d))
* **rust:** preserve binary CLI artifacts ([4dd6293](https://github.com/cpaikr/opendart/commit/4dd629326f618d7ac9fd2223e251d2cd458c1c37))
* **rust:** promote wire-boundary and CI hardening ([f41e467](https://github.com/cpaikr/opendart/commit/f41e46735dbb3188a637a1283ac2f1e3ab52b29d))


### Bug Fixes

* **cli:** close command and dispatch contract gaps ([#56](https://github.com/cpaikr/opendart/issues/56)) ([14eb163](https://github.com/cpaikr/opendart/commit/14eb1635a7dce01cd7faf0a1913805773ff45946))
* **cli:** make output and home discovery portable ([8a12da8](https://github.com/cpaikr/opendart/commit/8a12da871b0fbac1e2ed382337fd3b7b067ebfee))
* **cli:** preserve nested command and hyphen value contracts ([f9a154c](https://github.com/cpaikr/opendart/commit/f9a154c810f82f97c2011e5abc5c84bf0f0079b7))
* **cli:** preserve nested command context ([aa49880](https://github.com/cpaikr/opendart/commit/aa49880d33846a5cefe0957de8d0845b7182623f))
* **cli:** preserve option termination during normalization ([200341c](https://github.com/cpaikr/opendart/commit/200341c1a53cb9d86998deee844da4407494a61b))
* **cli:** preserve published SDK compatibility ([e6d1439](https://github.com/cpaikr/opendart/commit/e6d1439060123d50e6ea2af8be041b4d767f74d7))
* **cli:** publish artifacts from retained identity ([da3c2d9](https://github.com/cpaikr/opendart/commit/da3c2d909ccf9858f9dd9841d8bc007d5ecbce4b))
* **cli:** synchronize structured usage diagnostics ([afde06a](https://github.com/cpaikr/opendart/commit/afde06a884febd437463cc1f3f7eee38c09a23e8))
* **generator:** preserve request constraint semantics ([901cfed](https://github.com/cpaikr/opendart/commit/901cfed722f437632d291ad9713b26324e836a6f))
* **rust:** clarify CLI execution boundaries ([12898fd](https://github.com/cpaikr/opendart/commit/12898fd9940a488ea388f2bdcf512a985814a73e))
* **rust:** classify CLI path resolution failures ([bbfd9a4](https://github.com/cpaikr/opendart/commit/bbfd9a41a4e7cdb43a8ed9bcc55f059bf85728c2))
* **rust:** classify malformed CLI credentials ([74668f1](https://github.com/cpaikr/opendart/commit/74668f1b5392ea702197fd7572f166b986267339))
* **rust:** close CLI completion audit gaps ([b9c92cf](https://github.com/cpaikr/opendart/commit/b9c92cf7e0d4b5e90183f606c169b3bfd33cf34d))
* **rust:** correct CLI live smoke assertions ([edc2f2f](https://github.com/cpaikr/opendart/commit/edc2f2f7ffc2c0e0ffd5b32c80339f09508dac9b))
* **rust:** fail closed on encoded response metadata ([010e47d](https://github.com/cpaikr/opendart/commit/010e47da7935fd2f1582986a7668e88256cb4515))
* **rust:** harden portability and output boundaries ([7df7111](https://github.com/cpaikr/opendart/commit/7df7111e91bed3f6b07f374c4bbc459daacc44e8))
* **rust:** harden SDK wire trust boundaries ([750ba04](https://github.com/cpaikr/opendart/commit/750ba047b5610a99e4e1e512da800bcd9302f132))
* **rust:** keep generated CLI identities singular ([bdab2df](https://github.com/cpaikr/opendart/commit/bdab2df001d95879dadeeb556ea690bab0160d7f))
* **rust:** reject mixed CLI output contracts ([033336c](https://github.com/cpaikr/opendart/commit/033336cc69e973d0aabedc3b26931aac76a2da0b))
* **rust:** scope CLI discovery metadata ([811f44c](https://github.com/cpaikr/opendart/commit/811f44c3f91db533de04d9c42a90826af4ba57e8))
* **rust:** tolerate expected overflow disconnects ([e02d01f](https://github.com/cpaikr/opendart/commit/e02d01f6982df6f1cc5d1392eb2224dc70120a04))
* **rust:** validate artifact parents portably ([1342f04](https://github.com/cpaikr/opendart/commit/1342f0404885e438bc6961177517c3d29a29c51f))
* **rust:** validate XML before status classification ([44052ce](https://github.com/cpaikr/opendart/commit/44052cea25cc0eaf050cd60126497785c64c711a))

## Changelog

Release Please maintains the released `opendart-cli` history in this file.
