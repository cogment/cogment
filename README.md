# Cogment Model Registry

[![Latest Release](https://img.shields.io/docker/v/cogment/model-registry?label=docker%20release&sort=semver&style=flat-square)](https://hub.docker.com/r/cogment/model-registry) [![Apache 2 License](https://img.shields.io/badge/license-Apache%202-green?style=flat-square)](./LICENSE) [![Changelog](https://img.shields.io/badge/-Changelog%20-blueviolet?style=flat-square)](./CHANGELOG.md)

[Cogment](https://cogment.ai) is an innovative open source AI platform designed to leverage the advent of AI to benefit humankind through human-AI collaboration developed by [AI Redefined](https://ai-r.com). Cogment enables AI researchers and engineers to build, train and operate AI agents in simulated or real environments shared with humans. For the full user documentation visit <https://docs.cogment.ai>

This module, cogment Model Registry, is a versioned key value store dedicated to the storage of models.

- **Transient** (non-archived) model versions can be used to broadcast an updated version of a model to its users, e.g. during training. Transient model versions are stored in memory and can be evicted, in particular once the memory limit is reached.
- **Archived** model versions are stored on the filesystem, they are slower to create and to retrieve and should be used for long-term storage of specific versions, e.g. for validation or deployment purposes.

## Usage

```console
$ docker run -p 9000:9000 -v $(pwd)/relative/path/to/model/archive:/data cogment/model-registry
```

### Configuration

The following environment variables can be used to configure the server:

- `COGMENT_MODEL_REGISTRY_PORT`: The port to listen on. Defaults to 9000.
- `COGMENT_MODEL_REGISTRY_ARCHIVE_DIR`: The directory to store model archives. Docker images defaults to `/data`.
- `COGMENT_MODEL_REGISTRY_VERSION_CACHE_MAX_SIZE`: The maximum cumulated size for the model versions stored in memory. Defaults to 1024 \* 1024 \* 1024 (1GB).
- `COGMENT_MODEL_REGISTRY_VERSION_CACHE_EXPIRATION`: The maximum duration for which model versions are kept in memory. Defaults to 1h, can be set to any value parsable by [`time.ParseDuration`](https://pkg.go.dev/time#ParseDuration).
- `COGMENT_MODEL_REGISTRY_VERSION_CACHE_TO_PRUNE_COUNT`: The amount of model versions that are removed from cache at once. Defaults to 5.
- `COGMENT_MODEL_REGISTRY_SENT_MODEL_VERSION_DATA_CHUNK_SIZE`: The size of the model version data chunk sent by the server. Defaults to 5 \* 1024 \* 1024 (5MB).
- `COGMENT_MODEL_REGISTRY_GRPC_REFLECTION`: Set to start a [gRPC reflection server](https://github.com/grpc/grpc/blob/master/doc/server-reflection.md). Defaults to `false`.

## API

The Model Registry exposes a gRPC defined in the [Model Registry API](https://github.com/cogment/cogment-api/blob/main/model_registry.proto)

### Create or update a model - `cogmentAPI.ModelRegistrySP/CreateOrUpdateModel( .cogmentAPI.CreateOrUpdateModelRequest ) returns ( .cogmentAPI.CreateOrUpdateModelReply );`

_This example requires `COGMENT_MODEL_REGISTRY_GRPC_REFLECTION` to be enabled and requires [grpcurl](https://github.com/fullstorydev/grpcurl)_

```console
$ echo "{\"model_info\":{\"model_id\":\"my_model\",\"user_data\":{\"type\":\"my_model_type\"}}}" | grpcurl -plaintext -d @ localhost:9000 cogmentAPI.ModelRegistrySP/CreateOrUpdateModel
{

}
```

### Delete a model - `cogmentAPI.ModelRegistrySP/DeleteModel( .cogmentAPI.DeleteModelRequest ) returns ( .cogmentAPI.DeleteModelReply );`

_This example requires `COGMENT_MODEL_REGISTRY_GRPC_REFLECTION` to be enabled and requires [grpcurl](https://github.com/fullstorydev/grpcurl)_

```console
$ echo "{\"model_id\":\"my_model\"}" | grpcurl -plaintext -d @ localhost:9000 cogmentAPI.ModelRegistrySP/DeleteModel
{

}
```

### Retrieve models - `cogmentAPI.ModelRegistrySP/RetrieveModels( .cogmentAPI.RetrieveModelsRequest ) returns ( .cogmentAPI.RetrieveModelsReply );`

_These examples requires `COGMENT_MODEL_REGISTRY_GRPC_REFLECTION` to be enabled and requires [grpcurl](https://github.com/fullstorydev/grpcurl)_

#### List the models

```console
$ echo "{}" | grpcurl -plaintext -d @ localhost:9000 cogmentAPI.ModelRegistrySP/RetrieveModels
{
  "modelInfos": [
    {
      "modelId": "my_model",
      "userData": {
        "type": "my_model_type"
      }
    },
    {
      "modelId": "my_other_model",
      "userData": {
        "type": "my_model_type"
      }
    }
  ],
  "nextModelHandle": "2"
}
```

#### Retrieve specific model(s)

```console
$ echo "{\"model_ids\":[\"my_other_model\"]}" | grpcurl -plaintext -d @ localhost:9000 cogmentAPI.ModelRegistrySP/RetrieveModels
{
  "modelInfos": [
    {
      "modelId": "my_other_model",
      "userData": {
        "type": "my_model_type"
      }
    }
  ],
  "nextModelHandle": "1"
}
```

### Create a model version - `cogmentAPI.ModelRegistrySP/CreateVersion( stream .cogmentAPI.CreateVersionRequestChunk ) returns ( .cogmentAPI.CreateVersionReply );`

_This example requires `COGMENT_MODEL_REGISTRY_GRPC_REFLECTION` to be enabled and requires [grpcurl](https://github.com/fullstorydev/grpcurl)_

```console
$ echo "{\"header\":{\"version_info\":{
    \"model_id\":\"my_model\",\
    \"archived\":true,\
    \"data_size\":$(printf chunk_1chunk_2 | wc -c)\
  }}}\
  {\"body\":{\
    \"data_chunk\":\"$(printf chunk_1 | base64)\"\
  }}\
  {\"body\":{\
    \"data_chunk\":\"$(printf chunk_2 | base64)\"\
  }}" | grpcurl -plaintext -d @ localhost:9000 cogmentAPI.ModelRegistrySP/CreateVersion
{
  "versionInfo": {
    "modelId": "my_model",
    "versionNumber": 2,
    "creationTimestamp": "907957639",
    "archived": true,
    "dataHash": "jY0g3VkUK62ILPr2JuaW5g7uQi0EcJVZJu8IYp3yfhI=",
    "dataSize": "14"
  }
}
```

### Retrieve model versions infos - `cogmentAPI.ModelRegistrySP/RetrieveVersionInfos ( .cogmentAPI.RetrieveVersionInfosRequest ) returns ( .cogmentAPI.RetrieveVersionInfosReply );`

_These examples require `COGMENT_MODEL_REGISTRY_GRPC_REFLECTION` to be enabled and requires [grpcurl](https://github.com/fullstorydev/grpcurl)_

#### List the versions of a model

```console
$ echo "{\"model_id\":\"my_model\"}" | grpcurl -plaintext -d @ localhost:9000 cogmentAPI.ModelRegistrySP/RetrieveVersionInfos
{
  "versionInfos": [
    {
      "modelId": "my_model",
      "versionNumber": 1,
      "creationTimestamp": "1633119005107454620",
      "archived": true,
      "dataHash": "jY0g3VkUK62ILPr2JuaW5g7uQi0EcJVZJu8IYp3yfhI=",
      "dataSize": "14"
    },
    {
      "modelId": "my_model",
      "versionNumber": 2,
      "creationTimestamp": "1633119625907957639",
      "archived": true,
      "dataHash": "jY0g3VkUK62ILPr2JuaW5g7uQi0EcJVZJu8IYp3yfhI=",
      "dataSize": "14"
    }
  ],
  "nextVersionHandle": "3"
}
```

#### Retrieve specific versions of a model

```console
$ echo "{\"model_id\":\"my_model\", \"version_numbers\":[1]}" | grpcurl -plaintext -d @ localhost:9000 cogmentAPI.ModelRegistrySP/RetrieveVersionInfos
{
  "versionInfos": [
    {
      "modelId": "my_model",
      "versionNumber": 1,
      "creationTimestamp": "1633119005107454620",
      "archived": true,
      "dataHash": "jY0g3VkUK62ILPr2JuaW5g7uQi0EcJVZJu8IYp3yfhI=",
      "dataSize": "14"
    }
  ],
  "nextVersionHandle": "2"
}
```

#### Retrieve the n-th to last version of a model

```console
$ echo "{\"model_id\":\"my_model\", \"version_numbers\":[-2]}" | grpcurl -plaintext -d @ localhost:9000 cogmentAPI.ModelRegistrySP/RetrieveVersionInfos
{
  "versionInfos": [
    {
      "modelId": "my_model",
      "versionNumber": 1,
      "creationTimestamp": "1633119005107454620",
      "archived": true,
      "dataHash": "jY0g3VkUK62ILPr2JuaW5g7uQi0EcJVZJu8IYp3yfhI=",
      "dataSize": "14"
    }
  ],
  "nextVersionHandle": "2"
}
```

### Retrieve given version data - `cogmentAPI.ModelRegistrySP/RetrieveVersionData ( .cogmentAPI.RetrieveVersionDataRequest ) returns ( stream .cogmentAPI.RetrieveVersionDataReplyChunk );`

_This example requires `COGMENT_MODEL_REGISTRY_GRPC_REFLECTION` to be enabled and requires [grpcurl](https://github.com/fullstorydev/grpcurl)_

```console
$ echo "{\"model_id\":\"my_model\", \"version_number\":1}" | grpcurl -plaintext -d @ localhost:9000 cogment.ModelRegistrySP/RetrieveVersionData
{
  "dataChunk": "Y2h1bmtfMWNodW5rXzI="
}
```

To retrieve the n-th to last version, use `version_number:-n` (e.g. `-1` for the latest, `-2` for the 2nd to last).

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

### Release process

People having maintainers rights of the repository can follow these steps to release a version **MAJOR.MINOR.PATCH**. The versioning scheme follows [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

1. Run `./scripts/create_release_branch.sh` automatically compute and update the version of the package, create the release branch and update the changelog from the commit history,
2. On the release branch, check and update the changelog if needed
3. Update `.cogment-api.yaml` (in particular make sure it doesn't rely on the _"latest"_ version)
4. Make sure everything's fine on CI,
5. Run `./scripts/tag_release.sh MAJOR.MINOR.PATCH` to create the specific version section in the changelog, merge the release branch in `main`, create the release tag and update the `develop` branch with those.

The rest, publishing the packages to github releases and dockerhub and updating the mirror repositories, is handled directly by the CI.

```

```
