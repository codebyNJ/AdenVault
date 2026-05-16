# adenVault — local-first vault for developer secrets
# invoked from the shell as: adenV

BINARY      := adenV
PREFIX      ?= /usr/local
INSTALL_DIR := $(PREFIX)/bin
VERSION     := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS     := -s -w -X aden/cmd.Version=$(VERSION)

.PHONY: all build install uninstall test fmt vet tidy clean run release help

all: build

## build           build a stripped binary into ./bin/
build:
	@mkdir -p bin
	@go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) .
	@echo "built bin/$(BINARY)  ($(VERSION))"

## install         install the binary to $(INSTALL_DIR)
install: build
	@install -m 0755 bin/$(BINARY) $(INSTALL_DIR)/$(BINARY)
	@echo "installed $(INSTALL_DIR)/$(BINARY)"

## uninstall       remove the installed binary
uninstall:
	@rm -f $(INSTALL_DIR)/$(BINARY)
	@echo "removed $(INSTALL_DIR)/$(BINARY)"

## test            run the test suite
test:
	@go test ./...

## fmt             gofmt the codebase
fmt:
	@go fmt ./...

## vet             run go vet
vet:
	@go vet ./...

## tidy            tidy go.mod / go.sum
tidy:
	@go mod tidy

## clean           remove build artefacts
clean:
	@rm -rf bin
	@echo "cleaned"

## run             build and run with the supplied ARGS, e.g. make run ARGS=list
run: build
	@./bin/$(BINARY) $(ARGS)

## release         cross-compile binaries for macOS + Linux
release:
	@mkdir -p dist
	@for os in darwin linux; do \
	  for arch in amd64 arm64; do \
	    out=dist/$(BINARY)-$$os-$$arch ; \
	    echo "→ $$out"; \
	    GOOS=$$os GOARCH=$$arch go build -ldflags "$(LDFLAGS)" -o $$out . ; \
	  done ; \
	done

help:
	@grep -E '^## ' Makefile | sed 's/## /  /'
