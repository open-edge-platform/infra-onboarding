# SPDX-FileCopyrightText: 2025 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0


.DEFAULT_GOAL := help

# Optionally include tool version checks, not used in Docker builds
TOOL_VERSION_CHECK ?= 1
GO_CHECK ?=1

##### Variables #####
## Required Go Dependencies
GOLINT               := true
GOJUNITREPORT        := true
GOCOBERTURA          := true

# Project variables
PROJECT_NAME := tink-worker
BINARY_NAME  := tink-worker

# Code versions, tags, and so on
VERSION         := $(shell cat VERSION)
VERSION_MAJOR   := $(shell cut -c 1 VERSION)
DOCKER_IMG_NAME := ${PROJECT_NAME}
IMG_VERSION     ?= $(VERSION)
GIT_COMMIT      ?= $(shell git rev-parse HEAD)
BRANCH_NAME     ?= $(shell git rev-parse --abbrev-ref HEAD | sed 's/\//_/g')



# Include shared makefile
include ../common.mk

.PHONY: docker-push
docker-push:
	@# Help: Push all images
	@echo "Image push is complete"
	@echo "---END MAKEFILE docker-push---"

.PHONY: docker-build
docker-build:
	@# Help: Build all images
	@echo "Image build complete"
	@echo "---END MAKEFILE Build---"

.PHONY: dependency-check
dependency-check:
	@# Help: Runs dependency-check stage
	@echo "---MAKEFILE TEST---"
	echo $@
	@echo "---END MAKEFILE TEST---"

.PHONY: go-dependency-check
go-dependency-check:
	@# Help: Runs go-dependency-check stage
	@echo "---MAKEFILE TEST---"
	echo $@
	@echo "---END MAKEFILE TEST---"

.PHONY: build
build: go-build
	@# Help: Runs build stage
	@echo "---MAKEFILE TEST---"
	echo $@
	@echo "---END MAKEFILE TEST---"

.PHONY: lint
lint: license go-lint
	@# Help: Runs lint stage
	@echo "---MAKEFILE LINT---"
	echo $@
	@echo "---END MAKEFILE LINT---"

.PHONY: test
test: fuzztest
	@# Help: Runs test stage
	@echo "---MAKEFILE TEST---"
	echo $@
	@echo "---END MAKEFILE TEST---"

.PHONY: helm-build
helm-build:
	@# Help: Runs helm-build stage
	@echo "---MAKEFILE TEST---"
	echo $@
	@echo "---END MAKEFILE TEST---"

.PHONY: coverage
coverage:
	@# Help: Runs coverage stage
	@echo "---MAKEFILE COVERAGE---"
	echo $@
	@echo "---END MAKEFILE COVERAGE---"

.PHONY: go-build
go-build:
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(LDFLAGS) -o ./tink-worker ./cmd/tink-worker

.PHONY: fuzztest
fuzztest:
	go test -v -fuzz FuzzNewContainerName -fuzz FuzzParseCmdLine -fuzztime 30s -parallel=4 ./cmd/tink-worker/worker
