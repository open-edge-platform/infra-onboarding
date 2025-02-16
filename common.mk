# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# Makefile Style Guide:
# - Help will be generated from ## comments at end of any target line
# - Use smooth parens $() for variables over curly brackets ${} for consistency
# - Continuation lines (after an \ on previous line) should start with spaces
#   not tabs - this will cause editor highligting to point out editing mistakes
# - When creating targets that run a lint or similar testing tool, print the
#   tool version first so that issues with versions in CI or other remote
#   environments can be caught

# Optionally include tool version checks, not used in Docker builds
ifeq ($(TOOL_VERSION_CHECK), 1)
	include ../version.mk
endif

#### Go Targets ####

# Set GOPRIVATE to deal with private innersource repos
GOCMD := GOPRIVATE="github.com/intel/*,github.com/intel-tiber/*" go

#### Variables ####
SHELL	:= bash -eu -o pipefail
GOARCH	:= $(shell go env GOARCH)
CURRENT_UID := $(shell id -u)
CURRENT_GID := $(shell id -g)

# Path variables
OUT_DIR	   := out
APIPKG_DIR := pkg/api
BIN_DIR    := $(OUT_DIR)/bin
SECRETS_DIR := /var/run/secrets
SCRIPTS_DIR := ./ci_scripts
GOPATH     := $(shell go env GOPATH)
RBAC       := "$(OUT_DIR)/rego/authz.rego"

# Docker variables
DOCKER_ENV              := DOCKER_BUILDKIT=1
DOCKER_REGISTRY         ?= amr-registry.caas.intel.com
DOCKER_REPOSITORY       ?= one-intel-edge/maestro-i
DOCKER_TAG              := ${DOCKER_REGISTRY}/${DOCKER_REPOSITORY}/${IMG_NAME}
DOCKER_LABEL_REPO_URL   ?= $(shell git remote get-url $(shell git remote | head -n 1))
DOCKER_LABEL_VERSION    ?= ${IMG_VERSION}
DOCKER_LABEL_REVISION   ?= ${GIT_COMMIT}
DOCKER_LABEL_BUILD_DATE ?= $(shell date -u "+%Y-%m-%dT%H:%M:%SZ")
DB_CONTAINER_NAME := $(PROJECT_NAME)-db

# Docker networking flags for the database container.
# The problem is as follows: On a local MacOS machine we want to expose the port
# of the DB to the native host to enable smooth tooling and unit tests. During
# CI we're already inside a container, hence have to attach the DB container to
# the same network stack as the job. Because the port (-p) syntax cannot be used
# at the same time as the --network container:x flag, we need this variable.
ifeq ($(shell echo $${CI_CONTAINER:-false}), true)
  DOCKER_NETWORKING_FLAGS = --network container:$$HOSTNAME
else
  DOCKER_NETWORKING_FLAGS = -p 5432:5432
endif

#### Security Config ####
# Security config for Go Builds - see:
#   https://readthedocs.intel.com/SecureCodingStandards/latest/compiler/golang/
# -trimpath: Remove all file system paths from the resulting executable.
# -gcflags="all=-m": Print optimizations applied by the compiler for review and verification against security requirements.
# -gcflags="all=-spectre=all" Enable all available Spectre mitigations
# -ldflags="all=-s -w" remove the symbol and debug info
# -ldflags="all=-X ..." Embed binary build stamping information
ifeq ($(GOARCH),arm64)
	# Note that arm64 (Apple, similar) does not support any spectre mititations.
  GOEXTRAFLAGS := -trimpath -gcflags="all=-spectre= -N -l" -asmflags="all=-spectre=" -ldflags="all=-s -w -X 'main.RepoURL=$(DOCKER_LABEL_REPO_URL)' -X 'main.Version=$(DOCKER_LABEL_VERSION)' -X 'main.Revision=$(DOCKER_LABEL_REVISION)' -X 'main.BuildDate=$(DOCKER_LABEL_BUILD_DATE)'"
