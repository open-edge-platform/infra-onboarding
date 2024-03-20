# Edge Orchestration-I API Makefile

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
PROJECT_NAME := hook-os
BINARY_NAME  := hook-os

# Code versions, tags, and so on
VERSION       := $(shell cat VERSION)
VERSION_MAJOR := $(shell cut -c 1 VERSION)
IMG_NAME      := ${PROJECT_NAME}
IMG_VERSION   ?= $(VERSION)
GIT_COMMIT    ?= $(shell git rev-parse HEAD)
DOCKER_ENV              := DOCKER_BUILDKIT=1
DOCKER_REGISTRY         ?= amr-registry.caas.intel.com
DOCKER_REPOSITORY       ?= one-intel-edge/maestro-i
DOCKER_TAG              := ${DOCKER_REGISTRY}/${DOCKER_REPOSITORY}/${IMG_NAME}

# Create the virtualenv with python tools installed
VENV_NAME = venv_hookos

$(VENV_NAME): requirements.txt
	echo "Creating virtualenv $@"
	python3 -m venv $@;\
	. ./$@/bin/activate; set -u;\
	python3 -m pip install --upgrade pip;\
	python3 -m pip install -r requirements.txt

all: 
	@# Help: Runs build, lint, test stages
	build lint test 	
	
build:
	@# Help: Runs build stage
	@echo "---MAKEFILE BUILD---"
	@echo "---BUILD started---"
	@echo "--Updated step below--"
	chmod +x build_hookos.sh && bash -x build_hookos.sh
	@echo "---END MAKEFILE Build---"
	
lint-ci: license

# Maintenance and lint targets
license: $(VENV_NAME) ## Check licensing with the reuse tool.
	. ./$</bin/activate; set -u;\
	reuse --version;\
	reuse --root . lint

lint:
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

help:	
	@printf "%-20s %s\n" "Target" "Description"
	@printf "%-20s %s\n" "------" "-----------"
	@make -pqR : 2>/dev/null \
        | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' \
        | sort \
        | egrep -v -e '^[^[:alnum:]]' -e '^$@$$' \
        | xargs -I _ sh -c 'printf "%-20s " _; make _ -nB | (grep -i "^# Help:" || echo "") | tail -1 | sed "s/^# Help: //g"'
	
