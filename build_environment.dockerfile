FROM ubuntu:20.04

ENV DEBIAN_FRONTEND=noninteractive
ENV TZ=America/New_York

RUN apt-get update && apt-get install -y \
  autoconf \
  autogen \
  build-essential \
  clang \
  clang-format \
  curl \
  git \
  libcurl4-openssl-dev \
  libgflags-dev \
  libpulse-dev \
  libssl-dev \
  libtool \
  unzip

RUN curl -L --silent https://github.com/Kitware/CMake/releases/download/v3.22.3/cmake-3.22.3-linux-x86_64.sh --output install-cmake.sh && \
  sh install-cmake.sh --prefix=/usr/local/ --exclude-subdir

RUN curl -L --silent https://go.dev/dl/go1.16.15.linux-amd64.tar.gz --output go-linux-amd64.tar.gz && \
  tar -C /usr/local -xzf go-linux-amd64.tar.gz

ENV GOPATH=/root/go
ENV PATH=${PATH}:/usr/local/go/bin:${GOPATH}/bin

RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.42.0

WORKDIR /workspace
