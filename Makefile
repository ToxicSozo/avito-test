APP=reviewer
PKG=github.com/ToxicSozo/avito-test

.PHONY: build run test test-e2e lint gen up down

build:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/$(APP) ./cmd/reviewer

run:
	go run ./cmd/reviewer

test-e2e:
	go test ./test/e2e -run TestEndToEndScenario -count=1

lint:
	golangci-lint run

gen:
	go generate ./api

up:
	docker-compose up --build

down:
	docker-compose down -v
