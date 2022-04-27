# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## Unreleased

### Fixed

- Fix for Windows build being unable to load and get symbols from orchestrator.dll

## v2.2.0 - 2022-04-11

### Added

- Add the gRPC cogment api to the release package.
- Add support for installing the gRPC API from the install script.

### Fixed

- Fix the help message for the `--cache_max_items` option of `cogment services model_registry` command.

## v2.2.0-rc5 - 2022-04-07

### Fixed

- Fix failure of the install script when trying to get the latest version of Cogment.

## v2.2.0-rc4 - 2022-04-07

### Changed

- Update the prometheus C++ client to fix build errors in recent compilers.

### Added

- Build dedicated images for the legacy modules _orchestrator_, _model registry_, _trial datastore_ and _cli_ modules to facilitate migrations.

### Fixed

- Fix the built docker image.

## v2.2.0-rc3 - 2022-03-29

### Changed

- Update the install script to easily retrieve a local exec simply called `./cogment`.

## v2.2.0-rc2 - 2022-03-21

### Fixed

- Fix the artifact path for the "no_orchestrator" macos amd64 version.
- Fix environment variable used to configure the model registry archive directory.

## v2.2.0-rc1 - 2022-03-21

This is the initial release of the _unified_ Cogment executable that includes:

- the orchestrator, accessible as `cogment service orchestrator`,
- the model registry, accessible as `cogment service model_registry`,
- the trial datastore, accessible as `cogment service trial_datastore`,
- the CLI.

Cogment supports natively linux, macOS (> 10.15), and windows on amd64 (aka x84_64) architectures.

### Added

- Add orchestrator support for discovery endpoints (i.e. to query the directory)
- Add orchestrator support for providing the parameters with the StartTrial rpc

### Changed

- Deprecate `cogment run`, users should now rely on a dedicated script, e.g. bash script.
- Deprecate `cogment copy`, users should now rely on dedicated commands, e.g. `cp` if needed.
- Deprecate `cogment init`.
- Update the copyright notice year to 2022.

### Fixed

- Have the orchestrator datalog client consume the stream for the server

## _Model Registry_ - v0.6.0 - 2022-02-25

### Fixed

- Fix issue where the latest published model version would be purged from the memory cache while trials were trying to retrieve it.

### Changed

- Memory cache now uses `lru` package instead of `ccache`
- `COGMENT_MODEL_REGISTRY_VERSION_CACHE_MAX_SIZE` has been deprecated and replaced by `COGMENT_MODEL_REGISTRY_VERSION_CACHE_MAX_ITEMS`.
- `COGMENT_MODEL_REGISTRY_VERSION_CACHE_EXPIRATION` has been deprecated.
- `COGMENT_MODEL_REGISTRY_VERSION_CACHE_PRUNE_COUNT` has been deprecated.

## _Orchestrator_ - v2.1.0 - 2022-02-11

### Added

- Launch script to manage the starting of the webproxy with the orchestrator

### Changed

- The debug version is now suffixed with `_debug` instead of `_dbg`
- The debug version of the orchestrator can now be started from the launch script with an environment variable.

## _Model Registry_ - v0.5.0 - 2022-02-01

### Added

- Implement `cogmentAPI.ModelRegistrySP/RetrieveModels`, the method able to retrieve models and their data.

### Fixed

- Examples in the readme now uses the correct protobuf namespace.

## _Model Registry_ - v0.4.0 - 2022-01-19

### Added

- Add the ability to retrieve any n-th to last version to `cogmentAPI.ModelRegistrySP/RetrieveVersionInfos`, `cogmentAPI.ModelRegistrySP/RetrieveVersionData`.

### Changed

- **Breaking Change** Update Cogment API to 2.0
- Internal `backend.Backend` now uses `uint` for version numbers and uses 0 to request the creation of a new version.

## _CLI_ - 2.0.0 - 2022-01-10

