# Onboarding Manager Makefile

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

.DEFAULT_GOAL := help
.PHONY: test lint license help generate

all: generate build lint test

# Optionally include tool version checks, not used in Docker builds
TOOL_VERSION_CHECK ?= 1
GO_CHECK ?=1

##### Variables #####
## Required Go Dependencies
GOLINT               := true
GOJUNITREPORT        := true
GOCOBERTURA          := true
PROTOCGENDOC         := true
BUF                  := true
PROTOCGENGO          := true
PROTOCGENGOGRPC      := true
PROTOCGENVALIDATEGO  := true

# Project variables
PROJECT_NAME     := onboardingmgr
PROJECT_NICKNAME := om
BINARY_NAME      := $(PROJECT_NAME)

# Code versions, tags, and so on
VERSION         := $(shell cat VERSION)
VERSION_MAJOR   := $(shell cut -c 1 VERSION)
DOCKER_IMG_NAME := ${PROJECT_NAME}
DOCKER_VERSION  ?= $(shell git branch --show-current | sed 's/\//_/g')
GIT_COMMIT      ?= $(shell git rev-parse HEAD)
GIT_TAG_PREFIX  := onboarding-manager/v

DIR_TO_CLEAN    := pkg/api/*

# Test variables
# Set TEST_TARGET to '<testname1>' or '<testname1\|testname2>' to run specific tests or use other  regex '<testname.*>' - example:
#  TEST_TARGET='TestAuth.*' make test
TEST_TARGET       ?= .
# Default TEST_PKG, overwrite to select particular test pakages
TEST_PKG          ?= $(shell go list ./... | grep -v "internal/env" | grep -v "pkg/status" | tr '\n' ,)
TEST_ARGS         := -globalLogLevel=error
TEST_IGNORE_FILES := $(shell go list ./... | grep -v 'pkg/api' | grep -v 'internal/onboardingmgr/onboarding/onboardingmocks' | grep -v 'internal/testing' | grep -v 'cmd/onboardingmgr' | tr '\n' ,)
TEST_COVER        := atomic ./...
TEST_USE_DB       := true
GO_TEST_DEPS      := policy-build certificates

# Yamllint variables
YAML_FILES           := $(shell find . -path './venv_$(PROJECT_NAME)' -path './vendor' -prune -o -type f \( -name '*.yaml' -o -name '*.yml' \) -print )
YAML_IGNORE          := vendor, .github/workflows

# Include shared makefile
include ../common.mk

# Policy related
REGODIR           := $(shell pwd)/rego/
BUNDLE            := "$(shell pwd)/$(OUT_DIR)/policy_bundle.tar.gz"

##### CI Targets #####
generate: buf-gen

build: go-build

lint: license yamllint go-lint hadolint mdlint buf-lint

test: go-test fuzztest

#### Sub-targets ####

go-build: $(OUT_DIR) ## Run go build
	$(GOCMD) fmt ./...
	$(GOCMD) build $(GOEXTRAFLAGS) -o $(OUT_DIR)/$(BINARY_NAME)  cmd/onboardingmgr/main.go

go-run: $(TYPES) $(SERVER) $(CLIENT) ## Run go run
	$(GOCMD) run $(GOEXTRAFLAGS) cmd/onboardingmgr/main.go

buf-update: common-buf-update ## Update buf modules

buf-gen: common-buf-gen ## Compile protoc files into code

buf-lint: common-buf-lint  ## Lint and format protobuf files

fuzztest:
	$(MAKE) db-stop
	$(MAKE) db-start
	$(SCRIPTS_DIR)/fuzz_test.sh "internal/handlers/southbound/southbound_utils_test.go" "internal/handlers/controller/reconcilers/instance_test.go" "internal/handlers/southbound/grpcserver/grpc_server_test.go"
	$(MAKE) db-stop
