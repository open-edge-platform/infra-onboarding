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
BINARY_NAME  := midkammgr

# Code versions, tags, and so on
VERSION       := $(shell cat VERSION)
VERSION_MAJOR := $(shell cut -c 1 VERSION)
IMG_NAME      := ${PROJECT_NAME}
IMG_VERSION   ?= main
GIT_COMMIT    ?= $(shell git rev-parse HEAD)
DOCKER_ENV              := DOCKER_BUILDKIT=1
DOCKER_REGISTRY         ?= amr-registry.caas.intel.com
DOCKER_REPOSITORY       ?= one-intel-edge/maestro-i
DOCKER_TAG              := ${DOCKER_REGISTRY}/${DOCKER_REPOSITORY}/${IMG_NAME}

SCRIPTS_DIR     := ./pkg/script/Installer
# Postgres DB configuration and credentials for testing. This mimics the Aurora
# production environment.
export PGUSER=admin
export PGHOST=localhost
export PGDATABASE=postgres
export PGPORT=5432
export PGPASSWORD=pass
export PGSSLMODE=disable


# Tools versions
GOLINTVERSION_HAVE        := $(shell golangci-lint version | sed 's/.*version //' | sed 's/ .*//')
POSTGRES_VERSION          := 14.9
# Create the virtualenv with python tools installed
VENV_NAME = venv_midkammgr

$(VENV_NAME): requirements.txt
	echo "Creating virtualenv $@"
	python3 -m venv $@;\
	. ./$@/bin/activate; set -u;\
	python3 -m pip install --upgrade pip;\
	python3 -m pip install -r requirements.txt

##### CI Targets #####

all:
	@# Help: Runs build, lint, test stage
	build lint test

docker-build: go-vendor
	@export DOCKER_REGISTRY=${DOCKER_REGISTRY} DOCKER_REPOSITORY=${DOCKER_REPOSITORY} IMG_NAME=${IMG_NAME} VERSION=${IMG_VERSION} &&  earthly -V +all

docker-push:
	docker push ${DOCKER_REGISTRY}/${DOCKER_REPOSITORY}/${IMG_NAME}:${IMG_VERSION}

#lint-ci: license go-lint

lint-ci: go-lint license shellcheck

# Maintenance and lint targets
license: $(VENV_NAME) ## Check licensing with the reuse tool.
	. ./$</bin/activate; set -u;\
	reuse --version;\
	reuse --root . lint

go-lint: ## run go lint
	golangci-lint linters

shellcheck: ## Check all shell scripts
	shellcheck --version
	shellcheck $(SCRIPTS_DIR)

go-vendor:
	go mod vendor

lint:
	@# Help: Runs lint stage
	@echo "---MAKEFILE LINT---"
	echo $@
	@echo "---END MAKEFILE LINT---"

test:
	go test --race -coverprofile cover.out -coverpkg=$$(go list ./... | grep -v "api/grpc" | tr '\n' ,)  -covermode atomic ./...
	gocover-cobertura < cover.out > coverage.xml
	
coverage:
	@# Help: Runs coverage stage
	@echo "---MAKEFILE COVERAGE---"
	echo $@
	@echo "---END MAKEFILE COVERAGE---"

list: 
	@# Help: displays make targets
	help

help:	
	@printf "%-20s %s\n" "Target" "Description"
	@printf "%-20s %s\n" "------" "-----------"
	@make -pqR : 2>/dev/null \
        | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' \
        | sort \
        | egrep -v -e '^[^[:alnum:]]' -e '^$@$$' \
        | xargs -I _ sh -c 'printf "%-20s " _; make _ -nB | (grep -i "^# Help:" || echo "") | tail -1 | sed "s/^# Help: //g"'
	
