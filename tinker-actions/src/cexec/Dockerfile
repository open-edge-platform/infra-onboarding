# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# NOTICE: This file has been modified by Intel Corporation.
# Original file can be found at https://github.com/tinkerbell/actions.

# Build cexec
FROM golang:1.23.2-alpine3.20 AS cexec

RUN apk add --no-cache git ca-certificates gcc linux-headers musl-dev
COPY . /src/cexec
COPY pkg /pkg
WORKDIR /src/cexec
RUN --mount=type=cache,sharing=locked,id=gomod,target=/go/pkg/mod/cache \
    --mount=type=cache,sharing=locked,id=goroot,target=/root/.cache/go-build \
    CGO_ENABLED=1 GOOS=linux go build -a -ldflags "-linkmode external -extldflags '-static' -s -w" -o cexec

# Build final image
FROM alpine:3.20.3
RUN apk add --no-cache lsblk
COPY --from=cexec /src/cexec/cexec /usr/bin/cexec

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:50054/healthz || exit 1

ENTRYPOINT ["sh", "-c", "nsenter -t 1 -n /usr/bin/cexec"]
