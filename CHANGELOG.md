# Changelog

## [0.10.0](https://github.com/supercargo-dev/core/compare/github.com/supercargo-dev/supercargo-sdk-go-v0.9.0...github.com/supercargo-dev/supercargo-sdk-go-v0.10.0) (2026-09-02)


### Features

* Implement Go SDK Annotations & Architectural Updates ([083097f](https://github.com/supercargo-dev/core/commit/083097fb0de5d1890c413070bb8c7fb41768ada3))
* **platform:** terraform and SDK parity, authz hardening, and dataform SQLX migration ([da252bd](https://github.com/supercargo-dev/core/commit/da252bd3908528b6c8c1038d8a6f5d1db94185a4))
* **proto:** first-class explicit primary_key and ranked sort_rank across polyglot SDKs and Sentinel ([d42769a](https://github.com/supercargo-dev/core/commit/d42769ae6e75ff304129de451184e39bb7ac5be7))
* **schema:** central entity taxonomy auto-hydration across polyglot SDKs ([f0dccf3](https://github.com/supercargo-dev/core/commit/f0dccf34c6be7977935d9851235cd4d0dd209bfb))
* **schema:** cross-language contract parity, decoupled provider registry, and SDK docs ([70a9eb2](https://github.com/supercargo-dev/core/commit/70a9eb26ae40bf6af17474cba8317e6b52fa33fa))
* **schema:** support explicit column and field aliasing across DTOs, AST providers, and Gateway ([3d2fb50](https://github.com/supercargo-dev/core/commit/3d2fb50aada12c54f713a9efa014ee0756027a58))
* **sdk/go:** add Hub and Vault gRPC client abstractions with retries ([78d0d49](https://github.com/supercargo-dev/core/commit/78d0d49d6be5d795da8e7d1652d9d7a1724656bf))
* **sdk:** add go-playground validator tag adapter and inclusive bounds ([e6944f5](https://github.com/supercargo-dev/core/commit/e6944f5676099a9e262e62c40ee167f0b67d8ebd))
* **sdk:** add go-playground validator tag adapter to Go SDK ([1b29207](https://github.com/supercargo-dev/core/commit/1b2920708a81c72d7d948e8522ee9dd735ad19bd))
* **sdk:** Add native struct tag parsing and robust collection/graph validation ([b0b10ca](https://github.com/supercargo-dev/core/commit/b0b10ca0e0484196bd67911be11a45661e0a9a20))
* **sdk:** implement go sdk annotations and distributed repo map updates ([9d5b506](https://github.com/supercargo-dev/core/commit/9d5b506e6a10fbb7de492dd4fce445907704c555))
* **sdk:** Implement Go SDK annotations and runtime validator ([62362cc](https://github.com/supercargo-dev/core/commit/62362ccf00a0df8e6da936d0294759a898ad81df))
* **sdk:** implement go sdk distribution ([#204](https://github.com/supercargo-dev/core/issues/204)) ([0f156b2](https://github.com/supercargo-dev/core/commit/0f156b2819ce7a6c9dbaaa4244a3147c83679ca5))
* **sdks,providers,formatter:** explicit descriptive PII annotation support (pii:&lt;entity&gt;) ([9851623](https://github.com/supercargo-dev/core/commit/985162322dc6084e52311c5f4666f774cfba9015))
* **sdks/go:** add descriptive PII struct tag and provider support ([3d15c3e](https://github.com/supercargo-dev/core/commit/3d15c3e9b922e1e2fc62fd0748559ffba58a613d))
* **sdks/go:** add primary_key and sort_rank builder methods and tag parsing ([251b85f](https://github.com/supercargo-dev/core/commit/251b85f751000361ebfe5041f2a8b2cc7650db24))
* **sdks/go:** make Go SDK self-contained with internal proto stubs and script ([b42d9f5](https://github.com/supercargo-dev/core/commit/b42d9f554383152370625c0ee1be4da3c8244eb6))
* **sdks/go:** support shorthand entity struct tags in Go AST provider ([c02a5fa](https://github.com/supercargo-dev/core/commit/c02a5fac11c606bf7f323ad9853358c885444410))
* **sdk:** support field aliasing builders and decorators across Go, Python, TypeScript, and Java ([d0c4578](https://github.com/supercargo-dev/core/commit/d0c457845b457cf4b9df24c95d8857428a2093da))
* **sdk:** support inclusive bounds and exact length in rule compiler ([2ca0780](https://github.com/supercargo-dev/core/commit/2ca078081574754c45b9560c8bb3c1380eca4416))


### Bug Fixes

* **ci:** fix stale sdk test and disable deploy to dev on PRs ([b2ff9f8](https://github.com/supercargo-dev/core/commit/b2ff9f834074895ce74c3a8a6191760e8ccdb7a7))
* **ci:** handle PR branch drift in CI remote check and harden parity verification ([6c37f3b](https://github.com/supercargo-dev/core/commit/6c37f3b488f651805458c104e6bc9e3bf9f8fcc4))
* **governance:** remediate expert review findings on authz security, nil safety, payload isolation, and retry timer hygiene ([7a16af1](https://github.com/supercargo-dev/core/commit/7a16af154986b0c6afdf4371011f66ad316d5940))
* **governance:** remediate expert review findings on sentinel FULL mode, recursive context synthesis, and AST sort rank validation ([6d8665e](https://github.com/supercargo-dev/core/commit/6d8665e90468b511abfb7af23334b87b4f5cca80))
* **sdk:** Address expert review security and concurrency findings ([94a78ff](https://github.com/supercargo-dev/core/commit/94a78ffc476387a43dc10dac9faa5c4af0f29cf8))
* **sdk:** enforce fail-closed bounds, zero-alloc oneof, and context-derived PII redaction ([3724484](https://github.com/supercargo-dev/core/commit/3724484794b7f5be6ae2b8db6e83e16484fc2210))
* **sdk:** publish readme to public repo ([#208](https://github.com/supercargo-dev/core/issues/208)) ([bf95478](https://github.com/supercargo-dev/core/commit/bf9547855dd8d2d21588f62ef4b14343505efa8b))
* **sdks/java:** trigger release ([208f533](https://github.com/supercargo-dev/core/commit/208f5330695db789e3c848cf7022f2af660bcb75))
* **sdks/typescript:** include dist directory in npm package files ([d36f8e6](https://github.com/supercargo-dev/core/commit/d36f8e6b8e9fbd33c48294d55b81a78800b03fb8))

## [0.9.0](https://github.com/supercargo-dev/core/compare/github.com/supercargo-dev/supercargo-sdk-go-v0.8.0...github.com/supercargo-dev/supercargo-sdk-go-v0.9.0) (2026-09-02)


### Features

* Implement Go SDK Annotations & Architectural Updates ([083097f](https://github.com/supercargo-dev/core/commit/083097fb0de5d1890c413070bb8c7fb41768ada3))
* **platform:** terraform and SDK parity, authz hardening, and dataform SQLX migration ([da252bd](https://github.com/supercargo-dev/core/commit/da252bd3908528b6c8c1038d8a6f5d1db94185a4))
* **proto:** first-class explicit primary_key and ranked sort_rank across polyglot SDKs and Sentinel ([d42769a](https://github.com/supercargo-dev/core/commit/d42769ae6e75ff304129de451184e39bb7ac5be7))
* **schema:** central entity taxonomy auto-hydration across polyglot SDKs ([f0dccf3](https://github.com/supercargo-dev/core/commit/f0dccf34c6be7977935d9851235cd4d0dd209bfb))
* **schema:** cross-language contract parity, decoupled provider registry, and SDK docs ([70a9eb2](https://github.com/supercargo-dev/core/commit/70a9eb26ae40bf6af17474cba8317e6b52fa33fa))
* **schema:** support explicit column and field aliasing across DTOs, AST providers, and Gateway ([3d2fb50](https://github.com/supercargo-dev/core/commit/3d2fb50aada12c54f713a9efa014ee0756027a58))
* **sdk/go:** add Hub and Vault gRPC client abstractions with retries ([78d0d49](https://github.com/supercargo-dev/core/commit/78d0d49d6be5d795da8e7d1652d9d7a1724656bf))
* **sdk:** add go-playground validator tag adapter and inclusive bounds ([e6944f5](https://github.com/supercargo-dev/core/commit/e6944f5676099a9e262e62c40ee167f0b67d8ebd))
* **sdk:** add go-playground validator tag adapter to Go SDK ([1b29207](https://github.com/supercargo-dev/core/commit/1b2920708a81c72d7d948e8522ee9dd735ad19bd))
* **sdk:** Add native struct tag parsing and robust collection/graph validation ([b0b10ca](https://github.com/supercargo-dev/core/commit/b0b10ca0e0484196bd67911be11a45661e0a9a20))
* **sdk:** implement go sdk annotations and distributed repo map updates ([9d5b506](https://github.com/supercargo-dev/core/commit/9d5b506e6a10fbb7de492dd4fce445907704c555))
* **sdk:** Implement Go SDK annotations and runtime validator ([62362cc](https://github.com/supercargo-dev/core/commit/62362ccf00a0df8e6da936d0294759a898ad81df))
* **sdk:** implement go sdk distribution ([#204](https://github.com/supercargo-dev/core/issues/204)) ([0f156b2](https://github.com/supercargo-dev/core/commit/0f156b2819ce7a6c9dbaaa4244a3147c83679ca5))
* **sdks,providers,formatter:** explicit descriptive PII annotation support (pii:&lt;entity&gt;) ([9851623](https://github.com/supercargo-dev/core/commit/985162322dc6084e52311c5f4666f774cfba9015))
* **sdks/go:** add descriptive PII struct tag and provider support ([3d15c3e](https://github.com/supercargo-dev/core/commit/3d15c3e9b922e1e2fc62fd0748559ffba58a613d))
* **sdks/go:** add primary_key and sort_rank builder methods and tag parsing ([251b85f](https://github.com/supercargo-dev/core/commit/251b85f751000361ebfe5041f2a8b2cc7650db24))
* **sdks/go:** make Go SDK self-contained with internal proto stubs and script ([b42d9f5](https://github.com/supercargo-dev/core/commit/b42d9f554383152370625c0ee1be4da3c8244eb6))
* **sdks/go:** support shorthand entity struct tags in Go AST provider ([c02a5fa](https://github.com/supercargo-dev/core/commit/c02a5fac11c606bf7f323ad9853358c885444410))
* **sdk:** support field aliasing builders and decorators across Go, Python, TypeScript, and Java ([d0c4578](https://github.com/supercargo-dev/core/commit/d0c457845b457cf4b9df24c95d8857428a2093da))
* **sdk:** support inclusive bounds and exact length in rule compiler ([2ca0780](https://github.com/supercargo-dev/core/commit/2ca078081574754c45b9560c8bb3c1380eca4416))


### Bug Fixes

* **ci:** fix stale sdk test and disable deploy to dev on PRs ([b2ff9f8](https://github.com/supercargo-dev/core/commit/b2ff9f834074895ce74c3a8a6191760e8ccdb7a7))
* **ci:** handle PR branch drift in CI remote check and harden parity verification ([6c37f3b](https://github.com/supercargo-dev/core/commit/6c37f3b488f651805458c104e6bc9e3bf9f8fcc4))
* **governance:** remediate expert review findings on authz security, nil safety, payload isolation, and retry timer hygiene ([7a16af1](https://github.com/supercargo-dev/core/commit/7a16af154986b0c6afdf4371011f66ad316d5940))
* **governance:** remediate expert review findings on sentinel FULL mode, recursive context synthesis, and AST sort rank validation ([6d8665e](https://github.com/supercargo-dev/core/commit/6d8665e90468b511abfb7af23334b87b4f5cca80))
* **sdk:** Address expert review security and concurrency findings ([94a78ff](https://github.com/supercargo-dev/core/commit/94a78ffc476387a43dc10dac9faa5c4af0f29cf8))
* **sdk:** enforce fail-closed bounds, zero-alloc oneof, and context-derived PII redaction ([3724484](https://github.com/supercargo-dev/core/commit/3724484794b7f5be6ae2b8db6e83e16484fc2210))
* **sdk:** publish readme to public repo ([#208](https://github.com/supercargo-dev/core/issues/208)) ([bf95478](https://github.com/supercargo-dev/core/commit/bf9547855dd8d2d21588f62ef4b14343505efa8b))
* **sdks/java:** trigger release ([208f533](https://github.com/supercargo-dev/core/commit/208f5330695db789e3c848cf7022f2af660bcb75))
* **sdk:** trigger release to pickup workflow fixes ([#172](https://github.com/supercargo-dev/core/issues/172)) ([c3defe8](https://github.com/supercargo-dev/core/commit/c3defe82fe272ba9ef9d74d27f64c476e607c447))

## [0.8.0](https://github.com/supercargo-dev/core/compare/github.com/supercargo-dev/supercargo-sdk-go-v0.7.0...github.com/supercargo-dev/supercargo-sdk-go-v0.8.0) (2026-09-02)


### Features

* **schema:** central entity taxonomy auto-hydration across polyglot SDKs ([f0dccf3](https://github.com/supercargo-dev/core/commit/f0dccf34c6be7977935d9851235cd4d0dd209bfb))
* **sdks,providers,formatter:** explicit descriptive PII annotation support (pii:&lt;entity&gt;) ([9851623](https://github.com/supercargo-dev/core/commit/985162322dc6084e52311c5f4666f774cfba9015))
* **sdks/go:** add descriptive PII struct tag and provider support ([3d15c3e](https://github.com/supercargo-dev/core/commit/3d15c3e9b922e1e2fc62fd0748559ffba58a613d))
* **sdks/go:** support shorthand entity struct tags in Go AST provider ([c02a5fa](https://github.com/supercargo-dev/core/commit/c02a5fac11c606bf7f323ad9853358c885444410))

## [0.7.0](https://github.com/supercargo-dev/core/compare/github.com/supercargo-dev/supercargo-sdk-go-v0.6.0...github.com/supercargo-dev/supercargo-sdk-go-v0.7.0) (2026-08-26)


### Features

* **sdks/go:** make Go SDK self-contained with internal proto stubs and script ([b42d9f5](https://github.com/supercargo-dev/core/commit/b42d9f554383152370625c0ee1be4da3c8244eb6))


### Bug Fixes

* **ci:** handle PR branch drift in CI remote check and harden parity verification ([6c37f3b](https://github.com/supercargo-dev/core/commit/6c37f3b488f651805458c104e6bc9e3bf9f8fcc4))

## [0.6.0](https://github.com/supercargo-dev/core/compare/github.com/supercargo-dev/supercargo-sdk-go-v0.5.0...github.com/supercargo-dev/supercargo-sdk-go-v0.6.0) (2026-08-23)


### Features

* **platform:** terraform and SDK parity, authz hardening, and dataform SQLX migration ([da252bd](https://github.com/supercargo-dev/core/commit/da252bd3908528b6c8c1038d8a6f5d1db94185a4))
* **proto:** first-class explicit primary_key and ranked sort_rank across polyglot SDKs and Sentinel ([d42769a](https://github.com/supercargo-dev/core/commit/d42769ae6e75ff304129de451184e39bb7ac5be7))
* **schema:** support explicit column and field aliasing across DTOs, AST providers, and Gateway ([3d2fb50](https://github.com/supercargo-dev/core/commit/3d2fb50aada12c54f713a9efa014ee0756027a58))
* **sdk/go:** add Hub and Vault gRPC client abstractions with retries ([78d0d49](https://github.com/supercargo-dev/core/commit/78d0d49d6be5d795da8e7d1652d9d7a1724656bf))
* **sdks/go:** add primary_key and sort_rank builder methods and tag parsing ([251b85f](https://github.com/supercargo-dev/core/commit/251b85f751000361ebfe5041f2a8b2cc7650db24))
* **sdk:** support field aliasing builders and decorators across Go, Python, TypeScript, and Java ([d0c4578](https://github.com/supercargo-dev/core/commit/d0c457845b457cf4b9df24c95d8857428a2093da))


### Bug Fixes

* **governance:** remediate expert review findings on authz security, nil safety, payload isolation, and retry timer hygiene ([7a16af1](https://github.com/supercargo-dev/core/commit/7a16af154986b0c6afdf4371011f66ad316d5940))
* **governance:** remediate expert review findings on sentinel FULL mode, recursive context synthesis, and AST sort rank validation ([6d8665e](https://github.com/supercargo-dev/core/commit/6d8665e90468b511abfb7af23334b87b4f5cca80))

## [0.5.0](https://github.com/supercargo-dev/core/compare/github.com/supercargo-dev/supercargo-sdk-go-v0.4.0...github.com/supercargo-dev/supercargo-sdk-go-v0.5.0) (2026-08-20)


### Features

* **sdk:** add go-playground validator tag adapter and inclusive bounds ([e6944f5](https://github.com/supercargo-dev/core/commit/e6944f5676099a9e262e62c40ee167f0b67d8ebd))
* **sdk:** add go-playground validator tag adapter to Go SDK ([1b29207](https://github.com/supercargo-dev/core/commit/1b2920708a81c72d7d948e8522ee9dd735ad19bd))
* **sdk:** support inclusive bounds and exact length in rule compiler ([2ca0780](https://github.com/supercargo-dev/core/commit/2ca078081574754c45b9560c8bb3c1380eca4416))


### Bug Fixes

* **sdk:** enforce fail-closed bounds, zero-alloc oneof, and context-derived PII redaction ([3724484](https://github.com/supercargo-dev/core/commit/3724484794b7f5be6ae2b8db6e83e16484fc2210))

## [0.4.0](https://github.com/supercargo-dev/core/compare/github.com/supercargo-dev/supercargo-sdk-go-v0.3.1...github.com/supercargo-dev/supercargo-sdk-go-v0.4.0) (2026-08-15)


### Features

* **schema:** cross-language contract parity, decoupled provider registry, and SDK docs ([70a9eb2](https://github.com/supercargo-dev/core/commit/70a9eb26ae40bf6af17474cba8317e6b52fa33fa))

## [0.3.1](https://github.com/supercargo-dev/core/compare/github.com/supercargo-dev/supercargo-sdk-go-v0.3.0...github.com/supercargo-dev/supercargo-sdk-go-v0.3.1) (2026-08-10)


### Bug Fixes

* **ci:** fix stale sdk test and disable deploy to dev on PRs ([b2ff9f8](https://github.com/supercargo-dev/core/commit/b2ff9f834074895ce74c3a8a6191760e8ccdb7a7))

## [0.3.0](https://github.com/supercargo-dev/core/compare/github.com/supercargo-dev/supercargo-sdk-go-v0.2.1...github.com/supercargo-dev/supercargo-sdk-go-v0.3.0) (2026-07-29)


### Features

* Implement Go SDK Annotations & Architectural Updates ([083097f](https://github.com/supercargo-dev/core/commit/083097fb0de5d1890c413070bb8c7fb41768ada3))
* **sdk:** Add native struct tag parsing and robust collection/graph validation ([b0b10ca](https://github.com/supercargo-dev/core/commit/b0b10ca0e0484196bd67911be11a45661e0a9a20))
* **sdk:** implement go sdk annotations and distributed repo map updates ([9d5b506](https://github.com/supercargo-dev/core/commit/9d5b506e6a10fbb7de492dd4fce445907704c555))
* **sdk:** Implement Go SDK annotations and runtime validator ([62362cc](https://github.com/supercargo-dev/core/commit/62362ccf00a0df8e6da936d0294759a898ad81df))


### Bug Fixes

* **sdk:** Address expert review security and concurrency findings ([94a78ff](https://github.com/supercargo-dev/core/commit/94a78ffc476387a43dc10dac9faa5c4af0f29cf8))

## [0.2.1](https://github.com/supercargo-dev/core/compare/github.com/supercargo-dev/supercargo-sdk-go-v0.2.0...github.com/supercargo-dev/supercargo-sdk-go-v0.2.1) (2026-07-13)


### Bug Fixes

* **sdk:** publish readme to public repo ([#208](https://github.com/supercargo-dev/core/issues/208)) ([bf95478](https://github.com/supercargo-dev/core/commit/bf9547855dd8d2d21588f62ef4b14343505efa8b))

## [0.2.0](https://github.com/supercargo-dev/core/compare/github.com/supercargo-dev/supercargo-sdk-go-v0.1.0...github.com/supercargo-dev/supercargo-sdk-go-v0.2.0) (2026-07-13)


### Features

* **sdk:** implement go sdk distribution ([#204](https://github.com/supercargo-dev/core/issues/204)) ([0f156b2](https://github.com/supercargo-dev/core/commit/0f156b2819ce7a6c9dbaaa4244a3147c83679ca5))


### Bug Fixes

* **sdks/java:** trigger release ([208f533](https://github.com/supercargo-dev/core/commit/208f5330695db789e3c848cf7022f2af660bcb75))
* **sdk:** trigger release to pickup workflow fixes ([#172](https://github.com/supercargo-dev/core/issues/172)) ([c3defe8](https://github.com/supercargo-dev/core/commit/c3defe82fe272ba9ef9d74d27f64c476e607c447))
