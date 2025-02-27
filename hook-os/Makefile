# HookOS Makefile

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

.DEFAULT_GOAL := help
.PHONY: build test lint license help generate

##### Variables #####
# Project variables
PROJECT_NAME := hook-os
BINARY_NAME  := hook-os

# Code versions, tags, and so on
VERSION       := $(shell cat VERSION)
VERSION_MAJOR := $(shell cut -c 1 VERSION)
IMG_NAME      := provisioning-hook-os
IMG_VERSION   ?= $(VERSION)
GIT_COMMIT    ?= $(shell git rev-parse HEAD)
BRANCH_NAME   ?= $(shell git rev-parse --abbrev-ref HEAD)
OCI_REPOSITORY := edge-orch/en/files/${IMG_NAME}
HOOK_KERNEL_REPOSITORY  := open-registry.espdprod.infra-host.com/edge-orch/edge-node/hook-kernel
HOOK_KERNEL_VERSION 	:= 5.10.228-e0637f99-dm-xz


# Include shared makefile
include ../common.mk

OCI_PATH	:= ${OCI_REGISTRY}/${OCI_REPOSITORY}
COMPONENTS	:= device_discovery fluent-bit caddy hook_dind
.PHONY: $(COMPONENTS)

all:
	@# Help: Runs build, lint, test stages
	build lint test

$(COMPONENTS):
	cd $@ && \
	chmod +x build.sh && bash build.sh
	
pull-kernel:
	docker pull ${HOOK_KERNEL_REPOSITORY}:${HOOK_KERNEL_VERSION}
	docker tag ${HOOK_KERNEL_REPOSITORY}:${HOOK_KERNEL_VERSION} quay.io/tinkerbell/hook-kernel:5.10.228-e0637f99

components: $(COMPONENTS) pull-kernel

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

build: configure certs components
	@# Help: Runs build stage
	@echo "---MAKEFILE BUILD---"
	@echo "--Updated step below--"
	chmod +x build_hookos.sh && bash build_hookos.sh
	git stash -- hook/linuxkit-templates/hook.template.yaml
	@echo "---END MAKEFILE Build---"

lint: license mdlint
	@# Help: Runs lint stage
	@echo "---MAKEFILE LINT---"
	echo $@
	@echo "---END MAKEFILE LINT---"

test:
	@# Help: Runs test stage
	@echo "---MAKEFILE TEST---"
	echo $@
	@echo "---END MAKEFILE TEST---"

artifact-publish:
	@echo "Publishing Hook OS binary to Production Release Service."
	cd $(OUT_DIR) && \
	aws ecr create-repository --region us-west-2 --repository-name $(OCI_REPOSITORY) || true && \
	oras push $(OCI_PATH):$(IMG_VERSION) --artifact-type application/vnd.intel.hookos.file hook_x86_64.tar.gz
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
