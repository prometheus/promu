## 0.15.0 / 2023-07-04

* [CHANGE] Use unmarshalstrict #240
* [CHANGE] Add linux/riscv64 to default platforms #254

## 0.14.0 / 2022-12-09

* [FEATURE] Add the ability to override tags per GOOS

## 0.13.0 / 2021-11-06

* [ENHANCEMENT] Add windows/arm64 platform #225

## 0.12.0 / 2021-04-12

* [CHANGE] Unified CGO crossbuild image #219

## 0.11.1 / 2021-03-20

* [BUGFIX] Deduplicate platforms when two regexes match the same platform #214
* [BUGFIX] Regexes are evaluated against all archs, and don't stop at the first match #214

## 0.11.0 / 2021-03-20

* [FEATURE] Add the ability to run parallel build threads independently #212

## 0.10.0 / 2021-03-17

* [FEATURE] Add parallel crossbuilds #208

## 0.9.0 / 2021-03-16

Note: promu crossbuild --platform flag is now a regexp. To
      use multiple options, the flag can be repeated.

* [CHANGE] Use regexp for crossbuild platforms #204

## 0.8.1 / 2021-03-12

This release is cut to publish `darwin/arm64` and `illumos/amd64` binaries of
promu.

* [ENHANCEMENT] Promu is now built from its default branch instead of a released
  binary #205

## 0.8.0 / 2021-03-11

Note: The default build now requires Go 1.16.

* [FEATURE] Add `darwin/arm64` and `illumos/amd64` to the default build #201

## 0.7.0 / 2020-11-03

* [FEATURE] Produce ZIP archives for Windows releases #195

## 0.6.1 / 2020-09-06

* [BUGFIX] Fix cgo builds on illumos by avoiding static linking #192

## 0.6.0 / 2020-09-02

* [CHANGE] Remove default build of darwin/386 #187
* [FEATURE] Add 'check changelog' command. #168
* [FEATURE] Support remotes other than "origin". #174
* [ENHANCEMNT] Improved error handling when parsing CHANGELOG. #161
* [ENANCEMENT] Support arm64 on BSDs. #186

## 0.5.0 / 2019-06-21

* [CHANGE] Remove --broken from git describe. #145
* [FEATURE] Add support for aix/ppc64. #151
* [ENHANCEMENT] cmd/release: add --timeout option. #142
* [ENHANCEMENT] cmd/release: create release in GitHub if none exists. #148

## 0.4.0 / 2019-05-03

* [FEATURE] Fallback to `git describe` output if no VERSION. #130
* [BUGFIX] cmd/tarball: restore --prefix flag. #133
* [BUGFIX] cmd/release: don't leak credentials in case of error. #136

## 0.3.0 / 2019-02-18

* [FEATURE] Make extldflags extensible by configuration. #125
* [ENHANCEMENT] Avoid bind-mounting to allow building with a remote docker engine #95

## 0.2.0 / 2018-11-07

* [FEATURE] Adding changes to support s390x
* [FEATURE] Add option to disable static linking
* [FEATURE] Add support for 32bit MIPS.
* [FEATURE] Added check_licenses Command to Promu
* [ENHANCEMENT] Allow to customize nested options via env variables
* [ENHANCEMENT] Bump Go version to 1.11
* [ENHANCEMENT] Add warning if promu info is unable to determine repo info
* [BUGFIX] Fix build on SmartOS by not setting gcc's -static flag
* [BUGFIX] Fix git repository url parsing

## 0.1.0 / 2017-09-22

Initial release
