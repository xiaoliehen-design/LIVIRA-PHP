.PHONY: run test build

run:
	go run ./cmd/server

test:
	go test ./...

build:
	go build -trimpath -o bin/tpp-app ./cmd/server
