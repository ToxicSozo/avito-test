APP=reviewer
PKG=github.com/ToxicSozo/avito-test

.PHONY: build run test test-e2e lint generate compose-up compose-down loadtest

build:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/$(APP) ./cmd/reviewer

run:
	go run ./cmd/reviewer

test:
	go test ./...

test-e2e:
	go test ./test/e2e -run TestEndToEndScenario -count=1

loadtest:
	BASE_URL?=http://localhost:8080
	k6 run loadtest/load.js

lint:
	golangci-lint run

generate:
	go generate ./api

compose-up:
	docker-compose up --build

compose-down:
	docker-compose down -v
