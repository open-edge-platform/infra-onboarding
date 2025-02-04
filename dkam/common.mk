# common.mk - common targets for Fleet Management repos

# SPDX-FileCopyrightText: (C) 2024 Intel Corporation
# SPDX-License-Identifier: LicenseRef-Intel

# Makefile Style Guide:
# - Help will be generated from ## comments at end of any target line
# - Use smooth parens $() for variables over curly brackets ${} for consistency
# - Continuation lines (after an \ on previous line) should start with spaces
#   not tabs - this will cause editor highligting to point out editing mistakes
# - When creating targets that run a lint or similar testing tool, print the
#   tool version first so that issues with versions in CI or other remote
#   environments can be caught

#### Go Targets ####

# Set GOPRIVATE to deal with private innersource repos
GOCMD := GOPRIVATE="github.com/intel-innersource/*" go

# optionally include tool version checks, not used in Docker builds
TOOL_VERSION_CHECK ?= 1
ifeq ($(TOOL_VERSION_CHECK), 1)
	include version.mk
endif

go-tidy: ## Run go mod tidy
	$(GOCMD) mod tidy

go-vendor: ## Run go mod vendor
	$(GOCMD) mod vendor

#go-lint: ## run go lint
#	golangci-lint linters

# TODO: add correct golintci configuration and uncomment
#go-lint: $(OUT_DIR) ## Lint go code with golangci-lint
#	golangci-lint --version
#	golangci-lint run --config .golangci.yml

#go-lint-fix: ## Apply automated lint/formatting fixes to go files
#	golangci-lint run --fix --config .golangci.yml

#### Docker Config & Targets ####
# Docker variables
DOCKER_ENV              := DOCKER_BUILDKIT=1
DOCKER_REGISTRY         ?= amr-registry.caas.intel.com
DOCKER_REPOSITORY       ?= one-intel-edge/maestro-i
DOCKER_TAG              := $(DOCKER_REGISTRY)/$(DOCKER_REPOSITORY)/$(IMG_NAME)
DOCKER_LABEL_REPO_URL   ?= $(shell git remote get-url $(shell git remote | head -n 1))
DOCKER_LABEL_VERSION    ?= $(IMG_VERSION)
DOCKER_LABEL_REVISION   ?= $(GIT_COMMIT)
DOCKER_LABEL_BUILD_DATE ?= $(shell date -u "+%Y-%m-%dT%H:%M:%SZ")

DB_CONTAINER_NAME 		?= inv-db

# Docker networking flags for the database container.
# The problem is as follows: On a local MacOS machine we want to expose the port
# of the DB to the native host to enable smooth tooling and unit tests. During
# CI we're already inside a container, hence have to attach the DB container to
# the same network stack as the job. Because the port (-p) syntax cannot be used
# at the same time as the --network container:x flag, we need this variable.
ifeq ($(shell set +u; echo $$CI), true)
  DOCKER_NETWORKING_FLAGS = --network container:$$HOSTNAME
else
  DOCKER_NETWORKING_FLAGS = -p 5432:5432
endif

docker-build: ## build Docker image
	$(GOCMD) mod vendor
	docker build . -f Dockerfile \
    -t maestro-i/$(IMG_NAME):$(IMG_VERSION) \
    --build-arg http_proxy="$(http_proxy)" --build-arg HTTP_PROXY="$(HTTP_PROXY)" \
    --build-arg https_proxy="$(https_proxy)" --build-arg HTTPS_PROXY="$(HTTPS_PROXY)" \
    --build-arg no_proxy="$(no_proxy)" --build-arg NO_PROXY="$(NO_PROXY)" \
    --build-arg REPO_URL="$(DOCKER_LABEL_REPO_URL)" \
    --build-arg VERSION="$(DOCKER_LABEL_VERSION)" \
    --build-arg REVISION="$(DOCKER_LABEL_REVISION)" \
    --build-arg BUILD_DATE="$(DOCKER_LABEL_BUILD_DATE)"

docker-push: docker-build ## tag and push Docker image
	docker tag maestro-i/$(IMG_NAME):$(IMG_VERSION) $(DOCKER_TAG):$(IMG_VERSION)
	docker push $(DOCKER_TAG):$(IMG_VERSION)

#### Database Config & Targets (for unit testing only) ####
PGHOST     := localhost
PGDATABASE := postgres
PGPORT     := 5432
PGSSLMODE  := disable
PGUSER     := admin
PGPASSWORD := pass

db-start: ## Start the local postgres database. See: db-stop
	if [ -z "`docker ps -aq -f name=^$(DB_CONTAINER_NAME)`" ]; then \
    docker run --name $(DB_CONTAINER_NAME) --rm $(DOCKER_NETWORKING_FLAGS) \
      -e POSTGRES_DB=$(PGDATABASE) \
      -e POSTGRES_USER=$(PGUSER) \
      -e POSTGRES_PASSWORD=$(PGPASSWORD) \
      -d postgres:$(POSTGRES_VERSION) ;\
  fi

db-stop: ## Stop the local postgres database. See: db-start
	@if [ -n "`docker ps -aq -f name=^$(DB_CONTAINER_NAME)`" ]; then \
    docker container kill $(DB_CONTAINER_NAME); \
  fi

