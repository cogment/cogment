FROM golang:1.16 as build

WORKDIR /app

# Retrieve and build transitive dependencies to help with caching
# Method suggested in https://github.com/golang/go/issues/27719
COPY go.mod go.sum ./
RUN go mod graph | awk '$1 !~ /@/ { print $2 }' | xargs -r go get

COPY . .
RUN make build

FROM ubuntu:20.04

VOLUME /data

COPY --from=build /app/build/cogment-model-registry /usr/local/bin/cogment-model-registry

ENV COGMENT_MODEL_REGISTRY_PORT=9000
ENV COGMENT_MODEL_REGISTRY_ARCHIVE_DIR=/data

ENTRYPOINT ["model-registry"]
