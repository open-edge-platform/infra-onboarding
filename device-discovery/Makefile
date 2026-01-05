# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt

# Binary name
BINARY_NAME=device-discovery
BINARY_PATH=./$(BINARY_NAME)

# Build directory
CMD_DIR=./cmd/device-discovery

# Docker parameters
DOCKER_IMAGE=device-discovery
DOCKER_TAG=latest

# Build flags
CGO_ENABLED=0
GOOS=linux
GOARCH=amd64

.PHONY: all build clean test fmt vet docker-build help

all: test build

build: ## Build the binary
	@echo "Building $(BINARY_NAME)..."
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) $(GOBUILD) -v -o $(BINARY_PATH) $(CMD_DIR)

clean: ## Clean build artifacts
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

test: ## Run tests
	@echo "Running tests..."
	$(GOTEST) -v ./...

fmt: ## Format code
	@echo "Formatting code..."
	$(GOFMT) -s -w .

vet: ## Run go vet
	@echo "Running go vet..."
	$(GOCMD) vet ./...

tidy: ## Tidy go modules
	@echo "Tidying go modules..."
	$(GOMOD) tidy

docker-build: ## Build docker image
	@echo "Building Docker image..."
	./build.sh
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
