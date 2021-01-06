# cogment-cli

> ⚠️ 🚧 This is part of an upcoming release of cogment and still unstable.
>
> Current stable version can be found at <https://gitlab.com/cogment/cogment>


[![Latest GitHub release](https://img.shields.io/github/v/release/cogment/cogment-cli?label=binary%20release&sort=semver&style=flat-square)](https://github.com/cogment/cogment-cli/releases) [![Latest Docker release](https://img.shields.io/docker/v/cogment/cli?label=docker%20release&sort=semver&style=flat-square)](https://hub.docker.com/r/cogment/cli)  [![Apache 2 License](https://img.shields.io/badge/license-Apache%202-green?style=flat-square)](./LICENSE) [![Changelog](https://img.shields.io/badge/-Changelog%20-blueviolet?style=flat-square)](./CHANGELOG.md)


## Introduction

The Cogment framework is a high-efficiency, open source framework designed to enable the training of models in environments where humans and agents interact with the environment and each other continuously. It’s capable of distributed, multi-agent, multi-model training.

The cogment CLI tool provides a set of useful command utilities that provide the following functions -
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

```
make check-fmt
```

To reformat the code to fix issues, simply run the following:

```
make fmt
```

To run a coding style check using [golint](https://github.com/golang/lint), run the following:

```
make check-codingstyle
```

> ⚠️ The full codebase does not yet conforms to the coding style, expect some errors.

#### Tests

Run the tests using the following:

```
make test
```

Some tests validate output against snapshots using [cupaloy](https://github.com/bradleyjkemp/cupaloy), you can update those snapshots using

```
make test-update-snapshots
```

Before committing updated snapshot, review their content.

#### Build

Build the executable for your platform to `./build/congment` using the following:

```
make build
```

Cross compile to all the supported platforms to `./build/` using the following:

```
make release
```

#### Usage

Built executables, in `./build/` can be used as-is.

However the best way to install the cli locally is to run `make install`, this will the executable in `$(go env GOPATH)/bin`, a directry that should already be part of your `$PATH`.

To use the cogment cli, simply run the following:

```
cogment
```

### Dockerized version

#### Prerequisites

- Fully working **Docker Desktop** setup, as described in the [official documentation](https://www.docker.com/products/docker-desktop).

#### Build

Build the docker image and tag it as `cogment/cli:latest` using the following:

```
docker build -t cogment/cli:latest .
```

#### Usage

The image expects the current cogment project to be mounted at `/cogment`. Also, if you want to be able to execute commands that manipulate the host's docker (such as building or running a project), then you need
to bind your host's docker socket

On windows:

```
docker run --rm -v%cd%:/cogment -v//var/run/docker.sock://var/run/docker.sock cogment/cli:latest run build
```

On linux and macOS:
```
docker run --rm -v$(pwd):/cogment -v/var/run/docker.sock:/var/run/docker.sock cogment/cli:latest run build
```

You may want to create an alias as a quick Quality of Life improvement:

```
alias cogment='docker run --rm -v$(pwd):/cogment -v/var/run/docker.sock:/var/run/docker.sock cogment/cli:latest'

cogment run build
```

> ⚠️ Take care to use simple quotes to create your alias, i.e. use `alias cogment='...'` for it to work properly.
