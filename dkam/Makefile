# DKAM manager Makefile

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

# Project variables
PROJECT_NAME     := dkammgr
PROJECT_NICKNAME := dkam
BINARY_NAME      := $(PROJECT_NAME)

DB_CONTAINER_NAME := $(PROJECT_NAME)-db

OUT_DIR  := $(PWD)/out
SCRIPTS_DIR := ./ci_scripts
# Code versions, tags, and so on
VERSION         := $(shell cat VERSION)
VERSION_MAJOR   := $(shell cut -c 1 VERSION)
DOCKER_IMG_NAME := ${PROJECT_NAME}
DOCKER_VERSION  ?= $(shell git branch --show-current | sed 's/\//_/g')
GIT_COMMIT      ?= $(shell git rev-parse HEAD)
GIT_TAG_PREFIX  := dkam/v

# Create the virtualenv with python tools installed
VENV_NAME := venv_$(PROJECT_NAME)

# Test variables
# Set TEST_TARGET to '<testname1>' or '<testname1\|testname2>' to run specific tests or use other  regex '<testname.*>' - example:
#  TEST_TARGET='TestAuth.*' make test
TEST_TARGET       ?= .
# Default TEST_PKG, overwrite to select particular test pakages
TEST_PKG          ?= $(shell go list ./... | grep -v "pkg/util"  | tr '\n' ,)
TEST_ARGS         := -globalLogLevel=error
TEST_IGNORE_FILES := $(shell go list ./... | grep -v "pkg/api/dkammgr/v1" | grep -v 'cmd' | grep -v 'testing' | tr '\n' ,)
TEST_COVER        := atomic ./...
TEST_USE_DB       := true

BASE_BRANCH ?= main
APIPKG_DIR := pkg/api

# Include shared makefile
include ../common.mk

build: go-build

lint: go-lint license shellcheck mdlint

test: go-test

go-build: $(OUT_DIR) ## build resource manager binary
	$(GOCMD) build $(GOEXTRAFLAGS) -o $(OUT_DIR)/$(BINARY_NAME) cmd/$(PROJECT_NAME)/main.go
