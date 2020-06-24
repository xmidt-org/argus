# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## [Unreleased]
- encode error as header 
- add ability to disable pullInterval for chrysom

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

[Unreleased]: https://github.com/xmidt-org/argus/compare/v0.3.2...HEAD
[v0.3.2]: https://github.com/xmidt-org/argus/compare/v0.3.1...v0.3.2
[v0.3.1]: https://github.com/xmidt-org/argus/compare/v0.3.0...v0.3.1
[v0.3.0]: https://github.com/xmidt-org/argus/compare/v0.2.1...v0.3.0
[v0.2.1]: https://github.com/xmidt-org/argus/compare/v0.2.0...v0.2.1
[v0.2.0]: https://github.com/xmidt-org/argus/compare/v0.1.1...v0.2.0
[v0.1.1]: https://github.com/xmidt-org/argus/compare/v0.1.0...v0.1.1
[v0.1.0]: https://github.com/xmidt-org/argus/compare/v0.1.0...v0.1.0

