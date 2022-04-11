# cogment

[![Latest Release](https://img.shields.io/github/v/release/cogment/cogment?label=latest%20release&sort=semver&style=flat-square)](https://github.com/cogment/cogment/releases)
[![Latest Docker Release](https://img.shields.io/docker/v/cogment/cogment?label=latest%20docker%20release&sort=semver&style=flat-square)](https://hub.docker.com/r/cogment/cogment) [![Apache 2 License](https://img.shields.io/badge/license-Apache%202-green?style=flat-square)](./LICENSE) [![Changelog](https://img.shields.io/badge/-Changelog%20-blueviolet?style=flat-square)](./CHANGELOG.md)

[Cogment](https://cogment.ai) enables AI researchers and engineers to build, train, and operate AI agents in simulated or real environments shared with humans. Developed by [AI Redefined](https://ai-r.com), Cogment is the first open source platform designed to address the challenges of continuously training humans and AI together. For the full user documentation visit <https://docs.cogment.ai>

This repository includes the the main Cogment module, a multiplatform stand alone CLI including:

- The **orchestrator** service, the heart of Cogment, it executes the trials involving actors and environments by orchestrating the different user implemented services.
- The **trial datastore** service, that is able to store and make available the data generated by the trials.
- The **model registry** service, that let's user store AI models and make them avaible to actor implementations during training and in production.
- The **init** tool to bootstrap cogment project locally (Deprecated).
- The **run** tool to define and run commands within a cogment project (Deprecated).

## Installation

### Standalone binary (preferred)

Download the install script and make sure you can run it using

```console
curl --silent -L https://raw.githubusercontent.com/cogment/cogment/main/install.sh --output install-cogment.sh
chmod +x install-cogment.sh
```

Download and install the latest final version using

```console
sudo ./install-cogment.sh
```

Other installation options are available using `./install-cogment.sh --help`

For futher installation options please refer to cogment's [installation guide](https://cogment.ai/docs/cogment/installation/).

### Docker image

```console
docker pull cogment/cogment
```

## Developers

This repository is organized in 2 different packages: [`orchestrator`](#orchestrator) is the orchestrator itself, developed in C++, [`cogment`](#cogment) is the host tool, developed in Go, it integrates the orchestrator and includes the other services and tools.

A [CMake](https://cmake.org) based systems glues the build of both together

#### Prerequisites

- Fully working **c++** build toolchain.
- Fully working **go** setup (1.16), as described in the [official documentation](https://golang.org/doc/install).
- **`Cmake`** (>= 3.10), the core of the build system, it should be installed as described in the [official documentation](https://cmake.org/install/).
- **`Make`**, most flavor should work fine.
- _Optional_ a [**docker**](https://www.docker.com/) installation to be able to build the docker image.

### Build

At the root of the repository you'll find the following scripts:

- `build_docker.sh` builds the docker image.
- `build_linux.sh` builds the linux amd64 binary.
- `build_macos.sh` builds the macos amd64 binary.
- `build_windows.bat` builds the windows amd64 binary.
- `build_all_no_orchestrator.sh` builds the binary for all supported platforms (linux/amd64, macos/amd64, macos/arm64 and windows/amd64) without the embedded orchestrator.

The build results are store in the `./install` directory per platform.

Those scripts run CMake and create a build directory, per platform, in the `./build` directory. Further build target are available there, especially for testing of code formatting.

### Formatting & coding style

- `orchestrator_lint` and `orchestrator_fix_lint` respectively check and fix the code formatting of the c++ orchestrator codebase using [`clang-format`](https://clang.llvm.org/docs/ClangFormat.html).
- `cogment_lint` and `cogment_fix_lint` respectively check and fix the code formatting of the go cogment codebase using [`golangci-lint`](https://golangci-lint.run)

### Test

- `cogment_test` runs a suite of tests over the go cogment codebase.

### Benchmark

- `cogment_benchmark` runs a suite of benchmarks over the go cogment codebase.

### Release process

People having mainteners rights of the repository can follow these steps to release a version **MAJOR.MINOR.PATCH**. The versioning scheme follows [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

1. Run `./scripts/create_release_branch.sh MAJOR.MINOR.PATCH` to create the release branch and update the version of the package,
2. On the release branch, check and update the changelog if needed,
3. Update .cogment-api.yaml to the latest version of cogment-api
4. Make sure everything's fine on CI,
5. Run `./scripts/tag_release.sh MAJOR.MINOR.PATCH` to create the specific version section in the changelog, merge the release branch in `main`, create the release tag and update the `develop` branch with those.

The rest, publishing the package to dockerhub and updating the mirror repositories, is handled directly by the CI.
