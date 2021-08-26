all: lint build install

install:
	go install

build:
	go build -o build/cogment-model-registry

clean:
	go clean
	rm -f build

lint:
	golangci-lint run

fix-lint:
	golangci-lint run --fix

test:
	go test -v ./...

test-with-report:
	rm -f test_failed.txt
	go test -v ./... 2>&1 > raw_report.txt || echo test_failed > test_failed.txt
	go run github.com/jstemmer/go-junit-report < raw_report.txt > report.xml
	@test ! -f test_failed.txt

run:
	go run main.go