else
  GOEXTRAFLAGS := -trimpath -gcflags="all=-spectre=all -N -l" -asmflags="all=-spectre=all" -ldflags="all=-s -w -X 'main.RepoURL=$(DOCKER_LABEL_REPO_URL)' -X 'main.Version=$(DOCKER_LABEL_VERSION)' -X 'main.Revision=$(DOCKER_LABEL_REVISION)' -X 'main.BuildDate=$(DOCKER_LABEL_BUILD_DATE)'"
endif

# Postgres DB configuration and credentials for testing. This mimics the Aurora
# production environment.
export PGUSER=admin
export PGHOST=localhost
export PGDATABASE=postgres
export PGPORT=5432
export PGPASSWORD=pass
export PGSSLMODE=disable

$(OUT_DIR): ## Create out directory
	mkdir -p $(OUT_DIR)

build: $(OUT_DIR) go-build

run: go-build ## Run the resource manager
	$(OUT_DIR)/$(BINARY_NAME)

#### Docker Target ####
docker-build: ## build Docker image
	$(GOCMD) mod vendor
	cp ../common.mk ../version.mk .
	docker build . -f Dockerfile \
		-t maestro-i/$(IMG_NAME):$(IMG_VERSION) \
		--build-arg http_proxy="$(http_proxy)" --build-arg HTTP_PROXY="$(HTTP_PROXY)" \
		--build-arg https_proxy="$(https_proxy)" --build-arg HTTPS_PROXY="$(HTTPS_PROXY)" \
		--build-arg no_proxy="$(no_proxy)" --build-arg NO_PROXY="$(NO_PROXY)" \
		--build-arg REPO_URL="$(DOCKER_LABEL_REPO_URL)" \
		--build-arg VERSION="$(DOCKER_LABEL_VERSION)" \
		--build-arg REVISION="$(DOCKER_LABEL_REVISION)" \
		--build-arg BUILD_DATE="$(DOCKER_LABEL_BUILD_DATE)"
	@rm -rf vendor common.mk version.mk

docker-push: ## tag and push Docker image
	docker tag maestro-i/$(IMG_NAME):$(IMG_VERSION) $(DOCKER_TAG):$(IMG_VERSION)
	docker push $(DOCKER_TAG):$(IMG_VERSION)

#### Python venv Target ####
VENV_NAME	:= venv_$(PROJECT_NAME)

$(VENV_NAME): requirements.txt
	python3 -m venv $@ ;\
  set +u; . ./$@/bin/activate; set -u ;\
  python -m pip install --upgrade pip ;\
  python -m pip install -r requirements.txt

#### GO Targets ####

go-tidy: ## Run go mod tidy
	$(GOCMD) mod tidy

go-lint-fix: $(OUT_DIR)## Apply automated lint/formatting fixes to go files
	golangci-lint run --fix --config .golangci.yml

#### Lint and Validator Targets ####
# https://github.com/koalaman/shellcheck
SH_FILES := $(shell find . -type f \( -name '*.sh' \) -print )
shellcheck: ## lint shell scripts with shellcheck
	shellcheck --version
	shellcheck -x -S style $(SH_FILES)

# https://pypi.org/project/reuse/
license: $(VENV_NAME) ## Check licensing with the reuse tool
	set +u; . ./$</bin/activate; set -u ;\
  reuse --version ;\
  reuse --root . lint

hadolint: ## Check Dockerfile with Hadolint
	hadolint Dockerfile

checksec: go-build #  to check various security properties that are available for executable,like RELRO, STACK CANARY, NX,PIE etc
	$(GOCMD) version -m ${OUT_DIR}/${BINARY_NAME}
	checksec --output=json --file=${OUT_DIR}/${BINARY_NAME}
	checksec --fortify-file=${OUT_DIR}/${BINARY_NAME}

yamllint: $(VENV_NAME) ## lint YAML files
	. ./$</bin/activate; set -u ;\
  yamllint --version ;\
  yamllint -d '{extends: default, rules: {line-length: {max: 99}}, ignore: [$(YAML_IGNORE)]}' -s $(YAML_FILES)

