# Cogment Model Registry

[![Latest Release](https://img.shields.io/docker/v/cogment/model-registry?label=docker%20release&sort=semver&style=flat-square)](https://hub.docker.com/r/cogment/model-registry) [![Apache 2 License](https://img.shields.io/badge/license-Apache%202-green?style=flat-square)](./LICENSE) [![Changelog](https://img.shields.io/badge/-Changelog%20-blueviolet?style=flat-square)](./CHANGELOG.md)

[Cogment](https://cogment.ai) is an innovative open source AI platform designed to leverage the advent of AI to benefit humankind through human-AI collaboration developed by [AI Redefined](https://ai-r.com). Cogment enables AI researchers and engineers to build, train and operate AI agents in simulated or real environments shared with humans. For the full user documentation visit <https://docs.cogment.ai>

This module, cogment Model Registry, is a versioned key value store dedicated to the storage of models.

## Usage

```console
$ docker run -p 9000:9000 -v $(pwd)/relative/path/to/model/archive:/data cogment/model-registry
```

## API

### Welcome - `GET /`

```console
$ curl -s http://localhost:80 | jq .
{
  "message": "Welcome to the model registry API"
}
```

### Create a model - `POST /models`

```console
$ curl -X POST -d '{"modelId":"my_model"}' -H "Content-Type: application/json" -s http://localhost:80/models | jq .
{
  "modelId": "my_model",
  "createdAt": "2021-04-29T09:43:33.71684-04:00",
  "latestVersionCreatedAt": "0001-01-01T00:00:00Z"
}
```

### Create a model version - `POST /models/{model_id}`

```console
curl -X POST -d 'this_is_a_predictive_model' -H "Content-Type: application/octet-stream" -s http://localhost:80/models/my_model | jq .
{
  "modelId": "my_model",
  "createdAt": "2021-04-29T09:46:23.832131-04:00",
  "number": 1,
  "hash": "PKEpqIpxMLOG1enOvkMIqOIGvgI4qglUZ35I1+Ij8m8="
}
```

### List models - `GET /models`

```console
$ curl -s http://localhost:80/models | jq .
[
  {
    "modelId": "my_model",
    "createdAt": "2021-04-29T09:43:33.71684-04:00",
    "latestVersionCreatedAt": "2021-04-29T09:46:23.832131-04:00",
    "latestVersionNumber": 1
  }
]
```

### List model versions - `GET /models/{model_id}`

```console
$ curl -s http://localhost:80/models/my_model | jq .
[
  {
    "modelId": "my_model",
    "createdAt": "2021-04-29T09:46:23.832131-04:00",
    "number": 1,
    "hash": "PKEpqIpxMLOG1enOvkMIqOIGvgI4qglUZ35I1+Ij8m8="
  },
  {
    "modelId": "my_model",
    "createdAt": "2021-04-29T09:48:19.783984-04:00",
    "number": 2,
    "hash": "nqiSuZ305oBkbP1EZI5j+BGD3tFCsiHoMghm+7s3S7Y="
  }
]
```

### Retrieve latest version - `GET /models/{model_id}/latest`

```console
$ curl -s http://localhost:80/models/my_model/latest
this_is_another_predictive_model%
```

### Retrieve given version - `GET /models/{model_id}/{version_number}`

```console
$ curl -s http://localhost:80/models/my_model/1
this_is_a_predictive_model%
```

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
