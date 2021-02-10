# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## Changed

- Improve output of bulk-git.
- [#18](https://github.com/creekorful/srcode/issues/18) set-cmd: edit with $EDITOR if command not directly provided.
- set-cmd: allow edition of existing command.
- set-cmd: rename to script.
- [#19](https://github.com/creekorful/srcode/issues/19) script: now we are making script and not just command.
- [#20](https://github.com/creekorful/srcode/issues/20) add git hook support.

## [0.6.0] - 2021-02-09

## Added

- [#15](https://github.com/creekorful/srcode/issues/15): Implement rm.
- [#4](https://github.com/creekorful/srcode/issues/4): Add --import to init.

## [0.5.0] - 2021-01-24

## Changed

- cmd/ls: Display git branch.
- cmd/ls: Display git status (dirty, clean)
- cmd/run: Display real time output.

## [0.4.0] - 2021-01-22

## Added

- Improve documentation.

## Changed

- Improve speed of clone/sync by using goroutines.
- Improve code coverage & testing.

## Fixed

- cmd/bulk-git: fails if no args provided.

## [0.3.0] - 2021-01-18

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

[Unreleased]: https://github.com/creekorful/srcode/compare/v0.6.0...HEAD
[0.6.0]: https://github.com/creekorful/srcode/compare/v0.6.0...HEAD
[0.5.0]: https://github.com/creekorful/srcode/compare/v0.5.0...HEAD
[0.5.0]: https://github.com/creekorful/srcode/compare/v0.5.0...HEAD
[0.4.0]: https://github.com/creekorful/srcode/compare/v0.4.0...HEAD
[0.3.0]: https://github.com/creekorful/srcode/compare/v0.3.0...HEAD
[0.2.0]: https://github.com/creekorful/srcode/compare/v0.2.0...HEAD
[0.1.0]: https://github.com/creekorful/srcode/releases/tag/v0.1.0
