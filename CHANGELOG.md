# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## Unreleased

## v0.3.0 - 2021-12-14

### Changed

- The model registry now stored transient model versions in a memory cache.

### Added

- Introduce `backend.MemoryCacheBackend` a cache backend that stores model version in a upper bounded memory cache and uses another backend for archived versions.

### Removed

- `backend.DbBackend` and `backend.HybridBackend` are no longer used and have been removed.

## v0.2.0 - 2021-10-05

### Fixed

- Fix `database is locked` errors during the initial sync by configuring SQLite for concurrent access.

## v0.1.0 - 2021-10-01

Introduce Model Registry, a simple versioned key-value store for models
