#!/usr/bin/env bash

set -o errexit

ACTIVITY_LOGGER_DIR="$(dirname "${BASH_SOURCE[0]}")/.."
PROTOS_RELATIVE_PATH="grpcapi"
API_PACKAGE="github.com/cogment/cogment-trial-datastore/${PROTOS_RELATIVE_PATH}/cogment/api"

cd "${ACTIVITY_LOGGER_DIR}"

API_YAML_PATH=".cogment-api.yaml"

# The following will
# 1 - Retrieve uncommented lines in the yaml file
# 2 - pick the first one
# 3 - extract the key and the value
key_value="$(grep -v '^#' ${API_YAML_PATH} | head -n 1 | sed -e 's/^ *\([^ :]*\) *: *\"\{0,1\}\([^\"]*\)\"\{0,1\} *$/\1 \2/')"
IFS=" " read -r -a key_value <<<"${key_value}"

if [[ ${key_value[0]} == "cogment_api_version" ]]; then
  printf "** Downloading version %s of the cogment API\n" "${key_value[1]}"
  curl --silent -L "https://cogment.github.io/cogment-api/${key_value[1]}/cogment-api-${key_value[1]}.tar.gz" | tar xz -C "${PROTOS_RELATIVE_PATH}/cogment/api"
elif [[ ${key_value[0]} == "cogment_api_path" ]]; then
  printf "** Copying the cogment API from %s\n" "${key_value[1]}"
  cp -r "${key_value[1]}"/* "${PROTOS_RELATIVE_PATH}/cogment/api"
else
  printf "** Unable to retrieve the desired cogment API please create a .cogment-api.yaml defining either 'cogment_api_version' or 'cogment_api_path'\n"
  exit 1
fi

protoc --go_out=${PROTOS_RELATIVE_PATH} --go-grpc_out=${PROTOS_RELATIVE_PATH} \
  --proto_path=${PROTOS_RELATIVE_PATH} \
  --go_opt=paths=source_relative \
  --go-grpc_opt=paths=source_relative \
  --go_opt=Mcogment/api/common.proto=${API_PACKAGE} \
  --go-grpc_opt=Mcogment/api/common.proto=${API_PACKAGE} \
  --go_opt=Mcogment/api/datalog.proto=${API_PACKAGE} \
  --go-grpc_opt=Mcogment/api/datalog.proto=${API_PACKAGE} \
  --go_opt=Mcogment/api/environment.proto=${API_PACKAGE} \
  --go-grpc_opt=Mcogment/api/environment.proto=${API_PACKAGE} \
  --go_opt=Mcogment/api/trial_datastore.proto=${API_PACKAGE} \
  --go-grpc_opt=Mcogment/api/trial_datastore.proto=${API_PACKAGE} \
  cogment/api/common.proto \
  cogment/api/datalog.proto \
  cogment/api/environment.proto \
  cogment/api/trial_datastore.proto
