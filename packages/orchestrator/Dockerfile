ARG ORCHESTRATOR_DEPENDENCIES_IMAGE
FROM ${ORCHESTRATOR_DEPENDENCIES_IMAGE} AS orchestrator_dependencies

FROM ubuntu:20.04

ENV DEBIAN_FRONTEND=noninteractive
ENV TZ=America/New_York

RUN apt-get update && apt-get install -y \
  build-essential \
  cmake \
  git \
  clang \
  clang-format \
  autogen \
  autoconf \
  libtool \
  libgflags-dev \
  wget \
  libcurl4-openssl-dev \
  libssl-dev \
  uuid-dev \
  libpulse-dev

ENV INSTALL_DIR="/workspace/install/linux_amd64"

WORKDIR /workspace/packages/orchestrator_dependencies
COPY --from=orchestrator_dependencies ${INSTALL_DIR} ${INSTALL_DIR}

WORKDIR /workspace/packages/orchestrator
COPY . .
RUN ./build.sh

