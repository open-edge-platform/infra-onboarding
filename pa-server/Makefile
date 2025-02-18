# Provisioning Artifacts Server Makefile

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

.DEFAULT_GOAL := help
.PHONY: build test lint license help generate

##### Variables #####

# Defining the shell, users and groups
SHELL       := bash -e -o pipefail
CURRENT_UID := $(shell id -u)
CURRENT_GID := $(shell id -g)

# Project variables
PROJECT_NAME := pa-server
BINARY_NAME  := $(PROJECT_NAME)

# Code versions, tags, and so on
VERSION       := $(shell cat VERSION)
VERSION_MAJOR := $(shell cut -c 1 VERSION)
IMG_NAME      := $(PROJECT_NAME)
IMG_VERSION   ?= $(VERSION)
GIT_COMMIT    ?= $(shell git rev-parse HEAD)
BRANCH_NAME   ?= $(shell git rev-parse --abbrev-ref HEAD)
DOCKER_TAG_BRANCH_PUSH	:= true

VENV_NAME = venv_$(PROJECT_NAME)
YAML_FILES           := $(shell find . -path './$(VENV_NAME)' -path './vendor' -prune -o -type f \( -name '*.yaml' -o -name '*.yml' \) -print )
# Create the virtualenv with python tools installed

# Include shared makefile
include ../common.mk

all: build lint test 	
	@# Help: Runs build, lint, test stage
	
build:
	@# Help: Runs build stage
	@echo "---MAKEFILE BUILD---"
	$(MAKE) docker-build
	@echo "---END MAKEFILE Build---"

docker-build:
	docker build pa-server -f pa-server/Dockerfile \
	-t $(IMG_NAME) \
	--rm \
	--build-arg http_proxy="$(http_proxy)" --build-arg HTTP_PROXY="$(HTTP_PROXY)" \
    --build-arg https_proxy="$(https_proxy)" --build-arg HTTPS_PROXY="$(HTTPS_PROXY)" \
    --build-arg no_proxy="$(no_proxy)" --build-arg NO_PROXY="$(NO_PROXY)" \

docker-push:
	# TODO: remove ecr create
	aws ecr create-repository --region us-west-2 --repository-name $(DOCKER_REPOSITORY)/$(IMG_NAME) || true
	docker tag $(IMG_NAME) $(DOCKER_TAG)
	docker push ${DOCKER_TAG}
ifeq ($(DOCKER_TAG_BRANCH_PUSH), true)
	docker tag $(IMG_NAME) $(DOCKER_TAG_BRANCH)
	docker push $(DOCKER_TAG_BRANCH)
endif

lint: license mdlint shellcheck

test:
	@# Help: Runs test stage
	@echo "---MAKEFILE TEST---"
	docker run --name pa-server --rm -d --network host $(IMG_NAME)
	sleep 10
	curl -s "http://localhost:8080/index.php" --noproxy '*' --output result.html
	docker stop pa-server
	if diff -wB "result.html" "reference.html"; then \
		echo "FILES MATCH"; \
		exit 0; \
	else \
		diff "result.html" "reference.html"; \
		echo "FILES DON'T MATCH"; \
		exit 1; \
	fi
	@echo "---END MAKEFILE TEST---"
	
dependency-check:
	@# Help: Runs Dependency Check
	@echo "---MAKEFILE COVERAGE---"
	echo $@
	@echo "---END MAKEFILE COVERAGE---"

coverage:
	@# Help: Runs coverage stage
	@echo "---MAKEFILE COVERAGE---"
	echo $@
	@echo "---END MAKEFILE COVERAGE---"

helm-build:
	@# Help: Runs coverage stage
	@echo "---MAKEFILE COVERAGE---"
	echo $@
	@echo "---END MAKEFILE COVERAGE---"
