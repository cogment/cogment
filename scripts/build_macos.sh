#!/usr/bin/env bash

set -o errexit

if [[ -z "${BUILD_TYPE}" ]]; then
  BUILD_TYPE=Release
fi

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
INSTALL_DIR="${ROOT_DIR}/install"

if [[ -z "${THIRD_PARTY_INSTALL_DIR}" ]]; then
  THIRD_PARTY_INSTALL_DIR=${INSTALL_DIR}
fi
mkdir -p "${THIRD_PARTY_INSTALL_DIR}"
THIRD_PARTY_INSTALL_DIR=$(cd "${THIRD_PARTY_INSTALL_DIR}" && pwd)

if [[ -z "${SKIP_BUILD_THIRD_PARTY}" ]]; then
  #################
  ## third party ##
  #################
  if [[ -z "${THIRD_PARTY_CLONE_DIR}" ]]; then
    THIRD_PARTY_CLONE_DIR=./third_party/
  fi
  mkdir -p "${THIRD_PARTY_CLONE_DIR}"
  THIRD_PARTY_CLONE_DIR=$(cd "${THIRD_PARTY_CLONE_DIR}" && pwd)

  MACOS_DEPLOYMENT_TARGET=10.15
  GRPC_RELEASE_TAG=v1.41.1
  PROMETHEUS_CPP_RELEASE_TAG=v0.6.0
  SPDLOG_RELEASE_TAG=v1.x
  YAML_CPP_RELEASE_TAG=yaml-cpp-0.6.2

  ##########
  ## grpc ##
  ##########
  GRPC_CLONE_DIR="${THIRD_PARTY_CLONE_DIR}/grpc"
  printf "**** Retrieving sources for grpc %s in %s.\n\n" "${GRPC_RELEASE_TAG}" "${GRPC_CLONE_DIR}"
  if [ -d "${GRPC_CLONE_DIR}" ]; then
    # If the directory already exist we trust it's actually a clone of grpc and simply checkout the target tag
    cd "${GRPC_CLONE_DIR}"
    git fetch --tags --depth 1
    git switch --detach "${GRPC_RELEASE_TAG}"
  else
    git clone -b "${GRPC_RELEASE_TAG}" --depth 1 --recurse-submodules --shallow-submodules https://github.com/grpc/grpc "${GRPC_CLONE_DIR}"
    cd "${GRPC_CLONE_DIR}"
  fi

  mkdir -p cmake/build
  cd cmake/build
  rm -rf ./*
  cmake \
    -DCMAKE_BUILD_TYPE="${BUILD_TYPE}" \
    -DCMAKE_OSX_DEPLOYMENT_TARGET="${MACOS_DEPLOYMENT_TARGET}" \
    -DgRPC_INSTALL=ON \
    -DgRPC_BUILD_TESTS=OFF \
    -DCMAKE_INSTALL_PREFIX="${THIRD_PARTY_INSTALL_DIR}" \
    ../..
  make -j
  make install

  ####################
  ## prometheus-cpp ##
  ####################
  PROMETHEUS_CPP_CLONE_DIR="${THIRD_PARTY_CLONE_DIR}/prometheus-cpp"
  printf "**** Retrieving sources prometheus-cpp %s in %s.\n\n" "${PROMETHEUS_CPP_RELEASE_TAG}" "${PROMETHEUS_CPP_CLONE_DIR}"
  if [ -d "${PROMETHEUS_CPP_CLONE_DIR}" ]; then
    # If the directory already exist we trust it's actually a clone of prometheus-cpp and simply checkout the target tag
    cd "${PROMETHEUS_CPP_CLONE_DIR}"
    git fetch --tags --depth 1
    git switch --detach "${PROMETHEUS_CPP_RELEASE_TAG}"
  else
    git clone -b "${PROMETHEUS_CPP_RELEASE_TAG}" --depth 1 --recurse-submodules --shallow-submodules https://github.com/jupp0r/prometheus-cpp.git "${PROMETHEUS_CPP_CLONE_DIR}"
    cd "${PROMETHEUS_CPP_CLONE_DIR}"
  fi

  mkdir -p build
  cd build
  rm -rf ./*
  cmake \
    -DCMAKE_BUILD_TYPE="${BUILD_TYPE}" \
    -DCMAKE_INSTALL_PREFIX="${THIRD_PARTY_INSTALL_DIR}" \
    -DCMAKE_OSX_DEPLOYMENT_TARGET="${MACOS_DEPLOYMENT_TARGET}" ..
  make -j
  make install

  ############
  ## spdlog ##
  ############
  SPDLOG_CLONE_DIR="${THIRD_PARTY_CLONE_DIR}/spdlog"
  printf "**** Retrieving sources spdlog %s in %s.\n\n" "${SPDLOG_RELEASE_TAG}" "${SPDLOG_CLONE_DIR}"
  if [ -d "${SPDLOG_CLONE_DIR}" ]; then
    # If the directory already exist we trust it's actually a clone of spdlog and simply checkout the target tag
    cd "${SPDLOG_CLONE_DIR}"
    git fetch --tags --depth 1
    git switch --detach "${SPDLOG_RELEASE_TAG}"
  else
    git clone -b "${SPDLOG_RELEASE_TAG}" --depth 1 --recurse-submodules --shallow-submodules https://github.com/gabime/spdlog.git "${SPDLOG_CLONE_DIR}"
    cd "${SPDLOG_CLONE_DIR}"
  fi

  mkdir -p build
  cd build
  rm -rf ./*
  cmake \
    -DCMAKE_BUILD_TYPE="${BUILD_TYPE}" \
    -DCMAKE_INSTALL_PREFIX="${THIRD_PARTY_INSTALL_DIR}" \
    -DCMAKE_OSX_DEPLOYMENT_TARGET="${MACOS_DEPLOYMENT_TARGET}" \
    -DSPDLOG_BUILD_TESTS=OFF \
    ..
  make -j
  make install

  ##############
  ## yaml-cpp ##
  ##############
  YAML_CPP_CLONE_DIR="${THIRD_PARTY_CLONE_DIR}/yaml-cpp"
  printf "**** Retrieving sources yaml-cpp %s in %s.\n\n" "${YAML_CPP_RELEASE_TAG}" "${YAML_CPP_CLONE_DIR}"
  if [ -d "${YAML_CPP_CLONE_DIR}" ]; then
    # If the directory already exist we trust it's actually a clone of yaml-cpp and simply checkout the target tag
    cd "${YAML_CPP_CLONE_DIR}"
    git fetch --tags --depth 1
    git switch --detach "${YAML_CPP_RELEASE_TAG}"
  else
    git clone -b "${YAML_CPP_RELEASE_TAG}" --depth 1 --recurse-submodules --shallow-submodules https://github.com/jbeder/yaml-cpp.git "${YAML_CPP_CLONE_DIR}"
    cd "${YAML_CPP_CLONE_DIR}"
  fi

  mkdir -p build
  cd build
  rm -rf ./*
  cmake \
    -DCMAKE_BUILD_TYPE="${BUILD_TYPE}" \
    -DCMAKE_INSTALL_PREFIX="${THIRD_PARTY_INSTALL_DIR}" \
    -DCMAKE_OSX_DEPLOYMENT_TARGET="${MACOS_DEPLOYMENT_TARGET}" \
    ..
  make -j
  make install
fi

##########################
## cogment orchestrator ##
##########################

cd "${ROOT_DIR}"
mkdir -p build
cd build
rm -rf ./*
cmake \
  -DCMAKE_BUILD_TYPE="${BUILD_TYPE}" \
  -DgRPC_ROOT="${THIRD_PARTY_INSTALL_DIR}/lib/cmake/grpc" \
  -DProtobuf_ROOT="${THIRD_PARTY_INSTALL_DIR}/lib/cmake/protobuf" \
  -Dabsl_ROOT="${THIRD_PARTY_INSTALL_DIR}/lib/cmake/absl" \
  -Dyaml-cpp_ROOT="${THIRD_PARTY_INSTALL_DIR}/lib/cmake/yaml-cpp" \
  -Dspdlog_ROOT="${THIRD_PARTY_INSTALL_DIR}/lib/cmake/spdlog" \
  -Dprometheus-cpp_ROOT="${THIRD_PARTY_INSTALL_DIR}/lib/cmake/prometheus-cpp/" \
  -DCMAKE_INSTALL_PREFIX="${INSTALL_DIR}" \
  ..
make -j
make install
