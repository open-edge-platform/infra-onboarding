# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.24.4-bookworm as builder

ENV GO111MODULE=on
ARG MAKE_TARGET=go-build

# Arguments used to stamp the binary
ARG REPO_URL
ARG VERSION
ARG REVISION
ARG BUILD_DATE

COPY common.mk version.mk common_go.mk /go/src/github.com/open-edge-platform/infra-onboarding/
COPY Makefile requirements.txt VERSION go.mod go.sum /go/src/github.com/open-edge-platform/infra-onboarding/onboarding-manager/
COPY cmd/ /go/src/github.com/open-edge-platform/infra-onboarding/onboarding-manager/cmd/
COPY internal/ /go/src/github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/
COPY pkg/ /go/src/github.com/open-edge-platform/infra-onboarding/onboarding-manager/pkg/
COPY rego/ /go/src/github.com/open-edge-platform/infra-onboarding/onboarding-manager/rego/
COPY vendor/ /go/src/github.com/open-edge-platform/infra-onboarding/onboarding-manager/vendor/

WORKDIR /go/src/github.com/open-edge-platform/infra-onboarding/onboarding-manager


RUN CGO_ENABLED=0 LABEL_REPO_URL=${REPO_URL} LABEL_VERSION=${VERSION} LABEL_REVISION=${REVISION} LABEL_BUILD_DATE=${BUILD_DATE} make ${MAKE_TARGET}


FROM gcr.io/distroless/static-debian12:nonroot

USER nobody

ARG REPO_URL
ARG VERSION
ARG REVISION
ARG BUILD_DATE

LABEL org.opencontainers.image.version=${VERSION} \
      org.opencontainers.image.source=${REPO_URL} \
      org.opencontainers.image.revision=${REVISION} \
      org.opencontainers.image.created=${BUILD_DATE}

COPY --from=builder --chown=nobody:nobody /go/src/github.com/open-edge-platform/infra-onboarding/onboarding-manager/out/onboardingmgr /onboardingmgr

COPY --from=builder  --chown=nobody:nobody /go/src/github.com/open-edge-platform/infra-onboarding/onboarding-manager/rego/authz.rego /rego/authz.rego

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:50054/healthz || exit 1

ENTRYPOINT ["/onboardingmgr"]
