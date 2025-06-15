.PHONY: build clean test install run deps

BINARY_NAME=koneksi-mcp
BUILD_DIR=build

build:
	go build -o $(BINARY_NAME) main.go

build-all:
	mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 main.go
	GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 main.go
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 main.go
	GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe main.go

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
	go run main.go

setup:
	cp .env.example .env
	@echo "Please edit .env with your Koneksi API credentials"