LD_FLAGS="-s -w"

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOFMT=gofmt
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BUILD_ARCH=amd64
BINARY_NAME=cogment
BINARY_LINUX=$(BINARY_NAME)-linux-${BUILD_ARCH}
BINARY_MAC=$(BINARY_NAME)-macOS-${BUILD_ARCH}
BINARY_WINDOWS=$(BINARY_NAME)-windows-${BUILD_ARCH}.exe

.PHONY: build

generate:
	$(GOCMD) run github.com/markbates/pkger/cmd/pkger

build: generate
	$(GOBUILD) -o build/$(BINARY_NAME) -v

test: generate
	$(GOTEST) -v ./...

test-update-snapshots: generate
	UPDATE_SNAPSHOTS=true $(GOTEST) -v ./...

install: generate test
	$(GOCMD) install

test-with-report: generate
	rm -f test_failed.txt
	$(GOTEST) -v ./... 2>&1 > raw_report.txt || echo test_failed > test_failed.txt
	$(GOCMD) run github.com/jstemmer/go-junit-report < raw_report.txt > report.xml
	@test ! -f test_failed.txt

clean:
	$(GOCLEAN)
	rm -f pkged.go
	rm -f $(BINARY_NAME) $(BINARY_LINUX) $(BINARY_MAC)

run: build
	build/$(BINARY_NAME)

fmt:
	$(GOFMT) -l -w ./..

lint: check-fmt

check-fmt:
	$(GOFMT) -l ./..
	@test -z "$(shell $(GOFMT) -l ./..)" || echo "[WARN] Fix formatting issues with 'make fmt'"

check-codingstyle:
	$(GOCMD) run golang.org/x/lint/golint  ./...

# Cross compilation
build-linux: generate
	CGO_ENABLED=0 GOOS=linux GOARCH=${BUILD_ARCH} $(GOBUILD) -ldflags ${LD_FLAGS} -o build/$(BINARY_LINUX) -v

build-mac: generate
	CGO_ENABLED=0 GOOS=darwin GOARCH=${BUILD_ARCH} $(GOBUILD) -ldflags ${LD_FLAGS} -o build/$(BINARY_MAC) -v

build-windows: generate
	CGO_ENABLED=0 GOOS=windows GOARCH=${BUILD_ARCH} $(GOBUILD) -ldflags ${LD_FLAGS} -o build/$(BINARY_WINDOWS) -v

release: build-linux build-mac build-windows
