FROM golang:1.16 as build

WORKDIR /app

RUN apt-get update -y && apt-get install -y protobuf-compiler

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN make build

FROM ubuntu:20.04

VOLUME /data
ENV COGMENT_MODEL_REGISTRY_ARCHIVE_DIR=/data

COPY --from=build /app/build/cogment-model-registry /usr/local/bin/cogment-model-registry

ENTRYPOINT ["cogment-model-registry"]
