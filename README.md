# cogment-cli

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

### Building

```
docker build -t cogment-cli .
```

### Running

The image expects the current cogment project to be mounted at `/cogment`. Also, if you want to be able to execute commands that manipulate the host's docker (such as building or running a project), then you need
to bind your host's docker socket

On windows:
```
docker run --rm -v%cd%:/cogment -v//var/run/docker.sock://var/run/docker.sock cogment-cli run build
```

On linux and macOS:
```
docker run --rm -v$(pwd):/cogment -v/var/run/docker.sock:/var/run/docker.sock cogment-cli run build
```

You may want to create an alias as a quick Quality of Life improvement:

```
alias cogment="docker run --rm -v$(pwd):/cogment -v//var/run/docker.sock://var/run/docker.sock cogment-cli"

cogment run build
```

### Formatting

Having a working go installation locally, run the following to check for the format of the file using [gofmt](https://golang.org/cmd/gofmt/):

```
make lint
```

To reformat the code to fix issues, simply run the following:

```
make fmt
```
