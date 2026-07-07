.PHONY: build build-all install test test-coverage lint lint-fix clean deps

# Module and version info
MODULE := github.com/ravi-hq/cli
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -ldflags "\
	-X '$(MODULE)/internal/version.Version=$(VERSION)' \
	-X '$(MODULE)/internal/version.Commit=$(COMMIT)' \
	-X '$(MODULE)/internal/version.BuildDate=$(BUILD_DATE)'"

# ----------------
#    Build
# ----------------

build:
	go build $(LDFLAGS) -o bin/ravi ./cmd/ravi

install:
	go install $(LDFLAGS) ./cmd/ravi

build-all:
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/ravi-darwin-amd64 ./cmd/ravi
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/ravi-darwin-arm64 ./cmd/ravi
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/ravi-linux-amd64 ./cmd/ravi
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o bin/ravi-linux-arm64 ./cmd/ravi
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/ravi-windows-amd64.exe ./cmd/ravi

# ----------------
#    Development
# ----------------

test:
	gotestsum --format pkgname-and-test-fails -- -count=1 ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

lint:
	golangci-lint run

lint-fix:
	golangci-lint run --fix

clean:
	rm -rf bin/

# ----------------
#    Dependencies
# ----------------

deps:
	go mod download
	go mod tidy
