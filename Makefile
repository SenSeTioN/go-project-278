.PHONY: run dev air build test lint lint-fix fmt tidy ci clean sqlc migrate-up migrate-down migrate-status migrate-create

BIN            := bin/app
PKG            := .
MIGRATIONS_DIR := db/migrations

ifneq (,$(wildcard ./.env))
  include .env
  export
endif

run:
	go run $(PKG)

dev:
	npm run dev

air:
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

sqlc:
	sqlc generate

migrate-up:
	goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" up

migrate-down:
	goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" down

migrate-status:
	goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" status

migrate-create:
	@test -n "$(NAME)" || (echo "usage: make migrate-create NAME=add_users"; exit 1)
	goose -dir $(MIGRATIONS_DIR) -s create $(NAME) sql
