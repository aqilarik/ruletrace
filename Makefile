.PHONY: test lint fmt tidy build

test:
	go test ./... -count=1

lint:
	golangci-lint run ./...

fmt:
	gofmt -w .

tidy:
	go mod tidy

build:
	go build ./cmd/playground