mdlint: ## link MD files
	markdownlint --version ;\
	markdownlint "**/*.md"

go-lint: $(OUT_DIR) ## run go lint
	golangci-lint --version
	golangci-lint run $(LINT_DIRS) --config .golangci.yml

go-test: $(OUT_DIR) ## Run go test and calculate code coverage
ifeq ($(TEST_USE_DB), true)
	$(MAKE) db-stop
	$(MAKE) db-start
endif
	$(GOCMD) test -race -v -p 1 \
	-coverpkg=$(TEST_PKG) -run $(TEST_TARGET) \
	-coverprofile=$(OUT_DIR)/coverage.out \
	-covermode $(TEST_COVER) $(if $(TEST_ARGS),-args $(TEST_ARGS)) \
	| tee >(go-junit-report -set-exit-code > $(OUT_DIR)/report.xml)
	gocover-cobertura $(if $(TEST_IGNORE_FILES),-ignore-files $(TEST_IGNORE_FILES)) < $(OUT_DIR)/coverage.out > $(OUT_DIR)/coverage.xml
	$(GOCMD) tool cover -html=$(OUT_DIR)/coverage.out -o $(OUT_DIR)/coverage.html
	$(GOCMD) tool cover -func=$(OUT_DIR)/coverage.out -o $(OUT_DIR)/function_coverage.log
ifeq ($(TEST_USE_DB), true)
	$(MAKE) db-stop
endif
#### Postgress DB Targets ####

db-start: ## Start the local postgres database. See: db-stop
	if [ -z "`docker ps -aq -f name=^$(DB_CONTAINER_NAME)`" ]; then \
		echo POSTGRES_PASSWORD=$$PGPASSWORD -e POSTGRES_DB=$$PGDATABASE -e POSTGRES_USER=$$PGUSER -d postgres:$(POSTGRES_VERSION); \
		docker run --name $(DB_CONTAINER_NAME) --rm $(DOCKER_NETWORKING_FLAGS) -e POSTGRES_PASSWORD=$$PGPASSWORD -e POSTGRES_DB=$$PGDATABASE -e POSTGRES_USER=$$PGUSER -d postgres:$(POSTGRES_VERSION); \
	fi

db-stop: ## Stop the local postgres database. See: db-start
	@if [ -n "`docker ps -aq -f name=^$(DB_CONTAINER_NAME)`" ]; then \
		docker container kill $(DB_CONTAINER_NAME); \
	fi

db-shell: ## Run the postgres shell connected to a local database. See: db-start
	docker run -it --network=host -e PGPASSWORD=${PGPASSWORD} --name inv-shell --rm postgres:$(POSTGRES_VERSION) psql -h $$PGHOST -U $$PGUSER -d $$PGDATABASE

#### Buf protobuf code generation tooling ###

common-buf-update: $(VENV_NAME) ## update buf modules
	set +u; . ./$</bin/activate; set -u ;\
  buf --version ;\
  pushd api; buf dep update; popd ;\
  buf build

common-buf-gen: ## Compile protoc files into code
	buf --version ;\
	buf generate

common-buf-lint: $(VENV_NAME) ## Lint and format protobuf files
	buf --version
	buf format -d --exit-code
	buf lint

#### Clean Targets ###
clean: ## Delete build and vendor directories
	rm -rf $(OUT_DIR) vendor $(DIR_TO_CLEAN)

clean-venv: ## Delete Python venv
	rm -rf "$(VENV_NAME)"

clean-all: clean clean-venv ## Delete all built artifacts and downloaded tool


#### Help Target ####
help: ## Print help for each target
	@echo $(PROJECT_NAME) make targets
	@echo "Target               Makefile:Line    Description"
	@echo "-------------------- ---------------- -----------------------------------------"
	@grep -H -n '^[[:alnum:]_-]*:.* ##' $(MAKEFILE_LIST) \
    | sort -t ":" -k 3 \
    | awk 'BEGIN  {FS=":"}; {sub(".* ## ", "", $$4)}; {printf "%-20s %-16s %s\n", $$3, $$1 ":" $$2, $$4};'
