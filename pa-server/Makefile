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
DOCKER_ENV              := DOCKER_BUILDKIT=1
DOCKER_REGISTRY         ?= amr-registry.caas.intel.com
DOCKER_REPOSITORY       ?= one-intel-edge/maestro-i
DOCKER_TAG              := ${DOCKER_REGISTRY}/${DOCKER_REPOSITORY}/${IMG_NAME}


# Create the virtualenv with python tools installed
VENV_NAME = venv_$(PROJECT_NAME)

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
	
go-build: build

build:
	@# Help: Runs build stage
	@echo "---MAKEFILE BUILD---"
	$(MAKE) docker-build
	@echo "---END MAKEFILE Build---"

docker-build:
	docker build pa-server -f pa-server/Dockerfile \
	-t ${DOCKER_TAG}:${IMG_VERSION} \
	--rm \
	--build-arg http_proxy="$(http_proxy)" --build-arg HTTP_PROXY="$(HTTP_PROXY)" \
    --build-arg https_proxy="$(https_proxy)" --build-arg HTTPS_PROXY="$(HTTPS_PROXY)" \
    --build-arg no_proxy="$(no_proxy)" --build-arg NO_PROXY="$(NO_PROXY)" \

docker-push:
	docker push ${DOCKER_TAG}:${IMG_VERSION}

lint-ci: license

# Maintenance and lint targets
license: $(VENV_NAME) ## Check licensing with the reuse tool.
	. ./$</bin/activate; set -u;\
	reuse --version;\
	reuse --root . lint

lint:
	@# Help: Runs lint stage
	@echo "---MAKEFILE LINT---"
	$(MAKE) lint-ci
	@echo "---END MAKEFILE LINT---"

test:
	@# Help: Runs test stage
	@echo "---MAKEFILE TEST---"
	docker run --name pa-server --rm -d --network container:$$HOSTNAME   ${DOCKER_TAG}:${IMG_VERSION}
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
	
coverage:
	@# Help: Runs coverage stage
	@echo "---MAKEFILE COVERAGE---"
	echo $@
	@echo "---END MAKEFILE COVERAGE---"

list: 
	@# Help: displays make targets
	help

dependency-check:
	@# Help: Runs Dependency Check
	@echo "---MAKEFILE COVERAGE---"
	echo $@
	@echo "---END MAKEFILE COVERAGE---"

help:	
	@printf "%-20s %s\n" "Target" "Description"
	@printf "%-20s %s\n" "------" "-----------"
	@make -pqR : 2>/dev/null \
        | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' \
        | sort \
        | egrep -v -e '^[^[:alnum:]]' -e '^$@$$' \
        | xargs -I _ sh -c 'printf "%-20s " _; make _ -nB | (grep -i "^# Help:" || echo "") | tail -1 | sed "s/^# Help: //g"'