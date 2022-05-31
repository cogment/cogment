#!/usr/bin/env bash

set -e

export COGMENT_BUILD_TYPE="Debug"

export COGMENT_BUILD_ENVIRONMENT_IMAGE="local/cogment_build_environment"
export COGMENT_BUILD_IMAGE="local/cogment_build_debug"
export COGMENT_IMAGE="local/cogment"

export COGMENT_BUILD_ENVIRONMENT_IMAGE_CACHE=${COGMENT_BUILD_ENVIRONMENT_IMAGE}
export COGMENT_BUILD_IMAGE_CACHE=${COGMENT_BUILD_IMAGE}

if [[ -z "$1" ]]; then

  ./build_docker.sh

elif [[ "$1" == "lint" ]]; then

  echo "Clang Format all C++ Orchestrator files..."
  pushd packages/orchestrator >>/dev/null
  docker run --rm \
    --user "$(id -u)":"$(id -g)" \
    --volume "$(pwd)":/workspace \
    ${COGMENT_BUILD_ENVIRONMENT_IMAGE} \
    find . \( -name "*.h" -o -name "*.cpp" \) \
    -exec clang-format --style=file -i "{}" \;
  popd >>/dev/null

  echo "Shellcheck..."
  docker run --rm \
    --user "$(id -u)":"$(id -g)" \
    --volume "$(pwd)":/workspace \
    ${COGMENT_BUILD_ENVIRONMENT_IMAGE} \
    find . \( -name "*.sh" -not -path "./build/*" \) \
    -exec shellcheck "{}" \;

  echo "Shfmt..."
  docker run --rm \
    --user "$(id -u)":"$(id -g)" \
    --volume "$(pwd)":/workspace \
    ${COGMENT_BUILD_ENVIRONMENT_IMAGE} \
    find . \( -name "*.sh" -not -path "./build/*" \) \
    -exec shfmt -i 2 -ci -d "{}" \;

fi
