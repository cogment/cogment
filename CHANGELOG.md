# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## Unreleased

## v0.2.0 - 2022-01-19

### Changed

- **Breaking Change** Update Cogment API to 2.0, no longer support API v1.X

## v0.1.2 - 2021-10-25

### Fixed

- Fix bad data initialization that would cause a nil pointer dereference when receiving samples through the datalog server

## v0.1.1 - 2021-10-22

### Fixed

- Fix naming collision in cogment gRPC API

## v0.1.0 - 2021-10-20

Introduce Trial Datastore, a simple logger & store for trial generated data.
