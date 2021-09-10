#!/usr/bin/env bash

# This build script is inspired by https://lipanski.com/posts/speed-up-your-docker-builds-with-cache-from
# It does 3 things
# - Retrieve a given tag for the "build" stage of the image
# - Build the "build" stage of the image using the retrieved image for cache
# - Build the final image

if [[ -z "${COGMENT_ACTIVITY_LOGGER_IMAGE_BUILD_CACHE}" ]]; then
  COGMENT_ACTIVITY_LOGGER_IMAGE_BUILD_CACHE="registry.gitlab.com/ai-r/cogment-activity-logger/build:latest"
  printf "** Using default value '%s' for the docker build image used for cache, set COGMENT_ACTIVITY_LOGGER_IMAGE_BUILD_CACHE to specify another one\n\n" "${COGMENT_ACTIVITY_LOGGER_IMAGE_BUILD_CACHE}"
fi

if [[ -z "${COGMENT_ACTIVITY_LOGGER_IMAGE_BUILD}" ]]; then
  COGMENT_ACTIVITY_LOGGER_IMAGE_BUILD="registry.gitlab.com/ai-r/cogment-activity-logger/build:local"
  printf "** Using default value '%s' for the docker build image to build, set COGMENT_ACTIVITY_LOGGER_IMAGE_BUILD to specify another one\n\n" "${COGMENT_ACTIVITY_LOGGER_IMAGE_BUILD}"
fi

if [[ -z "${COGMENT_ACTIVITY_LOGGER_IMAGE}" ]]; then
  COGMENT_ACTIVITY_LOGGER_IMAGE="cogment/model-registry:local"
  printf "** Using default value '%s' for the docker image to build, set COGMENT_ACTIVITY_LOGGER_IMAGE to specify another one\n\n" "${COGMENT_ACTIVITY_LOGGER_IMAGE}"
fi

if ! docker pull "${COGMENT_ACTIVITY_LOGGER_IMAGE_BUILD_CACHE}"; then
  printf "** Unable to pull build image for cache, skipping\n\n"
fi

set -o errexit

docker build --target build --cache-from "${COGMENT_ACTIVITY_LOGGER_IMAGE_BUILD_CACHE}" --tag "${COGMENT_ACTIVITY_LOGGER_IMAGE_BUILD}" .
docker build --cache-from "${COGMENT_ACTIVITY_LOGGER_IMAGE_BUILD}" --tag "${COGMENT_ACTIVITY_LOGGER_IMAGE}" .
