FROM ubuntu:19.10

LABEL org.label-schema.name="cogment-builder" \
      org.label-schema.vendor="Age of Minds" \
      org.label-schema.url="https://registry.gitlab.com/cogment/cogment:builder" \
      org.label-schema.vcs-url="https://gitlab.com/cogment/cogment"

# version tags
ARG GRPC_RELEASE_TAG=v1.24.0
ARG GRPC_WEB_RELEASE=https://github.com/grpc/grpc-web/releases/download/1.0.4/protoc-gen-grpc-web-1.0.4-linux-x86_64

RUN apt-get update && apt-get install -y \
  npm build-essential make git clang autogen autoconf libtool wget libgflags-dev clang-format

RUN git clone -b ${GRPC_RELEASE_TAG} --single-branch https://github.com/grpc/grpc /var/local/src/grpc && \
    cd /var/local/src/grpc && \
    git submodule update --init --recursive && \
    echo "--- installing protobuf ---" && \
    cd /var/local/src/grpc/third_party/protobuf && \
    ./autogen.sh && ./configure --enable-shared --disable-Werror && \
    make -j$(nproc) && make -j$(nproc) check && make install && make clean && ldconfig / && \
    echo "--- installing grpc ---" && \
    cd /var/local/src/grpc && \
    make CXXFLAGS="-w" CFLAGS="-w" -j$(nproc) && \
    make CXXFLAGS="-w" CFLAGS="-w" -j$(nproc) grpc_cli && \
    cp /var/local/src/grpc/bins/opt/grpc_cli /usr/local/bin/ && \
    make install && make clean && ldconfig / && \
    rm -rf /var/local/src/grpc

RUN wget -O /usr/local/bin/protoc-gen-grpc_web ${GRPC_WEB_RELEASE} && \
    chmod +x /usr/local/bin/protoc-gen-grpc_web

RUN ln -s /usr/local/bin/grpc_cpp_plugin /usr/local/bin/protoc-gen-grpc_cpp && \
    ln -s /usr/local/bin/grpc_python_plugin /usr/local/bin/protoc-gen-grpc_python

RUN npm -g install ts-protoc-gen

WORKDIR /app