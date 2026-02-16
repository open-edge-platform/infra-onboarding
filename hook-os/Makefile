# HookOS Makefile

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

.DEFAULT_GOAL := help
.PHONY: build test lint license help generate shellcheck

##### Variables #####
# Project variables
PROJECT_NAME := hook-os
BINARY_NAME  := hook-os

# Code versions, tags, and so on
VERSION                     := $(shell cat VERSION)
VERSION_MAJOR               := $(shell cut -c 1 VERSION)
IMG_VERSION                 ?= $(VERSION)
GIT_COMMIT                  ?= $(shell git rev-parse HEAD)
BRANCH_NAME                 ?= $(shell git rev-parse --abbrev-ref HEAD)


# Include shared makefile
include ../common.mk

COMPONENTS	:= device_discovery fluent-bit
.PHONY: $(COMPONENTS)

all:
	@# Help: Runs lint, test stages
	lint test

$(COMPONENTS):
	cd $@ && \
	chmod +x build.sh && bash build.sh
	
components: $(COMPONENTS)

build: components
	@# Help: Runs build stage
	@echo "---MAKEFILE BUILD---"

lint: license shellcheck mdlint
	@# Help: Runs lint stage
	@echo "---MAKEFILE LINT---"
	echo $@
	@echo "---END MAKEFILE LINT---"

test:
	@# Help: Runs test stage
	@echo "---MAKEFILE TEST---"
	echo $@
	@echo "---END MAKEFILE TEST---"

coverage:
	@# Help: Runs coverage stage
	@echo "---MAKEFILE COVERAGE---"
	echo $@
	@echo "---END MAKEFILE COVERAGE---"

list:
	@# Help: displays make targets
	help

dependency-check:
	@# Help: Runs dependency-check stage
	@echo "---MAKEFILE TEST---"
	echo $@
	@echo "---END MAKEFILE TEST---"

helm-build:
	@# Help: Runs helm-build stage
	@echo "---MAKEFILE TEST---"
	echo $@
	@echo "---END MAKEFILE TEST---"

docker-build:
	@# Help: Runs docker-build stage
	@echo "---MAKEFILE TEST---"
	echo $@
	@echo "---END MAKEFILE TEST---"
