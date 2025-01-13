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
PROJECT_NAME := tinker-actions
BINARY_NAME  := tinker-actions

# Code versions, tags, and so on
VERSION       := $(shell cat VERSION)
VERSION_MAJOR := $(shell cut -c 1 VERSION)
IMG_NAME      := ${PROJECT_NAME}
IMG_VERSION   ?= $(VERSION)
GIT_COMMIT    ?= $(shell git rev-parse HEAD)
BRANCH_NAME   ?= $(shell git rev-parse --abbrev-ref HEAD)
DOCKER_ENV              := DOCKER_BUILDKIT=1
DOCKER_REGISTRY         ?= amr-registry.caas.intel.com
DOCKER_REPOSITORY       ?= one-intel-edge/edge-node
DOCKER_TAG              := ${DOCKER_REGISTRY}/${DOCKER_REPOSITORY}/${IMG_NAME}

ACTION_ARRAY := \
	securebootflag \
	efibootset \
	fde \
	cred_copy \
	caddy_proxy \
	fluentbit \
	image2disk \
	cexec \
	writefile \
	erase_non_removable_disks \
	hook_dind \
	device-discovery \
	kernelupgrd \
	tiberos_partition

# Create the virtualenv with python tools installed
VENV_NAME = venv_midkammgr

# CI SCRIPTS
SCRIPTS_DIR=./ci_scripts

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

build:
	@# Help: Runs build stage
	@echo "---MAKEFILE BUILD---"
	./setup_actions.sh
	@echo "---END MAKEFILE Build---"

define docker_push_with_retry
sh -c ' \
retry=0; \
max_retries=5; \
while [ $$retry -lt $$max_retries ]; do \
	echo "Tagging and pushing ${1}:${2} (attempt $$((retry+1)))"; \
	docker tag ${1}:latest ${DOCKER_TAG}/${1}:${2}; \
	if docker push ${DOCKER_TAG}/${1}:${2}; then \
		echo "Successfully pushed ${1}:${2}"; \
		break; \
	else \
		echo "Failed to push ${1}:${2}, retrying..."; \
		retry=$$((retry+1)); \
		sleep 5; \
	fi; \
done; \
if [ $$retry -eq $$max_retries ]; then \
	echo "Failed to push ${1}:${2} after $$max_retries attempts"; \
	exit 1; \
fi'
endef

docker-push:
	@$(foreach action,$(ACTION_ARRAY), \
	       $(call docker_push_with_retry,${action},${BRANCH_NAME}); \
		   $(call docker_push_with_retry,${action},${VERSION}); \
		   echo ""; \
       )

docker-dev-push:
	@$(foreach action,$(ACTION_ARRAY), \
	       $(call docker_push_with_retry,${action},${BRANCH_NAME}); \
		   echo ""; \
       )

lint-ci: license

# Maintenance and lint targets
license: $(VENV_NAME) ## Check licensing with the reuse tool.
	. ./$</bin/activate; set -u;\
	reuse --version;\
	reuse --root . lint

lint:
	@# Help: Runs lint stage
	@echo "---MAKEFILE LINT---"
	@echo "---END MAKEFILE LINT---"

test:
	@# Help: Runs test stage
	@echo "---MAKEFILE TEST---"
	./test.sh
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

fuzztest:
	$(SCRIPTS_DIR)/fuzz_test.sh "tinker_actions/device_discovery/device-discovery_test.go"
