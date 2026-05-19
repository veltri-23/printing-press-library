.PHONY: build test lint install clean

build:
	go build -o bin/theclose-pp-cli ./cmd/theclose-pp-cli

test:
	go test ./...

lint:
	golangci-lint run

install:
	go install ./cmd/theclose-pp-cli

clean:
	rm -rf bin/

build-mcp:
	go build -o bin/theclose-pp-mcp ./cmd/theclose-pp-mcp

install-mcp:
	go install ./cmd/theclose-pp-mcp

build-all: build build-mcp
