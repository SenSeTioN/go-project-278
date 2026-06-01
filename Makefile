.PHONY: run dev build test lint lint-fix fmt tidy ci clean

BIN := bin/app
PKG := ./cmd/app

run:
	go run $(PKG)

dev:
	air

build:
	mkdir -p bin
	go build -o $(BIN) $(PKG)

test:
	go test -race -v ./...

lint:
	golangci-lint run ./...

lint-fix:
	golangci-lint run --fix ./...

fmt:
	go fmt ./...

tidy:
	go mod tidy

ci: lint test build

clean:
	rm -rf bin
