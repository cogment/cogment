# Cogment Model Registry

[![Latest Release](https://img.shields.io/docker/v/cogment/model-registry?label=docker%20release&sort=semver&style=flat-square)](https://hub.docker.com/r/cogment/model-registry) [![Apache 2 License](https://img.shields.io/badge/license-Apache%202-green?style=flat-square)](./LICENSE) [![Changelog](https://img.shields.io/badge/-Changelog%20-blueviolet?style=flat-square)](./CHANGELOG.md)

[Cogment](https://cogment.ai) is an innovative open source AI platform designed to leverage the advent of AI to benefit humankind through human-AI collaboration developed by [AI Redefined](https://ai-r.com). Cogment enables AI researchers and engineers to build, train and operate AI agents in simulated or real environments shared with humans. For the full user documentation visit <https://docs.cogment.ai>

This module, cogment Model Registry, is a versioned key value store dedicated to the storage of models.

## Usage

```console
$ docker run -p 9000:9000 -v $(pwd)/relative/path/to/model/archive:/data cogment/model-registry
```

### Configuration

The following environment variables can be used to configure the server:

- `COGMENT_MODEL_REGISTRY_PORT`: The port to listen on. Defaults to 9000.
- `COGMENT_MODEL_REGISTRY_ARCHIVE_DIR`: The directory to store model archives. Docker images defaults to `/data`.
- `COGMENT_MODEL_REGISTRY_SENT_MODEL_VERSION_DATA_CHUNK_SIZE`: The size of the model version data chunk sent by the server. Defaults to 5*1024*1024 (5MB).
- `COGMENT_MODEL_REGISTRY_GRPC_REFLECTION`: Set to start a [gRPC reflection server](https://github.com/grpc/grpc/blob/master/doc/server-reflection.md). Defaults to `false`.

## API

The Model Registry exposes a gRPC defined in the [Model Registry API](grpcapi/cogment/api/model_registry.proto) (This will be moved to the Cogment gRPC API in the future).

### Create a model - `cogment.ModelRegistry/CreateModel( .cogment.CreateModelRequest ) returns ( .cogment.CreateModelReply );`

```console
$ echo "{\"model_id\":\"my_model\"}" | grpcurl -plaintext -d @ localhost:9000 cogment.ModelRegistry/CreateModel
{
  "model": {
    "modelId": "my_model"
  }
}
```

### Create a model version - `cogment.ModelRegistry/CreateModelVersion( stream .cogment.CreateModelVersionRequestChunk ) returns ( .cogment.CreateModelVersionReply );`

```console
$ echo "{\
    \"model_id\":\"my_model\",\
    \"archive\":true,\
    \"data_chunk\":\"$(echo chunk_1 | base64)\"\
  }\
  {\
    \"data_chunk\":\"$(echo chunk_2 | base64)\",\
    \"last_chunk\":true\
  }" | grpcurl -plaintext -d @ localhost:9000 cogment.ModelRegistry/CreateModelVersion
{
  "versionInfo": {
    "modelId": "my_model",
    "createdAt": "2021-08-31T13:46:51.372417Z",
    "number": 1,
    "archive": true,
    "hash": "1FoOg5QHj7ctiXEZkr9rDkfJWrsy//DeUsHEmP7PcgA="
  }
}
```

### List model versions - `cogment.ModelRegistry/ListModelVersions ( .cogment.ListModelVersionsRequest ) returns ( .cogment.ListModelVersionsReply );`

```console
$ echo "{\"model_id\":\"my_model\"}" | grpcurl -plaintext -d @ localhost:9000 cogment.ModelRegistry/ListModelVersions
{
  "versions": [
    {
      "modelId": "my_model",
      "createdAt": "2021-08-31T13:42:59.399405Z",
      "number": 1,
      "archive": true,
      "hash": "47DEQpj8HBSa+/TImW+5JCeuQeRkm5NMpJWZG3hSuFU="
    },
    {
      "modelId": "my_model",
      "createdAt": "2021-08-31T13:45:04.092807Z",
      "number": 2,
      "archive": true,
      "hash": "1FoOg5QHj7ctiXEZkr9rDkfJWrsy//DeUsHEmP7PcgA="
    },
  ],
  "nextPageOffset": 2
}
```

### Retrieve given version info - `cogment.ModelRegistry/RetrieveModelVersionInfo ( .cogment.RetrieveModelVersionInfoRequest ) returns ( .cogment.RetrieveModelVersionInfoReply );`


```console
$ echo "{\"model_id\":\"my_model\", \"number\":1}" | grpcurl -plaintext -d @ localhost:9000 cogment.ModelRegistry/RetrieveModelVersionInfo
{
  "versionInfo": {
    "modelId": "my_model",
    "createdAt": "2021-08-31T13:42:59.399405Z",
    "number": 1,
    "archive": true,
    "hash": "1FoOg5QHj7ctiXEZkr9rDkfJWrsy//DeUsHEmP7PcgA="
  }
}
```

To retrieve the latest version, use `number:-1`.

### Retrieve given version data - `cogment.ModelRegistry/RetrieveModelVersionData ( .cogment.RetrieveModelVersionDataRequest ) returns ( stream .cogment.RetrieveModelVersionDataReplyChunk );`

```console
$ echo "{\"model_id\":\"my_model\", \"number\":1}" | grpcurl -plaintext -d @ localhost:9000 cogment.ModelRegistry/RetrieveModelVersionData
{
  "dataChunk": "Y2h1bmtfMQpjaHVua18yCg==",
  "lastChunk": true
}
```

To retrieve the latest version, use `number:-1`.

## Developers

### With a local Go installation

Linting is based on the "meta" linter [golangci-lint](https://golangci-lint.run) it needs to be installed locally:

```console
$ make lint
```

Testing:

```console
$ make test
```

Testing with JUnit style reports (for the CI):

```
$ make test-with-report
```

Build a binary in `build/cogment-model-registry`:

```
$ make build
```

### With Docker

Build image

```
$ ./scripts/build_docker.sh
```
