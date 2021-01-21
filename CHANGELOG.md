# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## Added

- Improve documentation.

## Changed

- Improve speed of clone/sync by using goroutines.
- Improve code coverage & testing.

## Fixed

- bulk-git: fails if no args provided.
- bulk-git: fix race condition.
- clone: fix race condition.
- sync: fix race condition.

## [0.3.0] -  2021-01-18

## Added

- [#6](https://github.com/creekorful/srcode/issues/6) Implement srcode set-cmd.
- [#7](https://github.com/creekorful/srcode/issues/7) Implement srcode mv.

## Changed

- Improve documentation.
- Improve output of srcode ls.

## [0.2.0] - 2021-01-17

## Added

- Now create README.md in meta directory to explain how to use it.
- [#2](https://github.com/creekorful/srcode/issues/2) Implement srcode ls.
- [#3](https://github.com/creekorful/srcode/issues/3) Implement srcode bulk-git.

## Changed

- cmd/clone: Display cloning progress.
- cmd/sync: Display sync progress.

## Fixed

- [#1](https://github.com/creekorful/srcode/issues/1) Fix codebase init & clone when using absolute path.

## [0.1.0] - 2021-01-15

Initial pre-release.

[Unreleased]: https://github.com/creekorful/srcode/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/creekorful/srcode/compare/v0.3.0...HEAD
[0.2.0]: https://github.com/creekorful/srcode/compare/v0.2.0...HEAD
[0.1.0]: https://github.com/creekorful/srcode/releases/tag/v0.1.0