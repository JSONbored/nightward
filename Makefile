BIN ?= nightward
PREFIX ?= $(HOME)/.local

.PHONY: test build install-local

test:
	go test ./...

build:
	go build -o bin/$(BIN) ./cmd/nightward

install-local:
	mkdir -p $(PREFIX)/bin
	go build -o $(PREFIX)/bin/$(BIN) ./cmd/nightward
