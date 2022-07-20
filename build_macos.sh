#!/usr/bin/env bash

set -o errexit

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

if [ "$(uname)" != "Darwin" ]; then
  printf "This script is designed for macos (%s detected).\n" "$(uname)"
  exit 1
fi

case $(uname -m) in
  "x86_64" | "amd64")
    native_arch="amd64"
    macos_native_arch="x86_64"
    cross_arch="arm64"
    macos_cross_arch="arm64"
    ;;
  "arm64")
    native_arch="arm64"
    macos_native_arch="arm64"
    cross_arch="amd64"
    macos_cross_arch="x86_64"
    ;;
  *)
    printf "%s: unsupported system architecture.\n" "${native_arch}"
    exit 1
    ;;
esac

NATIVE_BUILD_DIR="${ROOT_DIR}/build/macos_${native_arch}"
NATIVE_INSTALL_DIR="${ROOT_DIR}/install/macos_${native_arch}"

CROSS_BUILD_DIR="${ROOT_DIR}/build/macos_${cross_arch}"
CROSS_INSTALL_DIR="${ROOT_DIR}/install/macos_${cross_arch}"

function native_cmake_generate() {
  mkdir -p "${NATIVE_BUILD_DIR}"
  pushd "${NATIVE_BUILD_DIR}"
  cmake \
    -DCMAKE_BUILD_TYPE="Release" \
    -DCMAKE_INSTALL_PREFIX="${NATIVE_INSTALL_DIR}" \
    -DCMAKE_OSX_ARCHITECTURES=${macos_native_arch} \
    -DCOGMENT_EMBEDS_ORCHESTRATOR=ON \
    -DCOGMENT_OS=darwin \
    -DCOGMENT_ARCH="${native_arch}" \
    -DINSTALL_PROTOC=ON \
    "${ROOT_DIR}"
  popd
}

function cross_cmake_generate() {
  mkdir -p "${CROSS_BUILD_DIR}"
  pushd "${CROSS_BUILD_DIR}"
  PATH="${NATIVE_INSTALL_DIR}/bin:${PATH}"
  cmake \
    -DCMAKE_BUILD_TYPE="Release" \
    -DCMAKE_INSTALL_PREFIX="${CROSS_INSTALL_DIR}" \
    -DCMAKE_OSX_ARCHITECTURES=${macos_cross_arch} \
    -DCOGMENT_EMBEDS_ORCHESTRATOR=ON \
    -DCOGMENT_OS=darwin \
    -DCOGMENT_ARCH="${cross_arch}" \
    -DPROTOBUF_PROTOC_EXECUTABLE="${NATIVE_INSTALL_DIR}/bin/protoc" \
    -DGRPC_CPP_PLUGIN="${NATIVE_INSTALL_DIR}/bin/grpc_cpp_plugin" \
    "${ROOT_DIR}"
  popd
}

case $1 in
  "build" | "")
    native_cmake_generate
    pushd "${NATIVE_BUILD_DIR}"
    make -j"$(sysctl -n hw.ncpu)"
    mkdir -p "${NATIVE_INSTALL_DIR}"
    make install
    popd
    cross_cmake_generate
    pushd "${CROSS_BUILD_DIR}"
    make -j"$(sysctl -n hw.ncpu)"
    mkdir -p "${CROSS_INSTALL_DIR}"
    make install
    popd
    ;;
  "lint")
    native_cmake_generate
    pushd "${NATIVE_BUILD_DIR}"
    make cli_lint orchestrator_lint
    popd
    ;;
  "test")
    native_cmake_generate
    pushd "${NATIVE_BUILD_DIR}"
    make cli_test
    popd
    ;;
  "test_ci")
    native_cmake_generate
    pushd "${NATIVE_BUILD_DIR}"
    make cli_test_with_junit_report
    popd
    ;;
esac
