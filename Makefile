VERSION ?= dev
COMMIT  := $(shell git rev-parse --short HEAD)
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

.PHONY: build build-linux build-windows build-all test lint clean

build:
	@mkdir -p dist
	go build -ldflags "$(LDFLAGS)" -o dist/ow ./cmd/ow

build-linux:
	@mkdir -p dist
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/ow-linux-amd64 ./cmd/ow

build-windows:
	@mkdir -p dist
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/ow-windows-amd64.exe ./cmd/ow

build-all: build-linux build-windows

test:
	go test -race -coverprofile=coverage.out ./...

lint:
	golangci-lint run ./...

clean:
	rm -rf dist/ coverage.out
