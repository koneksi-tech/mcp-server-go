.PHONY: build clean test install run deps build-api run-api build-bridge run-bridge

BINARY_NAME=koneksi-mcp
API_BINARY_NAME=koneksi-api
BRIDGE_BINARY_NAME=koneksi-bridge
BUILD_DIR=build

build:
	go build -o $(BINARY_NAME) ./cmd/koneksi-mcp-server

build-all:
	mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/koneksi-mcp-server
	GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/koneksi-mcp-server
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/koneksi-mcp-server
	GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/koneksi-mcp-server

clean:
	go clean
	rm -f $(BINARY_NAME)
	rm -rf $(BUILD_DIR)

test:
	go test ./...

install:
	go install

deps:
	go mod download
	go mod tidy

run:
	go run ./cmd/koneksi-mcp-server

setup:
	cp .env.example .env
	@echo "Please edit .env with your Koneksi API credentials"

build-api:
	go build -o $(API_BINARY_NAME) ./cmd/koneksi-api-server

run-api:
	go run ./cmd/koneksi-api-server

build-bridge:
	go build -o $(BRIDGE_BINARY_NAME) ./cmd/koneksi-mcp-bridge

run-bridge:
	go run ./cmd/koneksi-mcp-bridge
