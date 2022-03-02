#!/usr/bin/env bash

set -o errexit

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
BUILD_DIR="${ROOT_DIR}/build/linux_amd64_no_orchestrator"
INSTALL_DIR="${ROOT_DIR}/install/linux_amd64_no_orchestrator"

mkdir -p "${BUILD_DIR}"
mkdir -p "${INSTALL_DIR}"

pushd "${BUILD_DIR}"
cmake \
  -DCMAKE_BUILD_TYPE="Release" \
  -DCMAKE_INSTALL_PREFIX="${INSTALL_DIR}" \
  -DCOGMENT_EMBEDS_ORCHESTRATOR=OFF \
  -DCOGMENT_OS=linux \
  -DCOGMENT_ARCH=amd64 \
  "${ROOT_DIR}"
make
make install
popd

BUILD_DIR="${ROOT_DIR}/build/windows_amd64_no_orchestrator"
INSTALL_DIR="${ROOT_DIR}/install/windows_amd64_no_orchestrator"

mkdir -p "${BUILD_DIR}"
mkdir -p "${INSTALL_DIR}"

pushd "${BUILD_DIR}"
cmake \
  -DCMAKE_BUILD_TYPE="Release" \
  -DCMAKE_INSTALL_PREFIX="${INSTALL_DIR}" \
  -DCOGMENT_EMBEDS_ORCHESTRATOR=OFF \
  -DCOGMENT_OS=windows \
  -DCOGMENT_ARCH=amd64 \
  "${ROOT_DIR}"
make
make install
popd

BUILD_DIR="${ROOT_DIR}/build/macos_amd64_no_orchestrator"
INSTALL_DIR="${ROOT_DIR}/install/macos_amd64_no_orchestrator"

mkdir -p "${BUILD_DIR}"
mkdir -p "${INSTALL_DIR}"

pushd "${BUILD_DIR}"
cmake \
  -DCMAKE_BUILD_TYPE="Release" \
  -DCMAKE_INSTALL_PREFIX="${INSTALL_DIR}" \
  -DCOGMENT_EMBEDS_ORCHESTRATOR=OFF \
  -DCOGMENT_OS=darwin \
  -DCOGMENT_ARCH=amd64 \
  "${ROOT_DIR}"
make
make install
popd

BUILD_DIR="${ROOT_DIR}/build/macos_arm64_no_orchestrator"
INSTALL_DIR="${ROOT_DIR}/install/macos_arm64_no_orchestrator"

mkdir -p "${BUILD_DIR}"
mkdir -p "${INSTALL_DIR}"

pushd "${BUILD_DIR}"
cmake \
  -DCMAKE_BUILD_TYPE="Release" \
  -DCMAKE_INSTALL_PREFIX="${INSTALL_DIR}" \
  -DCOGMENT_EMBEDS_ORCHESTRATOR=OFF \
  -DCOGMENT_OS=darwin \
  -DCOGMENT_ARCH=arm64 \
  "${ROOT_DIR}"
make
make install
popd
