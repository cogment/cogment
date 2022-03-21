#!/usr/bin/env bash

if [[ -z "${COGMENT_BUILD_ENVIRONMENT_IMAGE_CACHE}" ]]; then
  COGMENT_BUILD_ENVIRONMENT_IMAGE_CACHE="registry.gitlab.com/ai-r/cogment/build_environment:latest"
  printf "** Using default value '%s' for the docker build environment image used for cache, set COGMENT_BUILD_ENVIRONMENT_IMAGE_CACHE to specify another one\n" "${COGMENT_BUILD_ENVIRONMENT_IMAGE_CACHE}"
fi

if ! docker pull "${COGMENT_BUILD_ENVIRONMENT_IMAGE_CACHE}"; then
  printf "*** Unable to pull the build image for cache, skipping\n"
fi

if [[ -z "${COGMENT_BUILD_ENVIRONMENT_IMAGE}" ]]; then
  COGMENT_BUILD_ENVIRONMENT_IMAGE="registry.gitlab.com/ai-r/cogment/build_environment:local"
  printf "** Using default value '%s' for the docker build environment image to build, set COGMENT_BUILD_ENVIRONMENT_IMAGE to specify another one\n" "${COGMENT_BUILD_ENVIRONMENT_IMAGE}"
fi

if [[ -z "${COGMENT_BUILD_IMAGE_CACHE}" ]]; then
  COGMENT_BUILD_IMAGE_CACHE="registry.gitlab.com/ai-r/cogment/build:latest"
  printf "** Using default value '%s' for the docker build image used for cache, set COGMENT_BUILD_IMAGE_CACHE to specify another one\n" "${COGMENT_BUILD_IMAGE_CACHE}"
fi

if ! docker pull "${COGMENT_BUILD_IMAGE_CACHE}"; then
  printf "*** Unable to pull the build image for cache, skipping\n"
fi

if [[ -z "${COGMENT_BUILD_IMAGE}" ]]; then
  COGMENT_BUILD_IMAGE="registry.gitlab.com/ai-r/cogment/build:local"
  printf "** Using default value '%s' for the docker build image to build, set COGMENT_BUILD_IMAGE to specify another one\n" "${COGMENT_BUILD_IMAGE}"
fi

if [[ -z "${COGMENT_IMAGE}" ]]; then
  COGMENT_IMAGE="registry.gitlab.com/ai-r/cogment:local"
  printf "** Using default value '%s' for the docker image to build, set COGMENT_IMAGE to specify another one\n" "${COGMENT_IMAGE}"
fi

set -o errexit

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
BUILD_DIR="${ROOT_DIR}/build/linux_amd64"
INSTALL_DIR="${ROOT_DIR}/install/linux_amd64"

mkdir -p "${BUILD_DIR}"
mkdir -p "${INSTALL_DIR}"

cd "${ROOT_DIR}"

# Build the build environment image
docker build \
  --cache-from "${COGMENT_BUILD_ENVIRONMENT_IMAGE_CACHE}" \
  --cache-from "${COGMENT_BUILD_ENVIRONMENT_IMAGE}" \
  --tag "${COGMENT_BUILD_ENVIRONMENT_IMAGE}" \
  --file build_environment.dockerfile \
  .

# Build the build image
docker build \
  --build-arg "COGMENT_BUILD_ENVIRONMENT_IMAGE=${COGMENT_BUILD_ENVIRONMENT_IMAGE}" \
  --cache-from "${COGMENT_BUILD_IMAGE_CACHE}" \
  --cache-from "${COGMENT_BUILD_IMAGE}" \
  --tag "${COGMENT_BUILD_IMAGE}" \
  --file build.dockerfile \
  .

# Run the build
docker run \
  --rm \
  --volume="${BUILD_DIR}":/workspace/build/linux_amd64 \
  --volume="${INSTALL_DIR}":/workspace/install/linux_amd64 \
  "${COGMENT_BUILD_IMAGE}"

# Build the cogment image
docker build \
  --build-arg "COGMENT_EXEC=${INSTALL_DIR}/bin/cogment" \
  --tag "${COGMENT_IMAGE}" \
  --file cogment.dockerfile \
  .