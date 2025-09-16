# Variables
CONFIG_PATH ?= ./config.json
CONFIG_ENV ?= default

.PHONY: all deps build clean test run-rest run-grpc run-cli docker-build up down logs

all: build

deps:
	go mod tidy

build:
	go build ./cmd/...

clean:
	rm -f bin/* || true

run-rest:
	CONFIG_PATH=$(CONFIG_PATH) CONFIG_ENV=$(CONFIG_ENV) go run ./cmd/rest

run-grpc:
	CONFIG_PATH=$(CONFIG_PATH) CONFIG_ENV=$(CONFIG_ENV) go run ./cmd/grpc

run-cli:
	CONFIG_PATH=$(CONFIG_PATH) CONFIG_ENV=$(CONFIG_ENV) go run ./cmd/cli --customer $(CUSTOMER) --product $(PRODUCT) --file $(FILE)

docker-build:
	docker build -t importer:latest .

up:
	docker compose up --build -d

down:
	docker compose down

logs:
	docker compose logs -f --no-log-prefix
