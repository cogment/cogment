#!/usr/bin/env bash

if [[ -z "${COGMENT_IMAGE}" ]]; then
  COGMENT_IMAGE="registry.gitlab.com/ai-r/cogment-cli:local"
  printf "** Using default value '%s' for the cogment docker image, set COGMENT_IMAGE to specify another one\n" "${COGMENT_IMAGE}"
fi

if [[ -z "${COGMENT_CLI_IMAGE}" ]]; then
  COGMENT_CLI_IMAGE="registry.gitlab.com/ai-r/cogment-cli/cli:local"
  printf "** Using default value '%s' for the cli docker image to build, set COGMENT_CLI_IMAGE to specify another one\n" "${COGMENT_CLI_IMAGE}"
fi

if [[ -z "${COGMENT_ORCHESTRATOR_IMAGE}" ]]; then
  COGMENT_ORCHESTRATOR_IMAGE="registry.gitlab.com/ai-r/cogment-cli/orchestrator:local"
  printf "** Using default value '%s' for the orchestrator docker image to build, set COGMENT_ORCHESTRATOR_IMAGE to specify another one\n" "${COGMENT_ORCHESTRATOR_IMAGE}"
fi

if [[ -z "${COGMENT_TRIAL_DATASTORE_IMAGE}" ]]; then
  COGMENT_TRIAL_DATASTORE_IMAGE="registry.gitlab.com/ai-r/cogment-cli/trial-datastore:local"
  printf "** Using default value '%s' for the trial datastore docker image to build, set COGMENT_TRIAL_DATASTORE_IMAGE to specify another one\n" "${COGMENT_TRIAL_DATASTORE_IMAGE}"
fi

if [[ -z "${COGMENT_MODEL_REGISTRY_IMAGE}" ]]; then
  COGMENT_MODEL_REGISTRY_IMAGE="registry.gitlab.com/ai-r/cogment-cli/model-registry:local"
  printf "** Using default value '%s' for the model registry docker image to build, set COGMENT_MODEL_REGISTRY_IMAGE to specify another one\n" "${COGMENT_MODEL_REGISTRY_IMAGE}"
fi

set -o errexit

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# Build the cli image
docker build \
  --build-arg "COGMENT_IMAGE=${COGMENT_IMAGE}" \
  --tag "${COGMENT_CLI_IMAGE}" \
  --file "${SCRIPT_DIR}/cli.dockerfile" \
  .

# Build the orchestrator image
docker build \
  --build-arg "COGMENT_IMAGE=${COGMENT_IMAGE}" \
  --tag "${COGMENT_ORCHESTRATOR_IMAGE}" \
  --file "${SCRIPT_DIR}/orchestrator.dockerfile" \
  .

# Build the trial datastore image
docker build \
  --build-arg "COGMENT_IMAGE=${COGMENT_IMAGE}" \
  --tag "${COGMENT_TRIAL_DATASTORE_IMAGE}" \
  --file "${SCRIPT_DIR}/trial_datastore.dockerfile" \
  .

# Build the model registry image
docker build \
  --build-arg "COGMENT_IMAGE=${COGMENT_IMAGE}" \
  --tag "${COGMENT_MODEL_REGISTRY_IMAGE}" \
  --file "${SCRIPT_DIR}/model_registry.dockerfile" \
  .