- Updated dockerfiles and yaml files to comply with 2.0

## _Trial Datastore_ - v0.3.0 - 2022-02-24

### Added

- Introduce a backend based on bbolt (https://github.com/etcd-io/bbolt) a file based embedded key-value store.

## _Trial Datastore_ - v0.2.0 - 2022-01-19

### Changed

- **Breaking Change** Update Cogment API to 2.0, no longer support API v1.X

## _CLI_ - 2.0.0-rc1 - 2021-12-16

- Rename `cogment sync` to `cogment copy`
- Code generation updated for Cogment API 2.0

## _Orchestrator_ - v2.0.0 - 2021-12-15

### Changed

- Change warning to debug for expected (under special circumstances) exeptions
- Properly manage forced termination of pending trials

## _Orchestrator_ - v2.0.0-rc3 - 2021-12-10

### Changed

- Stricter control of streams to limit gRPC problems

## _Model Registry_ - v0.3.0 - 2021-12-14

### Changed

- The model registry now stores transient model versions in a memory cache.

### Added

- Introduce `backend.MemoryCacheBackend` a cache backend that stores model version in a upper bounded memory cache and uses another backend for archived versions.

### Removed

- `backend.DbBackend` and `backend.HybridBackend` are no longer used and have been removed.

## _Orchestrator_ - v2.0.0-rc2 - 2021-11-30

### Changed

- Better management of config: differentiate between absence of config and empty config

## _Orchestrator_ - v2.0.0-rc1 - 2021-11-29

### Changed

#### Breaking Changes

- Implement [cogment api 2.0.0](https://github.com/cogment/cogment-api/blob/main/CHANGELOG.md#v200---2021-11-12), in particular follow a new streaming model for actors & environments.
- Rename environment variables `TRIAL_LIFECYCLE_PORT`, `TRIAL_ACTOR_PORT` & `PROMETHEUS_PORT` to `COGMENT_LIFECYCLE_PORT`, `COGMENT_ACTOR_PORT` & `COGMENT_ORCHESTRATOR_PROMETHEUS_PORT`.
- The `cogment.yaml` file is now optional and needs to be provided with the `--params` command line argument or the `COGMENT_DEFAULT_PARAMS_FILE` environment variable, only the `trial_params` is take into account.

- Add the ability to provide a list of pretrial hook endpoints with the `--pre_trial_hooks` comand line argument or the `COGMENT_PRE_TRIAL_HOOKS` environment variable.
- Remove dependencies to easygrpc.
- General refactor and cleanup.

## _Trial Datastore_ - v0.1.2 - 2021-10-25

### Fixed

- Fix bad data initialization that would cause a nil pointer dereference when receiving samples through the datalog server

## _Trial Datastore_ - v0.1.1 - 2021-10-22

### Fixed

- Fix naming collision in cogment gRPC API

## _Trial Datastore_ - v0.1.0 - 2021-10-20

Introduce Trial Datastore, a simple logger & store for trial generated data.

## _Model Registry_ - v0.2.0 - 2021-10-05

### Fixed

- Fix `database is locked` errors during the initial sync by configuring SQLite for concurrent access.

## _Model Registry_ - v0.1.0 - 2021-10-01

Introduce Model Registry, a simple versioned key-value store for models

## _CLI_ - 1.2.0 - 2021-09-27

### Added

- Introduce `cogment sync` a command to synchronize the cogment project settings and proto files to the components directories

### Changed

- Upgrade the version used by `cogment init` of the python sdk to `v1.3.0`

## _CLI_ - 1.1.0 - 2021-09-09

### Added

- Introduce an install script for cogment CLI.

### Fixed

- Fix `cogment generate` generated typescript code by using `grpc_tools_node_protoc` instead of `protoc` directly.

### Changed

- Upgrade the version of the orchestrator used by `cogment init` to `v1.0.3`

## _Orchestrator_ - v1.0.3 - 2021-07-30

### Added

- Add Prometheus metrics for trial garbage collection, trial duration and tick duration
- Add the ability to disable Prometheus server by setting its port to 0 in `PROMETHEUS_PORT` or with the `--prometheus-port` CLI flag

### Fixed

- Fix several memory leaks that was causing the memory to grow with each trial execution

## _Orchestrator_ - v1.0.2 - 2021-07-07

### Changed

- Update copyright notice to use the legal name of AI Redefined Inc.
- Use strings everywhere for trial id

### Fixed

- Order of state recording in datalog sample (to be the state at the end of the tick).

## _CLI_ - 1.0.3 - 2021-07-07

### Changed

- `cogment init` now uses fixed version for the metrics and dashboard.
- cogment.yaml template includes client in docker-compose build command

## _CLI_ - 1.0.2 - 2021-06-17

## _CLI_ - 1.0.1 - 2021-06-04

### Changed

- cogment generate with js_dir argument will now install node modules if they're not found
- better error handling
- Update copyright notice to use the legal name of AI Redefined Inc.

## _Orchestrator_ - v1.0.1 - 2021-06-02

### Changed

- Cleanup logs and add trace logs
- Fix environment message sending (to only send when there are messages)
- Add SIGSEGV trapping and reporting
- Add log file output option

## _CLI_ - 1.0.0 - 2021-05-11

## _Orchestrator_ - v1.0.0 - 2021-05-10

- Initial public release.

### Fixed

- Environment can now receive messages
- The parameter `max_steps` now works

## _CLI_ - 1.0.0-beta3 - 2021-04-27

- Upgrade the version of cogment-orchestrator to `v1.0.0-beta3`
- Upgrade the version of cogment-py-sdk to `v1.0.0-beta3`

## _Orchestrator_ - v1.0.0-beta3 - 2021-04-26

### Added

- Add implementation for the GetTrialInfo function

### Fixed

- Fill in missing API data in the datalog `DatalogSample.TrialData` protobuf class
- Fix timing problems causing various issues

## _Orchestrator_ - v1.0.0-beta2 - 2021-04-15

### Fixed

- Fixed the problem where the last rewards from the environment would get to the actors too late
- Fixed the filling of DatalogSample (actions, rewards and messages were missing)
- Fixed one deadlock with actor responses

## _CLI_ - 1.0.0-beta1 - 2021-04-08

- Initial beta release, no more breaking changes should be introduced.

### Fixed

- Generated web-client no longer include a `.git`

## _Orchestrator_ - v1.0.0-beta1 - 2021-04-07

- Initial beta release, no more breaking changes should be introduced.

## _Orchestrator_ - v1.0.0-alpha9 - 2021-04-01

### Changed

- Rename ActorClass `id` to `name`

## _CLI_ - 1.0.0-alpha10 - 2021-04-01

### Changed

- `ActorClass`'s `id` field is now named `name` everywhere.
- `cogment init` supports the simplified event data structure in actor and environment event loops.
- `cogment init` uses `cogment.Endpoint` and `cogment.ServedEndpoint` instead of raw TCP ports.
- `cogment init` supports the new controller API.
- `cogment init` supports the new `RecvAction` & `RecvReward` classes.
- Upgrade the version of cogment-orchestrator to `v1.0.0-alpha9`
- Upgrade the version of cogment-py-sdk to `v1.0.0-alpha12`

### Fixed

- The default `cogment run start` properly attach to the actors and environment services.
- The default `cogment run build` properly build all the services.
- Fix `cogment generate` to properly support `import` in proto files.

## _Orchestrator_ - v1.0.0-alpha8 - 2021-03-30

- Technical release, updating dependencies to fixed versions.

## _Orchestrator_ - v1.0.0-alpha7 - 2021-03-30

### Added

- Log exporter is now available: trials param, observations, rewards, messages of every trials are sent to the log exporter service.

### Changed

- Tick ID management centralized in orchestrator

## _Orchestrator_ - v1.0.0-alpha6 - 2021-03-10

### Added

- Watch trials is now supported

## _CLI_ - 1.0.0-alpha9 - 2021-03-01

## _CLI_ - 1.0.0-alpha8 - 2021-02-25

## _CLI_ - 1.0.0-alpha7 - 2021-02-24

## _CLI_ - 1.0.0-alpha6 - 2021-02-23

### Changed

- Update to use github.com as import target

## _CLI_ - 1.0.0-alpha5 - 2021-02-22

### Added

- Add both the creation and generation of a web-client through `cogment init` and `cogment generate`, respectively. A node.js distribution is now required to be available on `$PATH` for certain features. These are disabled by default and must be enabled by stdin or flags.

### Changed

- `cogment generate` now accepts `--python-out` instead of `--python_out`. `--python-out` can be repeated multiple times to target multiple output directories, eg: `cogment generate --python-out environment --python-out client --python-out actor`.
- `cogment generate` now accepts the `--js-out` flag, which enables generation of protobuf definitions and `CogSettings.ts` from a `cogment.yaml`
- `cogment generate` now accepts a `--typescript` flag that depends on the `--js-out` flag, will enable typing definition generation for user protobufs. This can be repeated multiple times to target multiple output directories, just like python-out
- `cogment init` templates uses dependencies between services (`depends_on` clause) for bringing up the stack vs. having service names repeated in `cogment run` commands. `docker-compose up web-client` will bring up the entire stack, `docker-compose up dashboard` will bring up all the necessary containers.

## _Orchestrator_ - v1.0.0-alpha5 - 2021-02-19

### Added

- Add support for messages and rewrads, they can be sent from the actor client.
- Addition for tls communication

## _CLI_ - 1.0.0-alpha4 - 2021-02-19

## _CLI_ - 1.0.0-alpha3 - 2021-02-19

### Changed

- `cogment init` now generates actor & environment implementations handling all the possible events.
- Initialize Metrics and Dashboard when `cogment init` is run

## _Orchestrator_ - v1.0.0-alpha4 - 2021-02-17

### Added

- Add support for messages and rewrads, they can be sent from the actor client.

## _Orchestrator_ - v1.0.0-alpha3 - 2021-01-28

### Fixed

- Fix a crash occuring at the end of trials.
- Fix occasional crash when employing prehooks.

## _CLI_ - 1.0.0-alpha2 - 2021-01-28

### Changed

- `cogment run` now runs the command in the `cogment.yaml` directory.
- The `cogment init` prompt integrate the concept of actor implementation and make it optional to create an actor in the client.
- Upgrade the version of cogment-orchestrator to `v1.0.0-alpha3`
- Upgrade the version of cogment-py-sdk to `v1.0.0-alpha5`

### Fixed

- `cogment init` now generates a working project structure and implementation for actor, client and environment services.
- Build and publish a `latest` tag for `cogment/cli` on dockerhub at <https://hub.docker.com/r/cogment/cli>.
- `cogment init` on project whose name contains a `-` now generates valid `.proto` files.

## _Orchestrator_ - v1.0.0-alpha2 - 2021-01-11

### Added

- Add support for messages, they can be sent between actors and the environment.
- Add dispatch of immediate rewards to actors.

## _CLI_ - v1.0.0-alpha1 - 2020-12-07

- Initial alpha release, expect some breaking changes.

### Known issues

- Files generated by `cogment init` are not up-to-date.

### Added

- Add build script for macOS
- Support for discovery endpoints (i.e. to query the directory)
- Support for providing the parameters with the StartTrial rpc

### Fixed

- Fix the build on macOS
- Have the datalog client consume the stream for the server

### Fixed

- Fix the logging levels, making the default less verbose.

## _Orchestrator_ - v1.0.0-alpha1 - 2020-12-07

- Initial alpha release, expect some breaking changes.
