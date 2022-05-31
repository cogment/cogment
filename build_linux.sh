#!/usr/bin/env bash

set -o errexit

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
if [[ -z "${BUILD_DIR}" ]]; then
  BUILD_DIR="${ROOT_DIR}/build/linux_amd64"
fi
mkdir -p "${BUILD_DIR}"
BUILD_DIR=$(cd "${BUILD_DIR}" && pwd)

if [[ -z "${INSTALL_DIR}" ]]; then
  INSTALL_DIR="${ROOT_DIR}/install/linux_amd64"
fi
mkdir -p "${INSTALL_DIR}"
INSTALL_DIR=$(cd "${INSTALL_DIR}" && pwd)

if [[ -z "${COGMENT_BUILD_TYPE}" ]]; then
  COGMENT_BUILD_TYPE="Release"
fi

pushd "${BUILD_DIR}"

cmake \
  -DCMAKE_BUILD_TYPE="${COGMENT_BUILD_TYPE}" \
  -DCMAKE_INSTALL_PREFIX="${INSTALL_DIR}" \
  -DCOGMENT_EMBEDS_ORCHESTRATOR=ON \
  -DCOGMENT_OS=linux \
  -DCOGMENT_ARCH=amd64 \
  "${ROOT_DIR}"
make -j"$(nproc)"

make install

popd
