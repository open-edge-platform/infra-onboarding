# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# NOTICE: This file has been modified by Intel Corporation.
# Original file can be found at https://github.com/tinkerbell/actions.

FROM golang:1.24.1-alpine AS image2disk
RUN apk add --no-cache git ca-certificates gcc linux-headers musl-dev
COPY . /src/image2disk
COPY pkg /pkg/
WORKDIR /src/image2disk

RUN CGO_ENABLED=1 GOOS=linux go build -a -ldflags "-linkmode external -extldflags '-static' -s -w" -o image2disk

FROM alpine:3.21.3
RUN apk add --no-cache lsblk
# Add Certificates into the image, for anything that does HTTPS calls
COPY --from=image2disk /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=image2disk /src/image2disk/image2disk /usr/bin/image2disk
# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:50054/healthz || exit 1
ENTRYPOINT ["sh", "-c", "nsenter -t 1 -n /usr/bin/image2disk"]
