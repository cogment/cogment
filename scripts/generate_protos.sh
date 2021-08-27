#!/usr/bin/env bash

set -o errexit

MODEL_REGISTRY_DIR="$(dirname "${BASH_SOURCE[0]}")/.."
API_RELATIVE_PATH="grpcapi/cogment/api"
API_PACKAGE="github.com/cogment/model-registry/${API_RELATIVE_PATH}"

cd "${MODEL_REGISTRY_DIR}"

API_YAML_PATH=".cogment-api.yaml"

# The following will
# 1 - Retrieve uncommented lines in the yaml file
# 2 - pick the first one
# 3 - extract the key and the value
key_value="$(grep -v '^#' ${API_YAML_PATH} | head -n 1 | sed -e 's/^ *\([^ :]*\) *: *\"\{0,1\}\([^\"]*\)\"\{0,1\} *$/\1 \2/')"
IFS=" " read -r -a key_value <<<"${key_value}"

if [[ ${key_value[0]} == "cogment_api_version" ]]; then
  printf "** Downloading version %s of the cogment API\n" "${key_value[1]}"
  curl --silent -L "https://cogment.github.io/cogment-api/${key_value[1]}/cogment-api-${key_value[1]}.tar.gz" | tar xz -C "${API_RELATIVE_PATH}"
elif [[ ${key_value[0]} == "cogment_api_path" ]]; then
  printf "** Copying the cogment API from %s\n" "${key_value[1]}"
  cp -r "${key_value[1]}"/* "${API_RELATIVE_PATH}"
else
  printf "** Unable to retrieve the desired cogment API please create a .cogment-api.yaml defining either 'cogment_api_version' or 'cogment_api_path'\n"
  exit 1
fi

MODEL_REGISTRY_PROTO_RELATIVE_PATH="${API_RELATIVE_PATH}/model_registry.proto"

protoc --go_out=. --go-grpc_out=. \
  --go_opt=paths=source_relative \
  --go-grpc_opt=paths=source_relative \
  --go_opt=M"${MODEL_REGISTRY_PROTO_RELATIVE_PATH}"="${API_PACKAGE}" \
  --go-grpc_opt=M"${MODEL_REGISTRY_PROTO_RELATIVE_PATH}"="${API_PACKAGE}" \
  "${MODEL_REGISTRY_PROTO_RELATIVE_PATH}"
