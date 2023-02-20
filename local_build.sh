#!/usr/bin/env bash

set -e

export COGMENT_BUILD_TYPE="Release"

export COGMENT_BUILD_ENVIRONMENT_IMAGE="local/cogment_build_environment"
export COGMENT_BUILD_IMAGE="local/cogment_build"
export COGMENT_IMAGE="local/cogment"

export COGMENT_BUILD_ENVIRONMENT_IMAGE_CACHE=${COGMENT_BUILD_ENVIRONMENT_IMAGE}
export COGMENT_BUILD_IMAGE_CACHE=${COGMENT_BUILD_IMAGE}

if [[ -z "$1" ]]; then

  ./build_docker.sh

elif [[ "$1" == "lint" ]]; then
  # Lint files that were part of the build (above).
  # I.e. Only the version of the files that were part
  #      of the normal build will be "linted".

  printf "Lint...\n"
  docker run --rm \
    --volume "$(pwd)/build":/workspace/build \
    ${COGMENT_BUILD_IMAGE} \
    ./build_linux.sh lint

elif [[ "$1" == "style" ]]; then
  # Update and check style of files in folder.

  printf "Clang Format all C++ Orchestrator files...\n"
  pushd packages/orchestrator >>/dev/null
  docker run --rm \
    --user "$(id -u)":"$(id -g)" \
    --volume "$(pwd)":/workspace \
    ${COGMENT_BUILD_ENVIRONMENT_IMAGE} \
    find . \( -name "*.h" -o -name "*.cpp" \) \
    -exec clang-format --style=file -i "{}" \;
  popd >>/dev/null

  printf "Shellcheck...\n"
  docker run --rm \
    --user "$(id -u)":"$(id -g)" \
    --volume "$(pwd)":/workspace \
    ${COGMENT_BUILD_ENVIRONMENT_IMAGE} \
    find . \( -name "*.sh" -not -path "./build/*" \) \
    -exec shellcheck "{}" \;

  printf "Shfmt...\n"
  docker run --rm \
    --user "$(id -u)":"$(id -g)" \
    --volume "$(pwd)":/workspace \
    ${COGMENT_BUILD_ENVIRONMENT_IMAGE} \
    find . \( -name "*.sh" -not -path "./build/*" \) \
    -exec shfmt -i 2 -ci -d "{}" \;

elif [[ "$1" == "go_proto" ]]; then
  GO_PROTOC_IMAGE="local/go_protoc"
  LOCAL_DEST="build/go"

  docker build -t ${GO_PROTOC_IMAGE} -f go_protoc.dockerfile .

  rm -rdf ${LOCAL_DEST}
  mkdir ${LOCAL_DEST}
  docker run --rm \
    --user "$(id -u)":"$(id -g)" \
    --volume "$(pwd)/packages/grpc_api":/in \
    --volume "$(pwd)/${LOCAL_DEST}":/out \
    ${GO_PROTOC_IMAGE} \
    --go_out=/out \
    --go_opt=paths=source_relative \
    --go-grpc_out=/out \
    --go-grpc_opt=paths=source_relative \
    --proto_path=/in \
    cogment/api/agent.proto \
    cogment/api/common.proto \
    cogment/api/datalog.proto \
    cogment/api/directory.proto \
    cogment/api/environment.proto \
    cogment/api/health.proto \
    cogment/api/hooks.proto \
    cogment/api/model_registry.proto \
    cogment/api/orchestrator.proto

  printf "Generated gRPC and protobuf Go files: %s\n" ./${LOCAL_DEST}

fi
