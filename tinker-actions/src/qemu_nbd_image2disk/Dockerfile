# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.24.8-alpine3.22 AS qemu-nbd-image2disk
RUN apk add --no-cache git ca-certificates gcc linux-headers musl-dev
COPY . /src/qemu-nbd-image2disk
COPY pkg /pkg/
WORKDIR /src/qemu-nbd-image2disk
RUN CGO_ENABLED=1 GOOS=linux go build -a -ldflags "-linkmode external -extldflags '-static' -s -w" -o qemu-nbd-image2disk

FROM alpine:3.21.3
RUN apk add --no-cache lsblk qemu-img
# Add Certificates into the image, for anything that does HTTPS calls
COPY --from=qemu-nbd-image2disk /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=qemu-nbd-image2disk /src/qemu-nbd-image2disk/qemu-nbd-image2disk /usr/bin/qemu-nbd-image2disk
# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:50054/healthz || exit 1
ENTRYPOINT ["sh", "-c", "nsenter -t 1 -n /usr/bin/qemu-nbd-image2disk"]
