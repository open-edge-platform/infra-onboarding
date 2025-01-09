# SPDX-FileCopyrightText: (C) 2024 Intel Corporation
# SPDX-License-Identifier: LicenseRef-Intel

FROM golang:1.23.2-bookworm as build

SHELL ["/bin/bash", "-euo", "pipefail", "-c"]

ARG MAKE_TARGET=build

# Arguments used to stamp the binary
ARG REPO_URL
ARG VERSION
ARG REVISION
ARG BUILD_DATE

COPY Makefile common.mk requirements.txt VERSION go.mod go.sum /go/src/github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/dkam/
COPY cmd/ /go/src/github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/dkam/cmd/
COPY internal/ /go/src/github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/dkam/internal/
COPY pkg/ /go/src/github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/dkam/pkg/
COPY rego/ /go/src/github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/dkam/rego/
COPY vendor/ /go/src/github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/dkam/vendor/

WORKDIR /go/src/github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/dkam

RUN CGO_ENABLED=0 TOOL_VERSION_CHECK=0 \
  DOCKER_LABEL_REPO_URL=${REPO_URL} \
  DOCKER_LABEL_VERSION=${VERSION} \
  DOCKER_LABEL_REVISION=${REVISION} \
  DOCKER_LABEL_BUILD_DATE=${BUILD_DATE} \
  make ${MAKE_TARGET}

FROM debian:12.8

ARG USERNAME=appuser
ARG USER_UID=1000
ARG USER_GID=$USER_UID

RUN groupadd --gid $USER_GID $USERNAME && \
    useradd --uid $USER_UID --gid $USER_GID -m $USERNAME && \
    apt update && apt install -y autoconf automake make gcc \
    m4 git gettext autopoint pkg-config \
    autoconf-archive python3 bison flex \
    gawk efitools sbsigntool gnupg uuid-runtime \
    pigz qemu-utils pax curl unzip

ARG REPO_URL
ARG VERSION
ARG REVISION
ARG BUILD_DATE

LABEL org.opencontainers.image.version=${VERSION} \
      org.opencontainers.image.source=${REPO_URL} \
      org.opencontainers.image.revision=${REVISION} \
      org.opencontainers.image.created=${BUILD_DATE}

WORKDIR /home/$USERNAME

COPY --from=build --chown=$USER_UID:$USER_GID /go/src/github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/dkam/build/dkammgr /usr/local/bin/dkammgr
COPY --from=build --chown=$USER_UID:$USER_GID /go/src/github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/dkam/pkg /home/$USERNAME/pkg
COPY --from=build --chown=$USER_UID:$USER_GID /go/src/github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/dkam/rego/authz.rego /rego/authz.rego

RUN chown -R $USERNAME:$USERNAME /home/$USERNAME
RUN chmod +x pkg/script/build_sign_ipxe.sh
RUN chmod +x pkg/script/hook/secure_hookos.sh
RUN chmod +x pkg/script/hook/cpio_build/build_image_at_DKAM.sh
USER appuser

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:5581/healthz || exit 1

ENTRYPOINT ["dkammgr"]
