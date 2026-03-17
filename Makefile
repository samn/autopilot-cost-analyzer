BINARY    := gke-cost-analyzer
MODULE    := github.com/samn/gke-cost-analyzer
VERSION   ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT    := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE      := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

GOOS      ?= linux
GOARCH    ?= amd64

LDFLAGS   := -s -w \
  -X '$(MODULE)/cmd.version=$(VERSION)' \
  -X '$(MODULE)/cmd.commit=$(COMMIT)' \
  -X '$(MODULE)/cmd.date=$(DATE)'

DIST_DIR  := dist

.PHONY: build clean test lint

## build: Compile the binary for the target OS/arch (default: linux/amd64).
build:
	@mkdir -p $(DIST_DIR)
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) \
	  go build -trimpath -ldflags "$(LDFLAGS)" \
	  -o $(DIST_DIR)/$(BINARY)-$(GOOS)-$(GOARCH) .

## test: Run all tests with race detection.
test:
	go test -race -v ./...

## lint: Run golangci-lint.
lint:
	golangci-lint run ./...

## clean: Remove build artifacts.
clean:
	rm -rf $(DIST_DIR)
