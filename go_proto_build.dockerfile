FROM golang:latest

RUN apt-get update
RUN apt-get install -y protobuf-compiler vim

RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.27.1
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.1.0

COPY ./packages/grpc_api/cogment/api /workspace/cogment/api
RUN mkdir -p /workspace/out

RUN protoc --go_out=/workspace/out \
           --go_opt=paths=source_relative \
           --go-grpc_out=/workspace/out \
           --go-grpc_opt=paths=source_relative \
           --proto_path=/workspace \
           cogment/api/agent.proto \
           cogment/api/common.proto \
           cogment/api/datalog.proto \
           cogment/api/directory.proto \
           cogment/api/environment.proto \
           cogment/api/health.proto \
           cogment/api/hooks.proto \
           cogment/api/model_registry.proto \
           cogment/api/orchestrator.proto

