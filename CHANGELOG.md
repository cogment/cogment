# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## Unreleased

### Fixed

- Fix `database is locked` errors during the initial sync by configuring SQLite for concurrent access.

## v0.1.0 - 2021-10-01

Introduce Model Registry, a simple versioned key-value store for models
