# HookOS Makefile

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

.DEFAULT_GOAL := help
.PHONY: build test lint license help generate shellcheck

##### Variables #####
# Project variables
PROJECT_NAME := hook-os
BINARY_NAME  := hook-os

# Code versions, tags, and so on
VERSION                     := $(shell cat VERSION)
VERSION_MAJOR               := $(shell cut -c 1 VERSION)
IMG_NAME                    := provisioning-hook-os
IMG_VERSION                 ?= $(VERSION)
GIT_COMMIT                  ?= $(shell git rev-parse HEAD)
BRANCH_NAME                 ?= $(shell git rev-parse --abbrev-ref HEAD)
OCI_REPOSITORY              := edge-orch/en/files/${IMG_NAME}
HOOK_KERNEL_REPOSITORY      := edge-orch/edge-node/hook-kernel
HOOK_KERNEL_POINT_RELEASE   :=228


# Include shared makefile
include ../common.mk

OCI_PATH	:= ${OCI_REGISTRY}/${OCI_REPOSITORY}
COMPONENTS	:= device_discovery fluent-bit hook_dind
.PHONY: $(COMPONENTS)

all:
	@# Help: Runs build, lint, test stages
	build lint test

$(COMPONENTS):
	cd $@ && \
	chmod +x build.sh && bash build.sh
	
components: $(COMPONENTS)

configure:
	echo 'http_proxy=$(http_proxy)' > config
	echo 'https_proxy=$(http_proxy)' >> config
	echo 'ftp_proxy=$(ftp_proxy)' >> config
	echo 'socks_proxy=$(socks_proxy)' >> config
	echo 'no_proxy=$(no_proxy)' >> config
	echo 'keycloak_url=keycloak.example.com' >> config

certs:
	mkdir -p hook/files/idp
	echo "" >> hook/files/idp/ca.pem
	echo "" >> hook/files/idp/server_cert.pem

kernel:
	cd hook/ && \
	HOOK_KERNEL_OCI_BASE=${RELEASE_SVC_URL}/${HOOK_KERNEL_REPOSITORY} \
	FORCE_BUILD_KERNEL="yes" \
	KERNEL_POINT_RELEASE=${HOOK_KERNEL_POINT_RELEASE} \
	bash ./build.sh kernel hook-default-amd64

binaries:
	@# Help: Runs build binaries stage
	@echo "---MAKEFILE BUILD---"
	chmod +x build_hookos.sh && \
	HOOK_KERNEL_OCI_BASE=${RELEASE_SVC_URL}/${HOOK_KERNEL_REPOSITORY} \
	KERNEL_POINT_RELEASE=${HOOK_KERNEL_POINT_RELEASE} \
	bash build_hookos.sh
	git stash -- hook/linuxkit-templates/hook.template.yaml
	@echo "---END MAKEFILE Build---"

build: configure certs components
	@# Help: Runs build stage
	@# Builds HookOS binaries. If the kernel image is not available in Release Service or locally, it first builds the kernel image.
	@bash -c '$(MAKE) binaries 2> >(tee binarybuilderr.log >&2) || { \
		if grep -q "Error 7" binarybuilderr.log; then \
			echo "Found Error 7 in binarybuilderr.log, running `make kernel`..."; \
			$(MAKE) kernel && $(MAKE) binaries; \
		else \
			echo "Exit code is not 7"; \
			exit 1; \
		fi; \
	}'

SH_FILES := $(shell git ls-files | grep '\.sh$$' | grep -v '^hook/')
lint: license shellcheck mdlint
	@# Help: Runs lint stage
	@echo "---MAKEFILE LINT---"
	echo $@
	@echo "---END MAKEFILE LINT---"

test:
	@# Help: Runs test stage
	@echo "---MAKEFILE TEST---"
	echo $@
	@echo "---END MAKEFILE TEST---"

publish-binaries-ci:
	@echo "Publishing Hook OS binary to Production Release Service."
	cd $(OUT_DIR) && \
	aws ecr create-repository --region us-west-2 --repository-name $(OCI_REPOSITORY) || true && \
	oras push $(OCI_PATH):$(IMG_VERSION) --artifact-type application/vnd.intel.hookos.file hook_x86_64.tar.gz
	@echo "---END MAKEFILE TEST---"

push-kernel-ci:
	@# Help: Pushes kernel image to Release Service
	@# Pushes the Hook OS kernel image to the Release Service only if the image tag isn't already present in the Release Service.
	@echo "Publishing Hook OS kernel to Production Release Service."
	@image_name=$(shell cat hook/hook.hook-default-amd64.yaml | yq .kernel.image) && \
	tag_name=$(shell cat hook/hook.hook-default-amd64.yaml | yq .kernel.image | cut -d':' -f2) && \
	echo "Kernel Tag:" $$tag_name && \
	docker pull $$image_name || (docker tag $$image_name ${OCI_REGISTRY}/${HOOK_KERNEL_REPOSITORY}:$$tag_name && \
	docker push ${OCI_REGISTRY}/${HOOK_KERNEL_REPOSITORY}:$$tag_name)

artifact-publish: publish-binaries-ci push-kernel-ci

coverage:
	@# Help: Runs coverage stage
	@echo "---MAKEFILE COVERAGE---"
	echo $@
	@echo "---END MAKEFILE COVERAGE---"

list:
	@# Help: displays make targets
	help

dependency-check:
	@# Help: Runs dependency-check stage
	@echo "---MAKEFILE TEST---"
	echo $@
	@echo "---END MAKEFILE TEST---"

helm-build:
	@# Help: Runs helm-build stage
	@echo "---MAKEFILE TEST---"
	echo $@
	@echo "---END MAKEFILE TEST---"

docker-build:
	@# Help: Runs docker-build stage
	@echo "---MAKEFILE TEST---"
	echo $@
	@echo "---END MAKEFILE TEST---"
