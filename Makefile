# Project basic config
APP_NAME := csghub-airgap
VERSION_PACKAGE := main
BUILD_DATE := $(shell date -u +%Y-%m-%d_%H:%M:%S)
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "dev")
GIT_TAG := $(shell git describe --tags --exact-match 2>/dev/null || echo "")

# Version priority: TAG > COMMIT
ifeq ($(GIT_TAG),)
VERSION := $(GIT_COMMIT)
else
VERSION := $(GIT_TAG)
endif

# Build flags
LDFLAGS := -w -s \
	-X $(VERSION_PACKAGE).Version=$(VERSION)

# Target platforms & archs
LINUX_AMD64   := linux-amd64
LINUX_ARM64   := linux-arm64
DARWIN_AMD64  := darwin-amd64
DARWIN_ARM64  := darwin-arm64

# Output directory
OUTPUT_DIR := build
# Main entry file
ENTRY_FILE := cmd/csghub/airgap.go

.PHONY: all build build-linux build-darwin build-all clean run help

# Default: build for current system
all: build

## build: Build binary for current local system
build:
	@mkdir -p $(OUTPUT_DIR)
	@echo "Building ${APP_NAME} (version: ${VERSION}) for local system..."
	go build -ldflags "$(LDFLAGS)" -o $(OUTPUT_DIR)/$(APP_NAME) $(ENTRY_FILE)

## build-darwin: Build for macOS (Intel + Apple Silicon)
build-darwin: build-darwin-amd64 build-darwin-arm64

## build-darwin-amd64: Build for macOS Intel
build-darwin-amd64:
	@mkdir -p $(OUTPUT_DIR)
	@echo "Building ${APP_NAME} (version: ${VERSION}) for Darwin AMD64..."
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(OUTPUT_DIR)/$(APP_NAME)-$(DARWIN_AMD64) $(ENTRY_FILE)

## build-darwin-arm64: Build for macOS Apple Silicon
build-darwin-arm64:
	@mkdir -p $(OUTPUT_DIR)
	@echo "Building ${APP_NAME} (version: ${VERSION}) for Darwin ARM64..."
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(OUTPUT_DIR)/$(APP_NAME)-$(DARWIN_ARM64) $(ENTRY_FILE)

## build-linux: Build for Linux (amd64 + arm64)
build-linux: build-linux-amd64 build-linux-arm64

## build-linux-amd64: Build for Linux AMD64
build-linux-amd64:
	@mkdir -p $(OUTPUT_DIR)
	@echo "Building ${APP_NAME} (version: ${VERSION}) for Linux AMD64..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(OUTPUT_DIR)/$(APP_NAME)-$(LINUX_AMD64) $(ENTRY_FILE)

## build-linux-arm64: Build for Linux ARM64
build-linux-arm64:
	@mkdir -p $(OUTPUT_DIR)
	@echo "Building ${APP_NAME} (version: ${VERSION}) for Linux ARM64..."
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(OUTPUT_DIR)/$(APP_NAME)-$(LINUX_ARM64) $(ENTRY_FILE)

## build-all: Build all platforms (Linux + macOS)
build-all: build-linux build-darwin
	@echo "All platforms built successfully!"

## run: Run project locally
run:
	go run $(ENTRY_FILE)

## clean: Remove build directory and binaries
clean:
	@rm -rf $(OUTPUT_DIR)
	@echo "Clean completed."

## help: Show available commands
help:
	@echo "Usage:"
	@sed -n 's/^##//p' Makefile | column -t -s ':' | sed -e 's/^/ /'