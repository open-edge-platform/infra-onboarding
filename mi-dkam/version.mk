# version.mk - check versions of tools for FM repos

# SPDX-FileCopyrightText: (C) 2024 Intel Corporation
# SPDX-License-Identifier: LicenseRef-Intel

# Tools versions
GOLINTVERSION_HAVE             := $(shell golangci-lint version | sed 's/.*version //' | sed 's/ .*//')
GOLINTVERSION_REQ              := 1.54.2
GOJUNITREPORTVERSION_HAVE      := $(shell go-junit-report -version | sed s/.*" v"// | sed 's/ .*//')
GOJUNITREPORTVERSION_REQ       := 1.0.0
PROTOCGENDOCVERSION_HAVE       := $(shell protoc-gen-doc --version | sed s/.*"version "// | sed 's/ .*//')
PROTOCGENDOCVERSION_REQ        := 1.5.1
BUFVERSION_HAVE                := $(shell buf --version)
BUFVERSION_REQ                 := 1.27.1
OPAVERSION_HAVE                := $(shell opa version | grep "Version:" | grep -v "Go" | sed 's/.*Version: //')
OPAVERSION_REQ                 := 0.49.0
GOVERSION_REQ                  := 1.21.9
GOVERSION_HAVE                 := $(shell go version | sed 's/.*version go//' | sed 's/ .*//')
MOCKGENVERSION_HAVE            := $(shell mockgen -version | sed s/.*"v"// | sed 's/ .*//')
MOCKGENVERSION_REQ             := 1.6.0
# No version reported
GOCOBERTURAVERSION_REQ         := 1.2.0
PROTOCGENENTVERSION_REQ        := 0.3.0
POSTGRES_VERSION               := 14.9

dependency-check: ## check versions of installed tools against recommended versions
	@(echo "$(GOVERSION_HAVE)" | grep "$(GOVERSION_REQ)" > /dev/null) || \
    (echo "WARNING: Recommended version of go: $(GOVERSION_REQ) is not available - installed version: $(GOVERSION_HAVE)" && exit 1)
	@(echo "$(GOLINTVERSION_HAVE)" | grep "$(GOLINTVERSION_REQ)" > /dev/null) || \
    (echo "WARNING: Recommended version of golangci-lint: $(GOLINTVERSION_REQ) is not available - installed version: $(GOLINTVERSION_HAVE)" && exit 1)
	@(echo "$(GOJUNITREPORTVERSION_HAVE)" | grep "$(GOJUNITREPORTVERSION_REQ)" > /dev/null) || \
    (echo "WARNING: Recommended version of go-junit-report: $(GOJUNITREPORTVERSION_REQ) is not available - installed version: $(GOJUNITREPORTVERSION_HAVE)" && exit 1)
	@(echo "$(PROTOCGENDOCVERSION_HAVE)" | grep "$(PROTOCGENDOCVERSION_REQ)" > /dev/null) || \
    (echo "WARNING: Recommended version of protoc-gen-doc: $(PROTOCGENDOCVERSION_REQ) is not available - installed version: $(PROTOCGENDOCVERSION_HAVE)" && exit 1)
	@(echo "$(BUFVERSION_HAVE)" | grep "$(BUFVERSION_REQ)" > /dev/null) || \
    (echo "WARNING: Recommended version of buf: $(BUFVERSION_REQ) is not available - installed version: $(BUFVERSION_HAVE)" && exit 1)
	@(echo "$(OPAVERSION_HAVE)" | grep "$(OPAVERSION_REQ)" > /dev/null) || \
    (echo "WARNING: Recommended version of opa: $(OPAVERSION_REQ) is not available - installed version: $(OPAVERSION_HAVE)" && exit 1)
	@(echo "$(MOCKGENVERSION_HAVE)" | grep "$(MOCKGENVERSION_REQ)" > /dev/null) || \
    (echo "WARNING: Recommended version of mockgen: $(MOCKGENVERSION_REQ) is not available - installed version: $(MOCKGENVERSION_HAVE)" && exit 1)

# Please keep this list sorted
install-go-deps: ## Install Golang related tooling
	$(GOCMD) install entgo.io/contrib/entproto/cmd/protoc-gen-ent@v$(PROTOCGENENTVERSION_REQ)
	$(GOCMD) install github.com/bufbuild/buf/cmd/buf@v$(BUFVERSION_REQ)
	$(GOCMD) install github.com/golang/mock/mockgen@v$(MOCKGENVERSION_REQ)
	$(GOCMD) install github.com/golangci/golangci-lint/cmd/golangci-lint@v$(GOLINTVERSION_REQ)
	$(GOCMD) install github.com/jstemmer/go-junit-report@v$(GOJUNITREPORTVERSION_REQ)
	$(GOCMD) install github.com/pseudomuto/protoc-gen-doc/cmd/protoc-gen-doc@v$(PROTOCGENDOCVERSION_REQ)
