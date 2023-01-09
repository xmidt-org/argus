# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [v0.9.5]
- [Replace webpa common/logging, go-kit/log with zap #332](https://github.com/xmidt-org/argus/issues/332)
- Updated tracing configuration documentation in argus.yaml to reflect changes in Candlelight [#371](https://github.com/xmidt-org/argus/pull/371/)
- Vuln patches
  - [CVE-2022-32149 (High) detected in golang.org/x/text-v0.3.7](https://github.com/xmidt-org/argus/issues/304)

## [v0.9.4]
- Update docker and remove unused packaging files. [317](https://github.com/xmidt-org/argus/pull/317)
- Fix Broken Metric Middleware Function. [302](https://github.com/xmidt-org/argus/pull/302)
- Update dependencies

## [v0.9.3]
- Updated dependencies

## [v0.9.2]
- Update dependencies containing deprecated webpa-common #280 
- Fix Linter #294 

## [v0.9.1]
- Dependency update, note vulnerabilities
  - [github.com/gorilla/sessions (undefined affected versions) no patch available.]
    - https://ossindex.sonatype.org/vulnerability/sonatype-2021-4899
- JWT Migration [238](https://github.com/xmidt-org/scytale/issues/183)
  - updated to use clortho `Resolver` & `Refresher`
  - updated to use clortho `metrics` & `logging`
- Update Config
  - Update auth config for clortho
        
## [v0.9.0]
- Added query condition for GetAll in dynamodb that requires a secondary index in the database. [#230](https://github.com/xmidt-org/argus/pull/230)
- Added a configurable GetAll limit and metric for dynamodb to minimize performance issues from getting too many records. [#230](https://github.com/xmidt-org/argus/pull/230)

## [v0.8.0]
- Removed setLogger func dependency in chrysom basic client. [#228](https://github.com/xmidt-org/argus/pull/228)
- Fixed chrysom basic client fallback to a non-context logger. [#228](https://github.com/xmidt-org/argus/pull/228)

## [v0.7.0]
- Updated spec file and rpkg version macro to be able to choose when the 'v' is included in the version. [#224](https://github.com/xmidt-org/argus/pull/224)
- Make ID case sensitive. [#227](https://github.com/xmidt-org/argus/pull/227)

## [v0.6.0]
- Split Chrysom client into BasicClient and ListenerClient. [#206](https://github.com/xmidt-org/argus/pull/206)
- Bumped spf13 and added documentation. [#220](https://github.com/xmidt-org/argus/pull/220)
- Added logging when returning a non 200/404 response. [#222](https://github.com/xmidt-org/argus/pull/222)
- Bad request error information is sent in the error header. [#223](https://github.com/xmidt-org/argus/pull/223)

## [v0.5.2]
- Update store section in sample config files. [#200](https://github.com/xmidt-org/argus/pull/200)
- Update sample config value so argus webhook integration works out of the box in test environments. [#203](https://github.com/xmidt-org/argus/pull/203)
- Clarify behavior around requests that don't exercise the item owner header. [#202](https://github.com/xmidt-org/argus/pull/202)

## [v0.5.1]
- Fix github actions config for uploading test and coverage reports for sonarcloud. [#192](https://github.com/xmidt-org/argus/pull/192)
- Fix security warning by dropping use of github.com/dgrijalva/jwt-go dep. [#195](https://github.com/xmidt-org/argus/pull/195)

## [v0.5.0]
- Removed dependency on webpa-common, fixing circular dependency issue. [#190](https://github.com/xmidt-org/argus/pull/190)
- Bump bascule version and use it for all auth related middleware, modifying the authx configuration. [#190](https://github.com/xmidt-org/argus/pull/190)
- Use arrange package for servers. [#190](https://github.com/xmidt-org/argus/pull/190)
- Use arrange to unmarshal some config structs. [#190](https://github.com/xmidt-org/argus/pull/190)
- Use touchstone for metrics. [#190](https://github.com/xmidt-org/argus/pull/190)

## [v0.4.1]
- Changed nothing.

## [v0.4.0]
- Add URLParse Option to auth package. [#179](https://github.com/xmidt-org/argus/pull/179)
- Use latest version of httpaux. [#180](https://github.com/xmidt-org/argus/pull/180)
- Use arrange and zap logger. [#185](https://github.com/xmidt-org/argus/pull/185)
- Bumped bascule version. [#187](https://github.com/xmidt-org/argus/pull/187)

## [v0.3.16]
- Allow auth package client code to pass the basculehttp.OnErrorHTTPResponse option. [#174](https://github.com/xmidt-org/argus/pull/174)
- Fix bug that did not include context (with tracing data) in outgoing requests. [#176](https://github.com/xmidt-org/argus/pull/176)

## [v0.3.15]
- Add optional OpenTelemetry tracing feature. [#170](https://github.com/xmidt-org/argus/pull/170) thanks to @utsavbatra5
- Remove ErrBucketNotFound from InMem store implementation as it's not that helpful. [#171](https://github.com/xmidt-org/argus/pull/171)

## [v0.3.14]
- Rely on instrumented client to propagate OpenTelemetry trace context. [#162](https://github.com/xmidt-org/argus/pull/162)

## [v0.3.13]
### Added
- Add configurable validation for an item's data depth. [#146](https://github.com/xmidt-org/argus/pull/146)
- Add initial OpenTelemetry integration to Argus client. [#145](https://github.com/xmidt-org/argus/pull/145) thanks to @Sachin4403

### Fixed
- Sanitize dynamodb errors before reporting in HTTP response headers. [#149](https://github.com/xmidt-org/argus/pull/149)
- Sanitize errors in inMem and cassandra store implementations. [#155](https://github.com/xmidt-org/argus/pull/155)
- Sanitize errors in dynamodb store implementation. [#159](https://github.com/xmidt-org/argus/pull/159)
- Minor Go struct documentation rewording. [#155](https://github.com/xmidt-org/argus/pull/155)


## [v0.3.12]
### Changed
- Better validation and documentation for Argus config. [#141](https://github.com/xmidt-org/argus/pull/141)
- Make bucket a required field for the client not just the listener. [#141](https://github.com/xmidt-org/argus/pull/141)
- Group listener-only config for client. [#142](https://github.com/xmidt-org/argus/pull/142)

### Fixed
- Super user getAll requests should be filtered by owner when one is provided. [#136](https://github.com/xmidt-org/argus/pull/136)

## [v0.3.11]
### Changed
- Make client listener thread-safe and friendlier to uber/fx hooks. [#128](https://github.com/xmidt-org/argus/pull/128)
### Fixed
- Bug in which the userInputValidation config section was required even when it should've been optional. [#129](https://github.com/xmidt-org/argus/pull/129)
- Fix logging to use `xlog` instead of deprecated `webpa-common/logging` package. [#132](https://github.com/xmidt-org/argus/pull/132)
- Fix ListenerFunc interface. [#133](https://github.com/xmidt-org/argus/pull/133)


## [v0.3.10]
### Changed
- Migrate to github actions, normalize analysis tools, Dockerfiles and Makefiles. [#96](https://github.com/xmidt-org/argus/pull/96)
- Bumped webpa-common to v1.11.2 and updated setup for capability check accordingly. [#74](https://github.com/xmidt-org/argus/pull/74)
- UUID field is now ID. [#88](https://github.com/xmidt-org/argus/pull/88)
- Update buildtime format in Makefile to match RPM spec file. [#95](https://github.com/xmidt-org/argus/pull/95)
- Update configuration structure for inbound authn/z. [#101](https://github.com/xmidt-org/argus/pull/101)
- Admin mode flag now originates from JWT claims instead of an HTTP header. [#112](https://github.com/xmidt-org/argus/pull/112)
- Remove stored loggers. [#118](https://github.com/xmidt-org/argus/pull/118)
- Drop use of admin token headers from client. [#118](https://github.com/xmidt-org/argus/pull/118)
- Refactor client code and add unit tests around item CRUD operations [#119](https://github.com/xmidt-org/argus/pull/119)

### Fixed
- Fix behavior in which the owner of an existing item was overwritten in super user mode. [#116](https://github.com/xmidt-org/argus/pull/116)

### Added
- Item ID is validated to have the format of a SHA256 message hex digest. [#106](https://github.com/xmidt-org/argus/pull/106)
- Configurable item owner validation. [#121](https://github.com/xmidt-org/argus/pull/121)
- Configurable item bucket validation. [#114](https://github.com/xmidt-org/argus/pull/114)

### Removed
- Removed identifier as a field from the API. [#85](https://github.com/xmidt-org/argus/pull/85)


## [v0.3.9]
- Small bug fix to the client. [#72](https://github.com/xmidt-org/argus/pull/72)

## [v0.3.8]
- Update code to abide by latest API spec in the main repo readme. [#71](https://github.com/xmidt-org/argus/pull/71)

## [v0.3.7]
### Changed
- Changes the PUT creation route to a POST. [#68](https://github.com/xmidt-org/argus/pull/68)
- Adds a PUT route to update a specific resource. [#68](https://github.com/xmidt-org/argus/pull/68)
- Changes the way IDs are generated for resources on creation. [#68](https://github.com/xmidt-org/argus/pull/68)

## [v0.3.6]
### Changed
- Abstract away dynamodb dependency. [#35](https://github.com/xmidt-org/argus/pull/35)
- Add unit tests for new dynamodb abstraction changes. [#39](https://github.com/xmidt-org/argus/pull/39)
- Add helper functions for GetItem Handler. [#44](https://github.com/xmidt-org/argus/pull/44)
- Add helper functions for DeleteItem Handler. [#52](https://github.com/xmidt-org/argus/pull/52)
- Add helper functions for GetAllItems Handler. [#53](https://github.com/xmidt-org/argus/pull/53)
- Add helper functions for PushItem Handler. [#55](https://github.com/xmidt-org/argus/pull/55)
- Switch store provider to use new handlers. [#56](https://github.com/xmidt-org/argus/pull/56)
- Simplify metrics and add back instrumentation for dynamo. [#58](https://github.com/xmidt-org/argus/pull/58)

### Fixed 
- Add back listener to chrysom config. [#54](https://github.com/xmidt-org/argus/pull/54)


## [v0.3.5]
### Changed
- Changed setting/getting logger in context to use xlog package. [#17](https://github.com/xmidt-org/argus/pull/17)
- Simplify client constructor and add error logging. [#26](https://github.com/xmidt-org/argus/pull/26)

### Added
- Added a counter for chrysom data polls with a label around success/failure. [#23](https://github.com/xmidt-org/argus/pull/23)

### Fixed
- Expired ownable items are no longer returned. [#22](https://github.com/xmidt-org/argus/pull/22)
- Check route before entering auth chain. [#21](https://github.com/xmidt-org/argus/pull/21)

## [v0.3.4]
- handle error case of identifier being too large [#14](https://github.com/xmidt-org/argus/pull/14)
- handle TTL edge cases [#14](https://github.com/xmidt-org/argus/pull/14)
- fix error of data field not being required [#14](https://github.com/xmidt-org/argus/pull/14)
- removed returning raw error code in http headers [#14](https://github.com/xmidt-org/argus/pull/14)
- add itemTTL configuration [#14](https://github.com/xmidt-org/argus/pull/14)
- Updated references to the main branch [#15](https://github.com/xmidt-org/argus/pull/15)
- Changed docker-compose to reference yb-manager [#15](https://github.com/xmidt-org/argus/pull/15)

## [v0.3.3]
- encode error as header  [#10](https://github.com/xmidt-org/argus/pull/10)
- add ability to disable pullInterval for chrysom  [#10](https://github.com/xmidt-org/argus/pull/10)
- remove authHeader from logging [#10](https://github.com/xmidt-org/argus/pull/10)
- update response header and status code for invalided requests [#10](https://github.com/xmidt-org/argus/pull/10)

## [v0.3.2]
- fix missing conf folder

## [v0.3.1]
- fix missing rpkg.macros file

## [v0.3.0]
- added bascule for authentication
- added api/v1 to path
- reworked webhookclient to be more generic
- renamed webhookclient to chrysom

## [v0.2.1]
- added capacity consumed metrics to dynamoDB
- improved error handling for dynamoDB
- fixed webhookclient to use `PUT` instead of `POST`

## [v0.2.0]
- reworked api
- added dynamodb support
- removed cache from webhookclient package

## [v0.1.1]
- fixed import error with webhookclient package

## [v0.1.0] Tue May 07 2020 Jack Murdock - 0.1.0
- initial creation

[Unreleased]: https://github.com/xmidt-org/argus/compare/v0.9.5...HEAD
[v0.9.4]: https://github.com/xmidt-org/argus/compare/v0.9.4...v0.9.5 
[v0.9.4]: https://github.com/xmidt-org/argus/compare/v0.9.3...v0.9.4
[v0.9.3]: https://github.com/xmidt-org/argus/compare/v0.9.2...v0.9.3
[v0.9.2]: https://github.com/xmidt-org/argus/compare/v0.9.1...v0.9.2
[v0.9.1]: https://github.com/xmidt-org/argus/compare/v0.9.0...v0.9.1
[v0.9.0]: https://github.com/xmidt-org/argus/compare/v0.8.0...v0.9.0
[v0.8.0]: https://github.com/xmidt-org/argus/compare/v0.7.0...v0.8.0
[v0.7.0]: https://github.com/xmidt-org/argus/compare/v0.6.0...v0.7.0
[v0.6.0]: https://github.com/xmidt-org/argus/compare/v0.5.2...v0.6.0
[v0.5.2]: https://github.com/xmidt-org/argus/compare/v0.5.1...v0.5.2
[v0.5.1]: https://github.com/xmidt-org/argus/compare/v0.5.0...v0.5.1
[v0.5.0]: https://github.com/xmidt-org/argus/compare/v0.4.1...v0.5.0
[v0.4.1]: https://github.com/xmidt-org/argus/compare/v0.4.0...v0.4.1
[v0.4.0]: https://github.com/xmidt-org/argus/compare/v0.3.16...v0.4.0
[v0.3.16]: https://github.com/xmidt-org/argus/compare/v0.3.15...v0.3.16
[v0.3.15]: https://github.com/xmidt-org/argus/compare/v0.3.14...v0.3.15
[v0.3.14]: https://github.com/xmidt-org/argus/compare/v0.3.13...v0.3.14
[v0.3.13]: https://github.com/xmidt-org/argus/compare/v0.3.12...v0.3.13
[v0.3.12]: https://github.com/xmidt-org/argus/compare/v0.3.11...v0.3.12
[v0.3.11]: https://github.com/xmidt-org/argus/compare/v0.3.10...v0.3.11
[v0.3.10]: https://github.com/xmidt-org/argus/compare/v0.3.9...v0.3.10
[v0.3.9]: https://github.com/xmidt-org/argus/compare/v0.3.8...v0.3.9
[v0.3.8]: https://github.com/xmidt-org/argus/compare/v0.3.7...v0.3.8
[v0.3.7]: https://github.com/xmidt-org/argus/compare/v0.3.6...v0.3.7
[v0.3.6]: https://github.com/xmidt-org/argus/compare/v0.3.5...v0.3.6
[v0.3.5]: https://github.com/xmidt-org/argus/compare/v0.3.4...v0.3.5
[v0.3.4]: https://github.com/xmidt-org/argus/compare/v0.3.3...v0.3.4
[v0.3.3]: https://github.com/xmidt-org/argus/compare/v0.3.2...v0.3.3
[v0.3.2]: https://github.com/xmidt-org/argus/compare/v0.3.1...v0.3.2
[v0.3.1]: https://github.com/xmidt-org/argus/compare/v0.3.0...v0.3.1
[v0.3.0]: https://github.com/xmidt-org/argus/compare/v0.2.1...v0.3.0
[v0.2.1]: https://github.com/xmidt-org/argus/compare/v0.2.0...v0.2.1
[v0.2.0]: https://github.com/xmidt-org/argus/compare/v0.1.1...v0.2.0
[v0.1.1]: https://github.com/xmidt-org/argus/compare/v0.1.0...v0.1.1
[v0.1.0]: https://github.com/xmidt-org/argus/compare/v0.1.0...v0.1.0
