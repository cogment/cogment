# Build the grpc web proxy
FROM golang:1.17 as proxy

WORKDIR /go
ENV GOPATH=/go
RUN go get github.com/improbable-eng/grpc-web/go/grpcwebproxy@v0.14.1

# Build /usr/local/bin/orchestrator_debug
FROM cogment/orchestrator-build-env:v2.0.0-debug as debugbuild

WORKDIR /workspace
COPY . .

RUN mkdir build_dbg && cd build_dbg      \
  && cmake -DCMAKE_BUILD_TYPE=Debug .. \
  && make -j$(nproc)                   \
  && make install

# Build /usr/local/bin/orchestrator
FROM cogment/orchestrator-build-env:v2.0.0 as build

WORKDIR /workspace
COPY . .

RUN mkdir build_release && cd build_release \
  && cmake -DCMAKE_BUILD_TYPE=Release ..    \
  && make -j$(nproc)                        \
  && make install


FROM ubuntu:20.04

COPY --from=proxy      /go/bin/grpcwebproxy            /usr/local/bin/
COPY --from=build      /usr/local/bin/orchestrator     /usr/local/bin/
COPY --from=debugbuild /usr/local/bin/orchestrator_debug /usr/local/bin/
COPY ./scripts/launch_orchestrator.sh /usr/local/bin/launch_orchestrator.sh
RUN chmod +x /usr/local/bin/launch_orchestrator.sh

WORKDIR /app

ENTRYPOINT ["launch_orchestrator.sh"]

