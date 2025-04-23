# Edge Orchestration-I API Makefile

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

.DEFAULT_GOAL := help
.PHONY: docker-build docker-push test lint license help generate

##### Variables #####
# Project variables
PROJECT_NAME := tinker-actions
BINARY_NAME  := tinker-actions

# Code versions, tags, and so on
VERSION         := $(shell cat VERSION)
VERSION_MAJOR   := $(shell cut -c 1 VERSION)
DOCKER_IMG_NAME := ${PROJECT_NAME}
IMG_VERSION     ?= $(VERSION)
GIT_COMMIT      ?= $(shell git rev-parse HEAD)
BRANCH_NAME     ?= $(shell git rev-parse --abbrev-ref HEAD | sed 's/\//_/g')


ACTIONS := $(shell ls src)
VENV_NAME := venv_$(PROJECT_NAME) 


# Include shared makefile
include ../common.mk

DOCKER_TAG    := ${DOCKER_REGISTRY}/${DOCKER_REPOSITORY}/$(DOCKER_SECTION)/${IMG_NAME}

all: 
	@# Help: Runs build, lint, test stage
	lint docker-build docker-push
DOCKER_PATH_PREFIX  := $(shell echo ${DOCKER_TAG} | sed 's/:.*//')
CREATE_REPO_PREFIX  := $(shell echo ${DOCKER_PATH_PREFIX} | sed 's|^[^/]*/||')

$(ACTIONS): 
	cp -r pkg/ src/$@/pkg/ 
	docker build src/$@ -f src/$@/Dockerfile \
		-t $@:latest \
		--rm \
		--build-arg http_proxy="$(http_proxy)" --build-arg HTTP_PROXY="$(HTTP_PROXY)" \
		--build-arg https_proxy="$(https_proxy)" --build-arg HTTPS_PROXY="$(HTTPS_PROXY)" \
		--build-arg no_proxy="$(no_proxy)" --build-arg NO_PROXY="$(NO_PROXY)"

docker-build: $(ACTIONS)
	@# Help: Build all tinker action images
	@echo "Image build complete"
	@echo "---END MAKEFILE Build---"

dependency-check:
	@# Help: Runs dependency-check stage
	@echo "---MAKEFILE TEST---"
	echo $@
	@echo "---END MAKEFILE TEST---"

build:
	@# Help: Runs build stage
	@echo "---MAKEFILE TEST---"
	echo $@
	@echo "---END MAKEFILE TEST---"

define docker_push_with_retry
sh -c ' \
retry=0; \
max_retries=5; \
while [ $$retry -lt $$max_retries ]; do \
	aws ecr create-repository --region us-west-2 --repository-name ${CREATE_REPO_PREFIX}/${1} || true \
	echo "Tagging and pushing ${1}:${2} (attempt $$((retry+1)))"; \
	docker tag ${1}:latest ${DOCKER_PATH_PREFIX}/${1}:${2}; \
	if docker push ${DOCKER_PATH_PREFIX}/${1}:${2}; then \
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

.PHONY: push-%
push-%: ## Push a specific action image to the registry. This recipe assumes you are already authenticated with the registry.
	$(call docker_push_with_retry,$*,${BRANCH_NAME}); \

.PHONY: release-%
release-%: ## Push a specific action image to the registry. This recipe assumes you are already authenticated with the registry.
	$(call docker_push_with_retry,$*,${VERSION}); \

docker-dev-push: $(addprefix push-,$(ACTIONS))

docker-push: docker-dev-push $(addprefix release-,$(ACTIONS))
	@# Help: Push all tinker action images
	@echo "Image push is complete"
	@echo "---END MAKEFILE docker-push---"	

lint: license yamllint mdlint
	@# Help: Runs lint stage
	@echo "---MAKEFILE LINT---"
	echo $@
	@echo "---END MAKEFILE LINT---"

test:
	@# Help: Runs test stage
	@echo "---MAKEFILE TEST---"
	echo $@
	@echo "---END MAKEFILE TEST---"

helm-build:
	@# Help: Runs helm-build stage
	@echo "---MAKEFILE TEST---"
	echo $@
	@echo "---END MAKEFILE TEST---"

coverage:
	@# Help: Runs coverage stage
	@echo "---MAKEFILE COVERAGE---"
	echo $@
	@echo "---END MAKEFILE COVERAGE---"


