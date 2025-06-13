# SPDX-FileCopyrightText: 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

.DEFAULT_GOAL := help
.PHONY: lint docker-build

all: lint docker-build

# Optionally include tool version checks, not used in Docker builds
TOOL_VERSION_CHECK ?= 1
GO_CHECK ?=0

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

lint: license hadolint mdlint

test:
	@# Help: Runs test stage
	@echo "---MAKEFILE TEST---"
	echo $@
	@echo "---END MAKEFILE TEST---"
