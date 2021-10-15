all: lint build install

install:
	go install

generate-protos:
	go generate -tags tools

build: generate-protos
	go build -o build/cogment-trial-datastore

clean:
	go clean
	rm -f build

lint: generate-protos
	golangci-lint run

fix-lint: generate-protos
	golangci-lint run --fix

test: generate-protos
	go test -v ./...

test-with-report: generate-protos
	rm -f test_failed.txt
	go test -v ./... 2>&1 > raw_report.txt || echo test_failed > test_failed.txt
	go run github.com/jstemmer/go-junit-report < raw_report.txt > report.xml
	@test ! -f test_failed.txt

run:
	go run main.go
