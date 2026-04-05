BINARY     := psst
INSTALL    := /usr/local/bin
VERSION    := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_FLAGS := -ldflags="-s -w -X github.com/ChengaDev/persist/cmd.Version=$(VERSION)"

.PHONY: all build install uninstall clean tidy test

all: build

## build: compile the binary into the project root
build:
	go build $(BUILD_FLAGS) -o $(BINARY) .

## install: build and copy the binary to INSTALL (default /usr/local/bin)
install: build
	install -m 755 $(BINARY) $(INSTALL)/$(BINARY)
	@echo "Installed $(INSTALL)/$(BINARY)"

## uninstall: remove the installed binary
uninstall:
	rm -f $(INSTALL)/$(BINARY)
	@echo "Removed $(INSTALL)/$(BINARY)"

## clean: remove the local build artifact
clean:
	rm -f $(BINARY)

## tidy: sync go.mod / go.sum
tidy:
	go mod tidy

## test: run all tests (includes slow Argon2id tests)
test:
	go test -race -timeout 180s ./...

## test-short: run tests, skipping slow Argon2id calls
test-short:
	go test -short -timeout 30s ./...

## help: print this message
help:
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'
