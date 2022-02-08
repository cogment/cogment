# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## Unreleased

### Changed

- Change the default size of the pruned versions group from 50 to 5 to avoid situations where the latest version of a model was pruned as it was inserted.
- Improve the error message when retrieving a n-th to last version that does not exist.

## v0.5.0 - 2022-02-01

### Added

- Implement `cogmentAPI.ModelRegistrySP/RetrieveModels`, the method able to retrieve models and their data.

### Fixed

- Examples in the readme now uses the correct protobuf namespace.

## v0.4.0 - 2022-01-19

### Added

- Add the ability to retrieve any n-th to last version to `cogmentAPI.ModelRegistrySP/RetrieveVersionInfos`, `cogmentAPI.ModelRegistrySP/RetrieveVersionData`.

### Changed

- **Breaking Change** Update Cogment API to 2.0
- Internal `backend.Backend` now uses `uint` for version numbers and uses 0 to request the creation of a new version.

## v0.3.0 - 2021-12-14

### Changed

- The model registry now stores transient model versions in a memory cache.

### Added

- Introduce `backend.MemoryCacheBackend` a cache backend that stores model version in a upper bounded memory cache and uses another backend for archived versions.

### Removed

- `backend.DbBackend` and `backend.HybridBackend` are no longer used and have been removed.

## v0.2.0 - 2021-10-05

### Fixed

- Fix `database is locked` errors during the initial sync by configuring SQLite for concurrent access.

## v0.1.0 - 2021-10-01

Introduce Model Registry, a simple versioned key-value store for models
