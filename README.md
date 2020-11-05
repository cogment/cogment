# cogment-orchestrator

## Introduction

The Cogment framework is a high-efficiency, open source framework designed to enable the training of models in environments where humans and agents interact with the environment and each other continuously. It’s capable of distributed, multi-agent, multi-model training.

The Orchestrator is the central entity in the Cogment framework that ties all the services together. From the perspective of a Cogment framework user, it can be considered as the live interpreter of the cogment.yaml configuration file. It is the service that client applications will connect to in order to start and run trials.

For further Cogment information, check out the documentation at <https://docs.cogment.ai>

## Developers

### Building the orchestrator

The orchestrator has a few dependecies:

* googletest
* grpc
* easy-grpc
* var_futures
* yaml-cpp
* spdlog

The easiest way to build the orchestrator is to make use of [cogment build env](https://gitlab.com/ai-r/cogment-build-env)
Docker image:

```
docker run --rm -it -v$(pwd):/workspace registry.gitlab.com/ai-r/cogment-build-env:latest
mkdir _bld
cd _bld
cmake ..
make
```

### Used Cogment protobuf API

The version of the used cogment protobuf API is defined in the `.cogment-api.yml` file at the root of the repository.

To use a **published version**, set `cogment_api_version`, by default it retrieved the _latest_ version of cogment api, i.e. the one on the `develop` branch for cogment-api.

To use a **local version**, set `cogment_api_path` to the absolute path to the local cogment-api directory. When set, this local path overrides any `cogment_api_version` set.

> ⚠️ when building in docker you need to mount the path to cogment api in the docker image. you can for example add `-v$(pwd)/../cogment-api/:/workspace/cogment-api` to the above `docker run` command line.

> For a change to be taken into account, you'll need to re-run `cmake`.

> ⚠️ Due to the limited parsing capabilities of CMake, commented out fully defined variables won't be ignored...

### Auto-formatting code

The following will apply clang-format to all included source, with the exception of the third_party directory:

```
make format
```
