# SPDX-FileCopyrightText: 2025 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0


.DEFAULT_GOAL := help

# Optionally include tool version checks, not used in Docker builds
TOOL_VERSION_CHECK ?= 1
GO_CHECK ?=1

##### Variables #####

# Project variables
PROJECT_NAME := pxe-server
BINARY_NAME  := pxe-server

# Code versions, tags, and so on
VERSION         := $(shell cat VERSION)
VERSION_MAJOR   := $(shell cut -c 1 VERSION)
DOCKER_IMG_NAME := ${PROJECT_NAME}
IMG_VERSION     ?= $(VERSION)
GIT_COMMIT      ?= $(shell git rev-parse HEAD)
BRANCH_NAME     ?= $(shell git rev-parse --abbrev-ref HEAD | sed 's/\//_/g')

# Include shared makefile
include ../common.mk

.PHONY: docker-build
docker-build:
	cp ../common.mk ../version.mk ../common_go.mk .
	docker build . -f build/Dockerfile \
		-t $(DOCKER_IMG_NAME):$(VERSION) \
		--build-arg http_proxy="$(http_proxy)" --build-arg HTTP_PROXY="$(HTTP_PROXY)" \
		--build-arg https_proxy="$(https_proxy)" --build-arg HTTPS_PROXY="$(HTTPS_PROXY)" \
		--build-arg no_proxy="$(no_proxy)" --build-arg NO_PROXY="$(NO_PROXY)" \
		--build-arg REPO_URL="$(LABEL_REPO_URL)" \
		--build-arg VERSION="$(LABEL_VERSION)" \
		--build-arg REVISION="$(LABEL_REVISION)" \
		--build-arg BUILD_DATE="$(LABEL_BUILD_DATE)"
	@rm -rf common.mk version.mk common_go.mk

.PHONY: dependency-check
dependency-check:
	@# Help: Runs dependency-check stage
	@echo "---MAKEFILE TEST---"
	echo $@
	@echo "---END MAKEFILE TEST---"

.PHONY: build
build: go-build
	@# Help: Runs build stage
	@echo "---MAKEFILE TEST---"
	echo $@
	@echo "---END MAKEFILE TEST---"

lint: license hadolint mdlint

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
	@# Help: Runs go-build stage
	@echo "---MAKEFILE COVERAGE---"
	echo $@
	@echo "---END MAKEFILE COVERAGE---"

.PHONY: fuzztest
fuzztest:
	@# Help: Runs fuzztest stage
	@echo "---MAKEFILE COVERAGE---"
	echo $@
	@echo "---END MAKEFILE COVERAGE---"
