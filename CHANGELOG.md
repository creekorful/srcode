# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## Changed

- cmd/script: allow executing script with arguments.
- cmd/hook: install hook as pre-push rather than pre-commit.
- cmd/script: prevent from adding blank script.

## [0.7.1] - 2021-02-11

## Changed

- cmd/ls: add demo message when running `ls` with no projects.
- change default branch from master to main.
- cmd/script: display available scripts when running with no arguments.

## Fixed

- cmd/script: allow adding global script even outside project folder.
- [#22](https://github.com/creekorful/srcode/issues/22) cmd/script: edition of global script does not set previous content in $EDITOR.

## [0.7.0] - 2021-02-10

## Added

- [#20](https://github.com/creekorful/srcode/issues/20) add git hook support.

## Changed

- cmd/bulk-git: Improve output.
- [#18](https://github.com/creekorful/srcode/issues/18) cmd/set-cmd: edit with $EDITOR if command not directly provided.
- cmd/set-cmd: allow edition of existing command.
- cmd/set-cmd: rename to script.
- [#19](https://github.com/creekorful/srcode/issues/19) cmd/script: now we are making script and not just command.

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

[Unreleased]: https://github.com/creekorful/srcode/compare/v0.7.1...HEAD
[0.7.1]: https://github.com/creekorful/srcode/compare/v0.7.1...HEAD
[0.7.0]: https://github.com/creekorful/srcode/compare/v0.7.0...HEAD
[0.6.0]: https://github.com/creekorful/srcode/compare/v0.6.0...HEAD
[0.5.0]: https://github.com/creekorful/srcode/compare/v0.5.0...HEAD
[0.5.0]: https://github.com/creekorful/srcode/compare/v0.5.0...HEAD
[0.4.0]: https://github.com/creekorful/srcode/compare/v0.4.0...HEAD
[0.3.0]: https://github.com/creekorful/srcode/compare/v0.3.0...HEAD
[0.2.0]: https://github.com/creekorful/srcode/compare/v0.2.0...HEAD
[0.1.0]: https://github.com/creekorful/srcode/releases/tag/v0.1.0
