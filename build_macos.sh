#!/usr/bin/env bash

set -o errexit

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

BUILD_DIR="${ROOT_DIR}/build/macos_amd64"
mkdir -p "${BUILD_DIR}"

INSTALL_DIR="${ROOT_DIR}/install/macos_amd64"
mkdir -p "${INSTALL_DIR}"

pushd "${BUILD_DIR}"
cmake \
  -DCMAKE_BUILD_TYPE="Release" \
  -DCMAKE_INSTALL_PREFIX="${INSTALL_DIR}" \
  -DCOGMENT_EMBEDS_ORCHESTRATOR=ON \
  -DCOGMENT_OS=darwin \
  -DCOGMENT_ARCH=amd64 \
  "${ROOT_DIR}"

case $1 in
  "build" | "")
    make -j"$(sysctl -n hw.ncpu)"
    make install
    ;;
  "lint")
    make cli_lint orchestrator_lint
    ;;
  "test")
    make cli_test
    ;;
  "test_ci")
    make cli_test_with_junit_report
    ;;
esac

popd
