# cogment-orchestrator

## Building the orchestrator

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


## Auto-formatting code

The following will apply clang-format to all included source, with the exception of the third_party directory:

```
make format
```