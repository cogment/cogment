# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## Unreleased

### Fixed

- Fixed GetTrialInfo function
- Filling in API data in the datalog `DatalogSample.TrialData` protobuf class
- Timing problems causing various issues

## v1.0.0-beta2 - 2021-04-15

### Fixed

- Fixed the problem where the last rewards from the environment would get to the actors too late
- Fixed the filling of DatalogSample (actions, rewards and messages were missing)
- Fixed one deadlock with actor responses

## v1.0.0-beta1 - 2021-04-07

- Initial beta release, no more breaking changes should be introduced.

## v1.0.0-alpha9 - 2021-04-01

### Changed

- Rename ActorClass `id` to `name`

## v1.0.0-alpha8 - 2021-03-30

- Technical release, updating dependencies to fixed versions.

## v1.0.0-alpha7 - 2021-03-30

### Added

- Log exporter is now available: trials param, observations, rewards, messages of every trials are sent to the log exporter service.

### Changed

- Tick ID management centralized in orchestrator

## v1.0.0-alpha6 - 2021-03-10

### Added

- Watch trials is now supported

## v1.0.0-alpha5 - 2021-02-19

### Added

- Add support for messages and rewrads, they can be sent from the actor client.
- Addition for tls communication

## v1.0.0-alpha4 - 2021-02-17

### Added

- Add support for messages and rewrads, they can be sent from the actor client.

## v1.0.0-alpha3 - 2021-01-28

### Fixed

- Fix a crash occuring at the end of trials.
- Fix occasional crash when employing prehooks.

## v1.0.0-alpha2 - 2021-01-11

### Added

- Add support for messages, they can be sent between actors and the environment.
- Add dispatch of immediate rewards to actors.

### Fixed

- Fix the logging levels, making the default less verbose.

## v1.0.0-alpha1 - 2020-12-07

- Initial alpha release, expect some breaking changes.
