.PHONY: test build vet fmt

test:
	go test ./...

build:
	go build -trimpath -ldflags="-s -w" -o bin/vantage ./cmd/vantage

vet:
	go vet ./...

fmt:
	gofmt -w cmd internal
