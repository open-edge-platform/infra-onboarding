# SPDX-FileCopyrightText: (C) 2024 Intel Corporation
# SPDX-License-Identifier: LicenseRef-Intel

FROM golang:1.21.13-bookworm as builder

ARG MAKE_TARGET=go-build

# Arguments used to stamp the binary
ARG REPO_URL
ARG VERSION
ARG REVISION
ARG BUILD_DATE

COPY Makefile requirements.txt VERSION go.mod go.sum /go/src/github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/
COPY cmd/ /go/src/github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/cmd/
COPY internal/ /go/src/github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/
COPY pkg/ /go/src/github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/pkg/
COPY rego/ /go/src/github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/rego/
COPY vendor/ /go/src/github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/vendor/

WORKDIR /go/src/github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service


RUN CGO_ENABLED=0 DOCKER_LABEL_REPO_URL=${REPO_URL} DOCKER_LABEL_VERSION=${VERSION} DOCKER_LABEL_REVISION=${REVISION} DOCKER_LABEL_BUILD_DATE=${BUILD_DATE} make ${MAKE_TARGET}



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

COPY --from=builder --chown=nobody:nobody /go/src/github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/build/onboardingmgr /onboardingmgr

COPY --from=builder  --chown=nobody:nobody /go/src/github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/rego/authz.rego /rego/authz.rego


ENTRYPOINT ["/onboardingmgr"]
