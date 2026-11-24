.PHONY: run test race fmt vet build

run:
	go run ./cmd/server

test:
	go test ./...

race:
	go test -race ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

build:
	go build -trimpath -o bin/ ./cmd/server
