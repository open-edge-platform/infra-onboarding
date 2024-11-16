# Edge Infrastructure Manager Onboarding Manager Makefile

# SPDX-FileCopyrightText: (C) 2024 Intel Corporation
# SPDX-License-Identifier: LicenseRef-Intel

.DEFAULT_GOAL := help
.PHONY: docker-build test lint license help

##### Variables #####

# Defining the shell, users and groups
SHELL       := bash -e -o pipefail
CURRENT_UID := $(shell id -u)
CURRENT_GID := $(shell id -g)

# Project variables
PROJECT_NAME := onboardingmgr
BINARY_NAME  := mionboarding

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
DOCKER_LABEL_REPO_URL   ?= $(shell git remote get-url $(shell git remote | head -n 1))
DOCKER_LABEL_VERSION    ?= ${IMG_VERSION}
DOCKER_LABEL_REVISION   ?= ${GIT_COMMIT}
DOCKER_LABEL_BUILD_DATE ?= $(shell date -u "+%Y-%m-%dT%H:%M:%SZ")
# Docker networking flags for the database container.
# The problem is as follows: On a local MacOS machine we want to expose the port
# of the DB to the native host to enable smooth tooling and unit tests. During
# CI we're already inside a container, hence have to attach the DB container to
# the same network stack as the job. Because the port (-p) syntax cannot be used
# at the same time as the --network container:x flag, we need this variable.
ifeq ($(shell echo $$CI), true)
  DOCKER_NETWORKING_FLAGS = --network container:$$HOSTNAME
else
  DOCKER_NETWORKING_FLAGS = -p 5432:5432
endif

# Base branch to check against for breaking API changes
BASE_BRANCH ?= main

# Path variables
APIPKG_DIR  := pkg/api
BUILD_DIR   := build
SECRETS_DIR := /var/run/secrets
SCRIPTS_DIR := ./ci_scripts

# Tools versions
GOLINTVERSION_HAVE        := $(shell golangci-lint version | sed 's/.*version //' | sed 's/ .*//')
POSTGRES_VERSION          := 16.4
BUFVERSION_REQ		      := 1.45.0
OPAVERSION_REQ            := 0.69.0

GOCMD := GOPRIVATE="github.com/intel-innersource/*" go

# Create the virtualenv with python tools installed
VENV_NAME = venv_mionboarding

# Postgres DB configuration and credentials for testing. This mimics the Aurora
# production environment.
export PGUSER=admin
export PGHOST=localhost
export PGDATABASE=postgres
export PGPORT=5432
export PGPASSWORD=pass
export PGSSLMODE=disable

$(VENV_NAME): requirements.txt
	echo "Creating virtualenv $@"
	python3 -m venv $@;\
	. ./$@/bin/activate; set -u;\
	python3 -m pip install --upgrade pip;\
	python3 -m pip install -r requirements.txt


#-trimpath: Remove all file system paths from the resulting executable.
#-mod=readonly: Ignore vendor directory and report an error if go.mod needs to be updated
#-gcflags="all=-m": Print optimizations applied by the compiler for review and verification against security requirements.
#-gcflags="all=-spectre=all" Enable all available Spectre mitigations
#-ldflags="all=-s -w" remove the symbol and debug info
#-ldflags="all=-X ..." Embedd binary build stamping information.
# Note that arm64 (MacBooks) does not support any spectre mititations.
GOARCH:=$(shell go env GOARCH)
ifeq ($(GOARCH),arm64)
  GOEXTRAFLAGS:=-trimpath -gcflags="all=-spectre= -N -l" -asmflags="all=-spectre=" -ldflags="all=-s -w"
else
  GOEXTRAFLAGS:=-trimpath -gcflags="all=-spectre=all -N -l" -asmflags="all=-spectre=all" -ldflags="all=-s -w"
endif

##### CI Targets #####

all: 
	@# Help: Runs build, lint, test stage
	build lint test

go-build:
	--mount=type=cache,target=/root/.cache/go-build \
    $(GOCMD) build $(GOEXTRAFLAGS) -ldflags="-X 'main.RepoURL=${DOCKER_LABEL_REPO_URL}' -X 'main.Version=${DOCKER_LABEL_VERSION}' \
	-X 'main.Revision=${DOCKER_LABEL_REVISION}' -X 'main.BuildDate=${DOCKER_LABEL_BUILD_DATE}'" \
	-o build/onboardingmgr cmd/onboardingmgr/main.go

hadolint: ## Check Dockerfile with Hadolint
	hadolint Dockerfile

docker-build:  ## build Docker image
	$(GOCMD) mod vendor
	docker build . -f Dockerfile --no-cache \
	    -t maestro-i/${IMG_NAME}:${IMG_VERSION} \
	    --build-arg http_proxy="$(http_proxy)" --build-arg https_proxy="$(https_proxy)" --build-arg no_proxy="$(no_proxy)" \
	    --build-arg HTTP_PROXY="$(HTTP_PROXY)" --build-arg HTTPS_PROXY="$(HTTPS_PROXY)" --build-arg NO_PROXY="$(NO_PROXY)" \
	    --build-arg REPO_URL="$(DOCKER_LABEL_REPO_URL)" --build-arg VERSION="$(DOCKER_LABEL_VERSION)"  \
	    --build-arg REVISION="$(DOCKER_LABEL_REVISION)" --build-arg BUILD_DATE="$(DOCKER_LABEL_BUILD_DATE)" \
	    --build-arg OPA_VERSION=${OPAVERSION_REQ}

