# Changelog

## 0.1.0 (2026-07-27)


### ⚠ BREAKING CHANGES

* **openapi:** Request preparation and the CLI now reject values outside curated OpenDART formats, allowed sets, and ranges; CLI discovery and error envelopes expose the corresponding stable contract details.
* **rust:** SourceValue::number now validates JSON number lexemes and returns Result<SourceValue, InvalidSourceNumberError>.
* **rust:** replace runtime representation selection with prepare_json, prepare_xml, and prepare_zip methods, and make structured PreparedRequest generic over its generated response type.

### Features

* **cli:** make artifact publication transactional ([c0908f9](https://github.com/cpaikr/opendart/commit/c0908f94cdfdcb29709294e92b189aa39616a292))
* **openapi:** enforce canonical request constraints ([3a8bccb](https://github.com/cpaikr/opendart/commit/3a8bccb3c27c789049c16053848a10a507171d69))
* **rust:** add bounded safe-default client ([ce34e5b](https://github.com/cpaikr/opendart/commit/ce34e5b23e629c21cb821651bb0e64bfe46b940e))
* **rust:** add bounded safe-default client ([b97091a](https://github.com/cpaikr/opendart/commit/b97091a59ddc32cb51c23037189deefc1ab48465))
* **rust:** add source-faithful JSON serialization ([078f05c](https://github.com/cpaikr/opendart/commit/078f05c2fc7e6faeb5f2658abc8f9ea02e9affa6))
* **rust:** add typed response decoding ([bd39f5b](https://github.com/cpaikr/opendart/commit/bd39f5b199410d946c73d4efda48bb97026a8853))
* **rust:** close SDK package coverage ([eb54352](https://github.com/cpaikr/opendart/commit/eb5435203c0cb15b88da94a1fbcdf30d5024b767))
* **rust:** establish the transport-independent SDK core ([f8a42ce](https://github.com/cpaikr/opendart/commit/f8a42ce8fd13bacade4507ac483107c381692b6c))
* **rust:** establish the transport-independent SDK core ([00c1e6a](https://github.com/cpaikr/opendart/commit/00c1e6ab1dd98898703a8d035de5becba9233911))
* **rust:** generate the complete SDK surface ([deaddf2](https://github.com/cpaikr/opendart/commit/deaddf2473f40ebf3055f9a6e775672262cb49dc))
* **rust:** generate the complete SDK surface ([6a32738](https://github.com/cpaikr/opendart/commit/6a327380794f2851683b1ca14a8326eeac8936da))
* **rust:** package the complete public SDK ([83135cc](https://github.com/cpaikr/opendart/commit/83135cc0fd1ee9b98340f83a1ebf75fd6a0415a2))
* **rust:** promote the public SDK to main ([dd5a325](https://github.com/cpaikr/opendart/commit/dd5a3253d4c2345a8715a30ce0a08e05e32f3b14))
* **rust:** promote wire-boundary and CI hardening ([f41e467](https://github.com/cpaikr/opendart/commit/f41e46735dbb3188a637a1283ac2f1e3ab52b29d))


### Bug Fixes

* **cli:** preserve published SDK compatibility ([e6d1439](https://github.com/cpaikr/opendart/commit/e6d1439060123d50e6ea2af8be041b4d767f74d7))
* **generator:** preserve request constraint semantics ([901cfed](https://github.com/cpaikr/opendart/commit/901cfed722f437632d291ad9713b26324e836a6f))
* **provenance:** refresh canonical bundle checksum ([b0b1db2](https://github.com/cpaikr/opendart/commit/b0b1db2f0f8160325d7ea0796a3f13547c0ad5ff))
* **rust:** address SDK generator review feedback ([21ecbd9](https://github.com/cpaikr/opendart/commit/21ecbd939088cb7f3031d59d4509e96d0fdf8f22))
* **rust:** address typed response review feedback ([84d37fe](https://github.com/cpaikr/opendart/commit/84d37fe0dfe5502811825f729604c1acbad1b7cf))
* **rust:** bound array iterator consumption ([8f73f52](https://github.com/cpaikr/opendart/commit/8f73f52520f48bcbad6435f61ca147fbd1940ea3))
* **rust:** close JSON serialization review gaps ([0802912](https://github.com/cpaikr/opendart/commit/080291283b1c47a8b7c04e16390ae25351c6d7fb))
* **rust:** complete API contracts and verification ([405f5ab](https://github.com/cpaikr/opendart/commit/405f5ab1c3572faa44f3147115d1f32c6e4c5539))
* **rust:** enforce response and transport contracts ([fb7d4b5](https://github.com/cpaikr/opendart/commit/fb7d4b5df7942eb677e5ced891125ba531b94976))
* **rust:** fail closed on encoded response metadata ([010e47d](https://github.com/cpaikr/opendart/commit/010e47da7935fd2f1582986a7668e88256cb4515))
* **rust:** harden portability and output boundaries ([7df7111](https://github.com/cpaikr/opendart/commit/7df7111e91bed3f6b07f374c4bbc459daacc44e8))
* **rust:** harden SDK wire trust boundaries ([750ba04](https://github.com/cpaikr/opendart/commit/750ba047b5610a99e4e1e512da800bcd9302f132))
* **rust:** keep API contracts portable ([fa624e6](https://github.com/cpaikr/opendart/commit/fa624e68ce37fd74442c90d4007c3dab668f7b4f))
* **rust:** keep native client dependencies off WebAssembly ([39638f5](https://github.com/cpaikr/opendart/commit/39638f532aa65da22e9437262f3cda03c43845a7))
* **rust:** preserve unambiguous wire evidence ([007331c](https://github.com/cpaikr/opendart/commit/007331ca22c0acedbb89c835c306e57225f84c3c))
* **rust:** reject partially decoded form secrets ([121c6be](https://github.com/cpaikr/opendart/commit/121c6be8bd0f25865df67c65ac9ba4e05f09d730))
* **rust:** tighten the core safety boundary ([8993a71](https://github.com/cpaikr/opendart/commit/8993a71313be330580d2f7dc5059d35481ca8e20))
* **rust:** validate XML before status classification ([44052ce](https://github.com/cpaikr/opendart/commit/44052cea25cc0eaf050cd60126497785c64c711a))

## Changelog

Release Please maintains the released `opendart` crate history in this file.
