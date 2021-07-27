# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## Unreleased

### Added

- Prometheus metrics for trial garbage collection, trial duration and tick duration

### Fixed

- Memory leaks

## v1.0.2 - 2021-07-07

### Changed

- Update copyright notice to use the legal name of AI Redefined Inc.
- Use strings everywhere for trial id
- Add ability to disable Prometheus server

### Fixed

- Order of state recording in datalog sample (to be the state at the end of the tick).

## v1.0.1 - 2021-06-02

### Changed

- Cleanup logs and add trace logs
- Fix environment message sending (to only send when there are messages)
- Add SIGSEGV trapping and reporting
- Add log file output option

## v1.0.0 - 2021-05-10

- Initial public release.

### Fixed

- Environment can now receive messages
- The parameter `max_steps` now works

## v1.0.0-beta3 - 2021-04-26

### Added

- Add implementation for the GetTrialInfo function

### Fixed

- Fill in missing API data in the datalog `DatalogSample.TrialData` protobuf class
- Fix timing problems causing various issues

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
