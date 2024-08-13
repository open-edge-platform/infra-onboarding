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

OUT_DIR  := $(PWD)/build

# Code versions, tags, and so on
VERSION       := $(shell cat VERSION)
VERSION_MAJOR := $(shell cut -c 1 VERSION)
IMG_NAME      := ${PROJECT_NAME}
IMG_VERSION   ?= main
GIT_COMMIT    ?= $(shell git rev-parse HEAD)

SH_FILES 	:= $(shell find . -type f \( -name '*.sh' \) -print )

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
generate: buf-generate ## generate all files

build: go-build ## Build local binaries

lint: go-lint license shellcheck buf-lint

shellcheck: ## lint shell scripts with shellcheck
	shellcheck --version
	shellcheck -x -S style $(SH_FILES)

test: go-test

#### Sub-targets ####

$(OUT_DIR):
	mkdir -p $(OUT_DIR)

go-build: $(OUT_DIR) ## build resource manager binary
	$(GOCMD) build $(GOEXTRAFLAGS) -o $(OUT_DIR)/$(BINARY_NAME) cmd/$(BINARY_NAME)/main.go

go-test: ${OUT_DIR}
	$(MAKE) db-stop
	$(MAKE) db-start
	go test --race -coverprofile ${OUT_DIR}/cover.out -coverpkg=$$(go list ./... | grep -v "pkg/api/dkammgr/v1" | grep -v "testing" | tr '\n' ,)  -covermode atomic ./...
	gocover-cobertura < ${OUT_DIR}/cover.out > ${OUT_DIR}/coverage.xml
	go tool cover -html=${OUT_DIR}/cover.out -o ${OUT_DIR}/coverage.html
	go tool cover -func ${OUT_DIR}/cover.out -o ${OUT_DIR}/function_coverage.log
	$(MAKE) db-stop

clean: ## delete all build artifacts
	rm -rf $(OUT_DIR) pkg/api/* || true

clean-all: clean ## delete all built artifacts and downloaded tools
	rm -rf $(VENV_NAME) vendor || true