docker-push: docker-build ## tag and push Docker image
	docker tag maestro-i/${IMG_NAME}:${IMG_VERSION} ${DOCKER_TAG}:${IMG_VERSION}
	docker push ${DOCKER_TAG}:${IMG_VERSION}

generate-ci: buf-gen

#lint-ci: license go-lint

lint-ci: license go-lint buf-lint hadolint

${BUILD_DIR}:
	mkdir -p ${BUILD_DIR}

${SECRETS_DIR}:
	sudo mkdir -p ${SECRETS_DIR}
	sudo chown ${CURRENT_UID}:${CURRENT_GID} -R ${SECRETS_DIR}

# Maintenance and lint targets
license: $(VENV_NAME) ## Check licensing with the reuse tool.
	. ./$</bin/activate; set -u;\
	reuse --version;\
	reuse --root . lint

buf-lint: ## Lint and format protobuf files
	buf --version
	buf format -d --exit-code
	# TODO: buf lint and breaking disabled for now
	#buf lint
	#buf breaking --against 'https://github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service.git#branch=${BASE_BRANCH}'

buf-gen: ## compile protoc files into code
	buf --version ;\
	buf generate

go-dependency: ## install go dependency tooling
	$(GOCMD) install github.com/bufbuild/buf/cmd/buf@v${BUFVERSION_REQ}

go-lint: ## lint code
	golangci-lint --version
	golangci-lint run --config .golangci.yml

db-start: ## Start the local postgres database. See: db-stop
	if [ -z "`docker ps -aq -f name=^inv-db`" ]; then \
		echo POSTGRES_PASSWORD=$$PGPASSWORD -e POSTGRES_DB=$$PGDATABASE -e POSTGRES_USER=$$PGUSER -d postgres:$(POSTGRES_VERSION); \
		docker run --name inv-db --rm $(DOCKER_NETWORKING_FLAGS) -e POSTGRES_PASSWORD=$$PGPASSWORD -e POSTGRES_DB=$$PGDATABASE -e POSTGRES_USER=$$PGUSER -d postgres:$(POSTGRES_VERSION); \
	fi

db-stop: ## Stop the local postgres database and remove the Docker image. See: db-start
	@if [ -n "`docker ps -aq -f name=^inv-db`" ]; then \
		docker container kill inv-db; \
	fi
	@if [ -n "`docker images -q postgres:$(POSTGRES_VERSION)`" ]; then \
		docker rmi postgres:$(POSTGRES_VERSION); \
	fi

db-shell: ## Run the postgres shell connected to a local database. See: db-start
	docker run -it --network=host -e PGPASSWORD=${PGPASSWORD} --name inv-shell --rm postgres:$(POSTGRES_VERSION) psql -h $$PGHOST -U $$PGUSER -d $$PGDATABASE

go-tidy: ## go mod tidy
	$(GOCMD) mod tidy

go-vendor:
	$(GOCMD) mod vendor

test: ${BUILD_DIR} ${SECRETS_DIR} ## run tests, generate output
	$(MAKE) db-stop
	$(MAKE) db-start
	$(GOCMD) test -race -v -p 1 -coverpkg=$$(go list ./... | grep -v 'api/grpc/onboardingmgr' | grep -v 'pkg/api' | grep -v 'internal/onboardingmgr/onboarding/onboardingmocks' | grep -v 'internal/testing' | grep -v 'cmd/onboardingmgr' | tr '\n' ,) -coverprofile=${BUILD_DIR}/coverage.out -covermode atomic ./... | tee \
		>(go-junit-report -set-exit-code > ${BUILD_DIR}/junit_report.xml)
	gocover-cobertura < ${BUILD_DIR}/coverage.out > ${BUILD_DIR}/coverage.xml
	$(GOCMD) tool cover -html=${BUILD_DIR}/coverage.out -o ${BUILD_DIR}/coverage.html
	$(GOCMD) tool cover -func ${BUILD_DIR}/coverage.out -o ${BUILD_DIR}/function_coverage.log
	$(MAKE) db-stop

coverage:
	@# Help: Runs coverage stage
	@echo "---MAKEFILE COVERAGE---"
	echo $@
	@echo "---END MAKEFILE COVERAGE---"

list: 
	@# Help: displays make targets
	help

clean:
	rm -rf ${BUILD_DIR}
	rm -rf api/proto/
	rm -rf ${APIPKG_DIR}/*.go

help: ## Print help for each target
	@echo inventory make targets
	@echo
	@grep '^[[:alnum:]_-]*:.* ##' $(MAKEFILE_LIST) \
    | sort | awk 'BEGIN {FS=":.* ## "}; {printf "%-25s %s\n", $$1, $$2};'

fuzztest:
	$(MAKE) db-stop
	$(MAKE) db-start
	$(SCRIPTS_DIR)/fuzz_test.sh "internal/handlers/southbound/southbound_utils_test.go" "internal/handlers/controller/reconcilers/instance_test.go" "internal/handlers/southbound/artifact/artifact_service_test.go"
	$(MAKE) db-stop