db-shell: ## Run the postgres shell connected to a local database. See: db-start
	docker run -it --name inv-shell --rm --network=host \
    -e PGPASSWORD=$(PGPASSWORD) \
    postgres:$(POSTGRES_VERSION) \
    psql -h $(PGHOST) -U $(PGUSER) -d $(PGDATABASE)

#### Security Config & Targets ####
# Security config for Go Builds - see:
#   https://readthedocs.intel.com/SecureCodingStandards/latest/compiler/golang/
# -trimpath: Remove all file system paths from the resulting executable.
# -gcflags="all=-m": Print optimizations applied by the compiler for review and verification against security requirements.
# -gcflags="all=-spectre=all" Enable all available Spectre mitigations
# -ldflags="all=-s -w" remove the symbol and debug info
# -ldflags="all=-X ..." Embed binary build stamping information
GOARCH := $(shell go env GOARCH)
ifeq ($(GOARCH),arm64)
	# Note that arm64 (Apple, similar) does not support any spectre mititations.
  GOEXTRAFLAGS := -trimpath -gcflags="all=-spectre= -N -l" -asmflags="all=-spectre=" -ldflags="all=-s -w -X 'main.RepoURL=$(DOCKER_LABEL_REPO_URL)' -X 'main.Version=$(DOCKER_LABEL_VERSION)' -X 'main.Revision=$(DOCKER_LABEL_REVISION)' -X 'main.BuildDate=$(DOCKER_LABEL_BUILD_DATE)'"
else
  GOEXTRAFLAGS := -trimpath -gcflags="all=-spectre=all -N -l" -asmflags="all=-spectre=all" -ldflags="all=-s -w -X 'main.RepoURL=$(DOCKER_LABEL_REPO_URL)' -X 'main.Version=$(DOCKER_LABEL_VERSION)' -X 'main.Revision=$(DOCKER_LABEL_REVISION)' -X 'main.BuildDate=$(DOCKER_LABEL_BUILD_DATE)'"
endif

# https://github.com/slimm609/checksec.sh
# checks various security properties on executoables, such as RELRO, STACK CANARY, NX, PIE, etc.
checksec: go-build ## Check security properties on executables
	checksec --output=json --file=$(BUILD_DIR)/$(BINARY_NAME)
	checksec --fortify-file=$(BUILD_DIR)/$(BINARY_NAME)

#### Python venv Target ####
VENV_NAME	:= venv_$(PROJECT_NAME)

$(VENV_NAME): requirements.txt
	python3 -m venv $@ ;\
  set +u; . ./$@/bin/activate; set -u ;\
  python -m pip install --upgrade pip ;\
  python -m pip install -r requirements.txt

#### Buf protobuf code generation tooling ###

APIPKG_DIR ?= pkg/api

buf-update: $(VENV_NAME) ## update buf modules
	set +u; . ./$</bin/activate; set -u ;\
  buf --version ;\
  pushd api; buf mod update; popd ;\
  buf build

buf-generate: $(VENV_NAME) ## compile protobuf files in api into code
	set +u; . ./$</bin/activate; set -u ;\
  buf --version ;\
  buf generate

buf-lint: $(VENV_NAME) ## Lint and format protobuf files
	buf --version
	buf format -d --exit-code
	buf lint

buf-lint-fix: $(VENV_NAME) ## Lint and when possible fix protobuf files
	buf --version
	buf format -d -w
	buf lint
	buf breaking --against 'https://github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service.git#branch=${BASE_BRANCH}

#### Lint Targets ####
# https://github.com/hadolint/hadolint/
hadolint: ## Check Dockerfile with Hadolint
	hadolint --version
	hadolint Dockerfile

# https://github.com/koalaman/shellcheck
SH_FILES := $(shell find . -type f \( -name '*.sh' \) -print )
shellcheck: ## lint shell scripts with shellcheck TODO: fix issues and add SH_FILES
	echo $(SH_FILES)
	shellcheck --version
	shellcheck -x -S style $(SH_FILES)

# https://pypi.org/project/reuse/
license: $(VENV_NAME) ## Check licensing with the reuse tool
	set +u; . ./$</bin/activate; set -u ;\
  reuse --version ;\
  reuse --root . lint

# https://pypi.org/project/yamllint/
YAML_FILES := $(shell find . -type f \( -name '*.yaml' -o -name '*.yml' \) -print )
yamllint: $(VENV_NAME) ## lint YAML files
	set +u; . ./$</bin/activate; set -u ;\
  yamllint --version ;\
  yamllint -d '{extends: default, rules: {line-length: {max: 99}}, ignore: [vendor, .github/workflows, $(VENV_NAME)]}' -s $(YAML_FILES)

#### Help Target ####
help: ## Print help for each target
	@echo $(PROJECT_NAME) make targets
	@echo "Target               Makefile:Line    Description"
	@echo "-------------------- ---------------- -----------------------------------------"
	@grep -H -n '^[[:alnum:]_-]*:.* ##' $(MAKEFILE_LIST) \
    | sort -t ":" -k 3 \
    | awk 'BEGIN  {FS=":"}; {sub(".* ## ", "", $$4)}; {printf "%-20s %-16s %s\n", $$3, $$1 ":" $$2, $$4};'
