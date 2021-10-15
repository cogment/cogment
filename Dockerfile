FROM golang:1.16 as build

WORKDIR /app
RUN apt-get update -y && apt-get install -y unzip && \
  PROTOC_VERSION=3.18.1 && \
  curl -LO --silent https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-linux-x86_64.zip && \
  unzip protoc-${PROTOC_VERSION}-linux-x86_64.zip -d /usr/local && \
  chmod +x /usr/local/bin/protoc

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN make build

FROM ubuntu:20.04

COPY --from=build /app/build/cogment-trial-datastore /usr/local/bin/cogment-trial-datastore

ENTRYPOINT ["cogment-trial-datastore"]
