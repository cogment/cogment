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


### Auto-formatting code

The following will apply clang-format to all included source, with the exception of the third_party directory:

```
make format
```