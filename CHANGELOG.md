# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## [Unreleased]


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

[Unreleased]: https://github.com/xmidt-org/argus/compare/v0.3.9...HEAD
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

