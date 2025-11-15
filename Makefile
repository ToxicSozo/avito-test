APP=reviewer

.PHONY: build run test generate lint

build:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/$(APP) ./cmd

run:
	go run ./cmd

test:
	go test ./...

generate:
	go generate ./internal/api

lint:
	golangci-lint run

load-test:
	@BASE_URL?=http://localhost:8080
	@USER_TOKEN?=super-secret-user
	BASE_URL=$(BASE_URL) USER_TOKEN=$(USER_TOKEN) k6 run loadtest/load.js
