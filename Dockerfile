FROM golang:1.16 as build

WORKDIR /app

RUN apt-get update -y && apt-get install -y protobuf-compiler

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN make build

FROM ubuntu:20.04

COPY --from=build /app/build/cogment-activity-logger /usr/local/bin/cogment-activity-logger

ENTRYPOINT ["model-registry"]
