# Maestro-I API Makefile

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
DOCKER_REPOSITORY       ?= one-intel-edge/edgenode
DOCKER_TAG              := ${DOCKER_REGISTRY}/${DOCKER_REPOSITORY}/${IMG_NAME}


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

build:
	@# Help: Runs build stage
	@echo "---MAKEFILE BUILD---"
	./setup_actions.sh
	@echo "---END MAKEFILE Build---"

docker-push:
	docker tag efibootset:latest ${DOCKER_TAG}/efibootset:${BRANCH_NAME}
	docker tag fdoclient_action:latest ${DOCKER_TAG}/fdoclient_action:${BRANCH_NAME}
	docker tag store_alpine:latest ${DOCKER_TAG}/store_alpine:${BRANCH_NAME}
	docker tag create_partition:latest ${DOCKER_TAG}/create_partition:${BRANCH_NAME}
	docker tag fde:latest ${DOCKER_TAG}/fde:${BRANCH_NAME}
	docker tag cred_copy:latest ${DOCKER_TAG}/cred_copy:${BRANCH_NAME}
	docker tag client_auth:latest ${DOCKER_TAG}/client_auth:${BRANCH_NAME}
	docker tag nginx_proxy_action:latest ${DOCKER_TAG}/nginx_proxy_action:${BRANCH_NAME}

	docker push ${DOCKER_TAG}/efibootset:${BRANCH_NAME}
	docker push ${DOCKER_TAG}/fdoclient_action:${BRANCH_NAME}
	docker push ${DOCKER_TAG}/store_alpine:${BRANCH_NAME}
	docker push ${DOCKER_TAG}/create_partition:${BRANCH_NAME}
	docker push ${DOCKER_TAG}/fde:${BRANCH_NAME}
	docker push ${DOCKER_TAG}/cred_copy:${BRANCH_NAME}
	docker push ${DOCKER_TAG}/client_auth:${BRANCH_NAME}
	docker push ${DOCKER_TAG}/nginx_proxy_action:${BRANCH_NAME}

	docker tag efibootset:latest ${DOCKER_TAG}/efibootset:${VERSION}
	docker tag fdoclient_action:latest ${DOCKER_TAG}/fdoclient_action:${VERSION}
	docker tag store_alpine:latest ${DOCKER_TAG}/store_alpine:${VERSION}
	docker tag create_partition:latest ${DOCKER_TAG}/create_partition:${VERSION}
	docker tag fde:latest ${DOCKER_TAG}/fde:${VERSION}
	docker tag cred_copy:latest ${DOCKER_TAG}/cred_copy:${VERSION}
	docker tag client_auth:latest ${DOCKER_TAG}/client_auth:${VERSION}
	docker tag nginx_proxy_action:latest ${DOCKER_TAG}/nginx_proxy_action:${VERSION}

	docker push ${DOCKER_TAG}/efibootset:${VERSION}
	docker push ${DOCKER_TAG}/fdoclient_action:${VERSION}
	docker push ${DOCKER_TAG}/store_alpine:${VERSION}
	docker push ${DOCKER_TAG}/create_partition:${VERSION}
	docker push ${DOCKER_TAG}/fde:${VERSION}
	docker push ${DOCKER_TAG}/cred_copy:${VERSION}
	docker push ${DOCKER_TAG}/client_auth:${VERSION}
	docker push ${DOCKER_TAG}/nginx_proxy_action:${VERSION}


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
