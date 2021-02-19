FROM golang:1.15.2 as dev

WORKDIR /go

ARG GO111MODULE=auto
ENV GO111MODULE=${GO111MODULE}
ENV GOPATH=/go

ENV COGMENT_URL=orchestrator:9000

RUN go get github.com/improbable-eng/grpc-web/go/grpcwebproxy

EXPOSE 8080

CMD ["grpcwebproxy", "--backend_addr=orchestrator:9000", "--run_tls_server=false", "--allow_all_origins", "--use_websockets"]
