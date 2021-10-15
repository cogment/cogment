#!/usr/bin/env bash

# This build script is inspired by https://lipanski.com/posts/speed-up-your-docker-builds-with-cache-from
# It does 3 things
# - Retrieve a given tag for the "build" stage of the image
# - Build the "build" stage of the image using the retrieved image for cache
# - Build the final image

if [[ -z "${COGMENT_TRIAL_DATASTORE_IMAGE_BUILD_CACHE}" ]]; then
  COGMENT_TRIAL_DATASTORE_IMAGE_BUILD_CACHE="registry.gitlab.com/ai-r/cogment-trial-datastore/build:latest"
  printf "** Using default value '%s' for the docker build image used for cache, set COGMENT_TRIAL_DATASTORE_IMAGE_BUILD_CACHE to specify another one\n\n" "${COGMENT_TRIAL_DATASTORE_IMAGE_BUILD_CACHE}"
fi

if [[ -z "${COGMENT_TRIAL_DATASTORE_IMAGE_BUILD}" ]]; then
  COGMENT_TRIAL_DATASTORE_IMAGE_BUILD="registry.gitlab.com/ai-r/cogment-trial-datastore/build:local"
  printf "** Using default value '%s' for the docker build image to build, set COGMENT_TRIAL_DATASTORE_IMAGE_BUILD to specify another one\n\n" "${COGMENT_TRIAL_DATASTORE_IMAGE_BUILD}"
fi

if [[ -z "${COGMENT_TRIAL_DATASTORE_IMAGE}" ]]; then
  COGMENT_TRIAL_DATASTORE_IMAGE="cogment/trial-datastore:local"
  printf "** Using default value '%s' for the docker image to build, set COGMENT_TRIAL_DATASTORE_IMAGE to specify another one\n\n" "${COGMENT_TRIAL_DATASTORE_IMAGE}"
fi

if ! docker pull "${COGMENT_TRIAL_DATASTORE_IMAGE_BUILD_CACHE}"; then
  printf "** Unable to pull build image for cache, skipping\n\n"
fi

set -o errexit

docker build --target build --cache-from "${COGMENT_TRIAL_DATASTORE_IMAGE_BUILD_CACHE}" --tag "${COGMENT_TRIAL_DATASTORE_IMAGE_BUILD}" .
docker build --cache-from "${COGMENT_TRIAL_DATASTORE_IMAGE_BUILD}" --tag "${COGMENT_TRIAL_DATASTORE_IMAGE}" .
