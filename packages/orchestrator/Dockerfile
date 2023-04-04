# Copyright 2023 AI Redefined Inc. <dev+cogment@ai-r.com>
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

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

