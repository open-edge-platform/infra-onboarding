# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.24.1-bookworm as build

SHELL ["/bin/bash", "-euo", "pipefail", "-c"]

ENV GO111MODULE=on
ARG MAKE_TARGET=build

# Arguments used to stamp the binary
ARG REPO_URL
ARG VERSION
ARG REVISION
ARG BUILD_DATE

COPY common.mk version.mk common_go.mk /go/src/github.com/open-edge-platform/infra-onboarding/
COPY Makefile requirements.txt VERSION go.mod go.sum /go/src/github.com/open-edge-platform/infra-onboarding/dkam/
COPY cmd/ /go/src/github.com/open-edge-platform/infra-onboarding/dkam/cmd/
COPY internal/ /go/src/github.com/open-edge-platform/infra-onboarding/dkam/internal/
COPY pkg/ /go/src/github.com/open-edge-platform/infra-onboarding/dkam/pkg/
COPY vendor/ /go/src/github.com/open-edge-platform/infra-onboarding/dkam/vendor/

WORKDIR /go/src/github.com/open-edge-platform/infra-onboarding/dkam

RUN CGO_ENABLED=0 LABEL_REPO_URL=${REPO_URL} LABEL_VERSION=${VERSION} LABEL_REVISION=${REVISION} LABEL_BUILD_DATE=${BUILD_DATE} make ${MAKE_TARGET}

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
    pax curl unzip

ARG REPO_URL
ARG VERSION
ARG REVISION
ARG BUILD_DATE

LABEL org.opencontainers.image.version=${VERSION} \
      org.opencontainers.image.source=${REPO_URL} \
      org.opencontainers.image.revision=${REVISION} \
      org.opencontainers.image.created=${BUILD_DATE}

WORKDIR /home/$USERNAME

COPY --from=build --chown=$USER_UID:$USER_GID /go/src/github.com/open-edge-platform/infra-onboarding/dkam/out/dkammgr /usr/local/bin/dkammgr
COPY --from=build --chown=$USER_UID:$USER_GID /go/src/github.com/open-edge-platform/infra-onboarding/dkam/pkg /home/$USERNAME/pkg

RUN chown -R $USERNAME:$USERNAME /home/$USERNAME
RUN chmod +x pkg/script/build_sign_ipxe.sh
RUN chmod +x pkg/script/hook/secure_hookos.sh
RUN chmod +x pkg/script/hook/cpio_build/build_image_at_DKAM.sh
USER appuser

ENTRYPOINT ["dkammgr"]
