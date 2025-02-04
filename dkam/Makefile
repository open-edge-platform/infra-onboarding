# DKAM manager Makefile

# SPDX-FileCopyrightText: (C) 2022 Intel Corporation
# SPDX-License-Identifier: LicenseRef-Intel

.DEFAULT_GOAL := help
.PHONY: build test lint license help generate

##### Variables #####

# Defining the shell, users and groups
SHELL       := bash -e -o pipefail
CURRENT_UID := $(shell id -u)
CURRENT_GID := $(shell id -g)

# Project variables
PROJECT_NAME := dkammgr
BINARY_NAME  := dkammgr

DB_CONTAINER_NAME := $(PROJECT_NAME)-db

BUILD_DIR  := $(PWD)/build
SCRIPTS_DIR := ./ci_scripts
# Code versions, tags, and so on
VERSION       := $(shell cat VERSION)
VERSION_MAJOR := $(shell cut -c 1 VERSION)
IMG_NAME      := ${PROJECT_NAME}
IMG_VERSION   ?= main
GIT_COMMIT    ?= $(shell git rev-parse HEAD)

# Create the virtualenv with python tools installed
VENV_NAME = venv_midkammgr

BASE_BRANCH ?= main
APIPKG_DIR := pkg/api

# Include shared makefile
include common.mk

$(VENV_NAME): requirements.txt
	echo "Creating virtualenv $@"
	python3 -m venv $@;\
	. ./$@/bin/activate; set -u;\
	python3 -m pip install --upgrade pip;\
	python3 -m pip install -r requirements.txt

#### Development Targets ####
build: go-build ## Build local binaries

## TODO: add hadolint and yamlint
lint: go-lint license shellcheck

go-lint: $(BUILD_DIR)## lint code
	golangci-lint --version
	golangci-lint run --config .golangci.yml

test: go-test fuzztest

#### Sub-targets ####

$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

go-build: $(BUILD_DIR) ## build resource manager binary
	$(GOCMD) build $(GOEXTRAFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) cmd/$(BINARY_NAME)/main.go

go-test: $(BUILD_DIR)
	$(MAKE) db-stop
	$(MAKE) db-start
	$(GOCMD) test -race -v -p 1 -coverpkg=$$(go list ./... | grep -v "pkg/api/dkammgr/v1" | grep -v "cmd" | grep -v "testing" | tr '\n' ,) -coverprofile ${BUILD_DIR}/coverage.out -covermode atomic ./... | tee \
		>(go-junit-report -set-exit-code > ${BUILD_DIR}/report.xml)
	gocover-cobertura < ${BUILD_DIR}/coverage.out > ${BUILD_DIR}/coverage.xml
	$(GOCMD) tool cover -html=${BUILD_DIR}/coverage.out -o ${BUILD_DIR}/coverage.html
	$(GOCMD) tool cover -func ${BUILD_DIR}/coverage.out -o ${BUILD_DIR}/function_coverage.log
	$(MAKE) db-stop

clean: ## delete all build artifacts
	rm -rf $(BUILD_DIR) pkg/api/* || true

clean-all: clean ## delete all built artifacts and downloaded tools
	rm -rf $(VENV_NAME) vendor || true
	
fuzztest: $(BUILD_DIR)
	$(MAKE) db-stop
	$(MAKE) db-start
	$(SCRIPTS_DIR)/fuzz_test.sh "internal/handlers/controller/reconcilers/os_test.go"
	$(MAKE) db-stop
