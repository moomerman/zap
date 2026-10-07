# builds zapd and the zap command into bin/
build:
	go build -o bin/zapd ./cmd/zapd
	go build -o bin/zap ./cmd/zap

test:
	go test ./...

.PHONY: build test
