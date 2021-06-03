# cogment-cli

[![Latest GitHub release](https://img.shields.io/github/v/release/cogment/cogment-cli?label=binary%20release&sort=semver&style=flat-square)](https://github.com/cogment/cogment-cli/releases) [![Latest Docker release](https://img.shields.io/docker/v/cogment/cli?label=docker%20release&sort=semver&style=flat-square)](https://hub.docker.com/r/cogment/cli) [![Apache 2 License](https://img.shields.io/badge/license-Apache%202-green?style=flat-square)](./LICENSE) [![Changelog](https://img.shields.io/badge/-Changelog%20-blueviolet?style=flat-square)](./CHANGELOG.md)

[Cogment](https://cogment.ai) is an innovative open source AI platform designed to leverage the advent of AI to benefit humankind through human-AI collaboration developed by [AI Redefined](https://ai-r.com). Cogment enables AI researchers and engineers to build, train and operate AI agents in simulated or real environments shared with humans. For the full user documentation visit <https://docs.cogment.ai>

This module, `cogment-cli`, is a command line tool providing a set of useful command utilities that
provide the following functions:

- generate, perform the code generation phase from a project's proto files.
- init, bootstrap a new project locally.
- run, run a command from the cogment.yaml 'commands' section.
- version, print the version nummber of the Cogment CLI.

## Developers

### Prerequisites

- Fully working **go** setup, as described in the [official documentation](https://golang.org/doc/install).
- **`Make`**, most flavor should work fine.
- **`protoc`**, the protocol buffer compiler, it should be installed as described in the [official documentation](https://github.com/protocolbuffers/protobuf#protocol-compiler-installation).

### Formatting & coding style

To check for the format of the file using [gofmt](https://golang.org/cmd/gofmt/), run the following:

```shell script
make check-fmt
```

To reformat the code to fix issues, simply run the following:

```shell script
make fmt
```

To run a coding style check using [golint](https://github.com/golang/lint), run the following:

```shell script
make check-codingstyle
```

> ⚠️ The full codebase does not yet conforms to the coding style, expect some errors.

### Tests

Run the tests using the following:

```shell script
make test
```

Some tests validate output against snapshots using [cupaloy](https://github.com/bradleyjkemp/cupaloy), you can update those snapshots using

```shell script
make test-update-snapshots
```

Before committing updated snapshot, review their content.

### Build

Build the executable for your platform to `./build/congment` using the following:

```shell script
make build
```

Cross compile to all the supported platforms to `./build/` using the following:

```shell script
make release
```

### Usage

After running `make build` you can simply put `./build/cogment` in your `$PATH` and simply run the following:

```shell script
cogment
```

Alternatively, you can run `make install`, this will install the executable in `$(go env GOPATH)/bin`, a directory that should already be part of your `$PATH`.

In this case the binary will be named `cogment-cli` and you can simply run

```shell script
cogment-cli
```

### Release process

People having maintainers rights of the repository can follow these steps to release a version **MAJOR.MINOR.PATCH**. The versioning scheme follows [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

1. Run `./scripts/create_release_branch.sh` automatically compute and update the version of the package, create the release branch and update the changelog from the commit history,
2. On the release branch, check and update the changelog if needed
3. Update api > default_cogment.yaml
4. Make sure everything's fine on CI,
5. Run `./scripts/tag_release.sh MAJOR.MINOR.PATCH` to create the specific version section in the changelog, merge the release branch in `main`, create the release tag and update the `develop` branch with those.

The rest, publishing the packages to github releases and dockerhub and updating the mirror repositories, is handled directly by the CI.
