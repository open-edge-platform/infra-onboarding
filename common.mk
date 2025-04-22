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
RBAC       := "$(OUT_DIR)/rego/authz.rego"

# Docker variables
DOCKER_ENV              := DOCKER_BUILDKIT=1
OCI_REGISTRY            ?= 080137407410.dkr.ecr.us-west-2.amazonaws.com
OCI_REPOSITORY          ?= edge-orch
DOCKER_SECTION          := infra
DOCKER_REGISTRY         ?= $(OCI_REGISTRY)
DOCKER_REPOSITORY       ?= $(OCI_REPOSITORY)
DOCKER_TAG              := $(DOCKER_REGISTRY)/$(DOCKER_REPOSITORY)/$(DOCKER_SECTION)/$(DOCKER_IMG_NAME):$(VERSION)
DOCKER_TAG_BRANCH       := $(DOCKER_REGISTRY)/$(DOCKER_REPOSITORY)/$(DOCKER_SECTION)/$(DOCKER_IMG_NAME):$(DOCKER_VERSION)

# release service
RELEASE_SVC_URL         ?= registry-rs.edgeorchestration.intel.com
# Decides if we shall push image tagged with the branch name or not.
DOCKER_TAG_BRANCH_PUSH	?= true
LABEL_REPO_URL          ?= $(shell git remote get-url $(shell git remote | head -n 1))
LABEL_VERSION           ?= $(VERSION)
LABEL_REVISION          ?= $(GIT_COMMIT)
LABEL_BUILD_DATE        ?= $(shell date -u "+%Y-%m-%dT%H:%M:%SZ")

ifeq ($(GO_CHECK), 1)
	include ../common_go.mk
endif

$(OUT_DIR): ## Create out directory
	mkdir -p $(OUT_DIR)

#### Python venv Target ####
VENV_NAME	:= venv_$(PROJECT_NAME)

$(VENV_NAME): requirements.txt
	python3 -m venv $@ ;\
  set +u; . ./$@/bin/activate; set -u ;\
  python -m pip install --upgrade pip ;\
  python -m pip install -r requirements.txt

#### Lint and Validator Targets ####
# https://github.com/koalaman/shellcheck
SH_FILES := $(shell git ls-files | grep '\.sh$$')
shellcheck: ## lint shell scripts with shellcheck
	shellcheck --version;
	@if [ -n "$(SH_FILES)" ]; then \
		shellcheck -x -S style $(SH_FILES); \
	else \
		echo "No shell scripts found to lint."; \
	fi

# https://pypi.org/project/reuse/
license: $(VENV_NAME) ## Check licensing with the reuse tool
	set +u; . ./$</bin/activate; set -u ;\
  reuse --version ;\
  reuse --root . lint

HADOLINT_FILES := $(shell find . -type f \( -name '*Dockerfile*' \) -print )
hadolint: ## Check Dockerfile with Hadolint
	hadolint $(HADOLINT_FILES)

yamllint: $(VENV_NAME) ## lint YAML files
	. ./$</bin/activate; set -u ;\
	yamllint --version ;\
	if [ -n "$(YAML_FILES)" ]; then \
	    yamllint -d '{extends: default, rules: {line-length: {max: 99}}, ignore: [$(YAML_IGNORE)]}' -s $(YAML_FILES); \
	else \
	    echo "No YAML files found to lint."; \
	fi

mdlint: ## link MD files
	markdownlint --version ;\
	markdownlint "**/*.md" -c ../.markdownlint.yml


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
