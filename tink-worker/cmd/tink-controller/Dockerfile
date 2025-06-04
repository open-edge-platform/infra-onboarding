# SPDX-FileCopyrightText: 2025 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

FROM alpine:3.15

ARG TARGETOS
ARG TARGETARCH

RUN apk add --no-cache --update --upgrade ca-certificates

COPY bin/tink-controller-${TARGETOS}-${TARGETARCH} /usr/bin/tink-controller

ENTRYPOINT ["/usr/bin/tink-controller"]
