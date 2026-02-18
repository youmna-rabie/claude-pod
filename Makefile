.PHONY: build test lint clean run

BINARY := gateway
BUILD_DIR := bin

build:
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/gateway

run: build
	./$(BUILD_DIR)/$(BINARY)

test:
	go test ./...

lint:
	golangci-lint run ./...

clean:
	rm -rf $(BUILD_DIR)
