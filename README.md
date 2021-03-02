# cogment-cli

> ⚠️ 🚧 This is part of an upcoming release of cogment and still unstable.
>
> Current stable version can be found at <https://gitlab.com/cogment/cogment>

[![Latest GitHub release](https://img.shields.io/github/v/release/cogment/cogment-cli?label=binary%20release&sort=semver&style=flat-square)](https://github.com/cogment/cogment-cli/releases) [![Latest Docker release](https://img.shields.io/docker/v/cogment/cli?label=docker%20release&sort=semver&style=flat-square)](https://hub.docker.com/r/cogment/cli) [![Apache 2 License](https://img.shields.io/badge/license-Apache%202-green?style=flat-square)](./LICENSE) [![Changelog](https://img.shields.io/badge/-Changelog%20-blueviolet?style=flat-square)](./CHANGELOG.md)

## Introduction

The Cogment framework is a high-efficiency, open source framework designed to enable the training of models in environments where humans and agents interact with the environment and each other continuously. It’s capable of distributed, multi-agent, multi-model training.

The cogment CLI tool provides a set of useful command utilities that
provide the following functions -

- generate (generate the cog_settings.py and compile your proto files)
- init (bootstrap a new project locally)
- run (run a command from the cogment.yaml 'commands' section)
- version (print the version nummber of the Cogment CLI)

For further Cogment information, check out the documentation at <https://docs.cogment.ai>

## Developers

The cogment cli can be built and used as a docker image or a standalone executable.

### Standalone executable

#### Prerequisites

- Fully working **go** setup, as described in the [official documentation](https://golang.org/doc/install).
- **`Make`**, most flavor should work fine.
- **`protoc`**, the protocol buffer compiler, it should be installed as described in the [official documentation](https://github.com/protocolbuffers/protobuf#protocol-compiler-installation).

#### Formatting & coding style

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

#### Tests

Run the tests using the following:

```shell script
make test
```

Some tests validate output against snapshots using [cupaloy](https://github.com/bradleyjkemp/cupaloy), you can update those snapshots using

```shell script
make test-update-snapshots
```

Before committing updated snapshot, review their content.

#### Build

Build the executable for your platform to `./build/congment` using the following:

```shell script
make build
```

Cross compile to all the supported platforms to `./build/` using the following:

```shell script
make release
```

#### Usage

After running `make build` you can simply put `./build/cogment` in your `$PATH` and simply run the following:

```shell script
cogment
```

Alternatively, you can run `make install`, this will install the executable in `$(go env GOPATH)/bin`, a directory that should already be part of your `$PATH`.

In this case the binary will be named `cogment-cli` and you can simply run

```shell script
cogment-cli
```
