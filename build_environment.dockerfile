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

FROM ubuntu:18.04

ENV DEBIAN_FRONTEND=noninteractive
ENV TZ=America/New_York

RUN apt-get update && apt-get install -y \
  autoconf \
  autogen \
  build-essential \
  curl \
  git \
  libcurl4-openssl-dev \
  libgflags-dev \
  libpulse-dev \
  libssl-dev \
  libtool \
  lsb-release \
  software-properties-common \
  unzip \
  wget \
  shellcheck

# Install clang-format v10
RUN curl --silent  https://apt.llvm.org/llvm-snapshot.gpg.key|apt-key add - && \
  echo "deb http://apt.llvm.org/bionic/ llvm-toolchain-bionic main" >> /etc/apt/sources.list && \
  echo "deb-src http://apt.llvm.org/bionic/ llvm-toolchain-bionic main" >> /etc/apt/sources.list && \
  apt-get update && apt-get install -y clang-format-10 && \
  ln -s $(which clang-format-10) /usr/bin/clang-format

# Install CMake v3.22.3
RUN curl -L --silent https://github.com/Kitware/CMake/releases/download/v3.22.3/cmake-3.22.3-linux-x86_64.sh --output install-cmake.sh && \
  sh install-cmake.sh --prefix=/usr/local/ --exclude-subdir

# Install Go v1.17.10
RUN curl -L --silent https://go.dev/dl/go1.17.10.linux-amd64.tar.gz --output go-linux-amd64.tar.gz && \
  tar -C /usr/local -xzf go-linux-amd64.tar.gz

ENV GOPATH=/opt/go
ENV PATH=${PATH}:/usr/local/go/bin:${GOPATH}/bin

RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.50.1

# Install shfmt v3
RUN go install mvdan.cc/sh/v3/cmd/shfmt@latest

WORKDIR /workspace
