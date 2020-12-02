FROM golang:1.13 as build

RUN apt-get update -y && apt-get install -y protobuf-compiler
WORKDIR /app
COPY . .

RUN make build-linux

FROM ubuntu:20.04

RUN apt-get update -y \
  && apt-get install -y \
  protobuf-compiler \
  curl \
  && rm -rf /var/lib/apt/lists/*

COPY --from=build /app/build/cogment-linux-amd64 /usr/local/bin/cogment

# Only install the docker client
ENV DOCKERVERSION=19.03.12
RUN curl -fsSLO https://download.docker.com/linux/static/stable/x86_64/docker-${DOCKERVERSION}.tgz \
  && tar xzvf docker-${DOCKERVERSION}.tgz --strip 1 \
  -C /usr/local/bin docker/docker \
  && rm docker-${DOCKERVERSION}.tgz


RUN curl -L "https://github.com/docker/compose/releases/download/1.26.0/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose && chmod +x /usr/local/bin/docker-compose

WORKDIR /cogment

ENTRYPOINT ["cogment"]
