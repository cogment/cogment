FROM golang:1.13 as build

RUN apt-get update -y && apt-get install -y protobuf-compiler
WORKDIR /app
COPY . .

RUN go test -v ./...
RUN make build-linux

FROM ubuntu:20.04

RUN apt-get update -y && apt-get install -y protobuf-compiler && rm -rf /var/lib/apt/lists/*
COPY --from=build /app/build/cogment-linux-amd64 /usr/local/bin/cogment

WORKDIR /cogment
CMD ["cogment"]