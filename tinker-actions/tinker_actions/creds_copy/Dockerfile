# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.23.2-alpine3.20 as dcopy
RUN apk add --no-cache git ca-certificates gcc linux-headers musl-dev
WORKDIR /app
COPY . .
ENV GO111MODULE=on
RUN --mount=type=cache,sharing=locked,id=gomod,target=/go/pkg/mod/cache \
    --mount=type=cache,sharing=locked,id=goroot,target=/root/.cache/go-build \
    CGO_ENABLED=1 GOOS=linux go build -a -ldflags "-linkmode external -extldflags '-static' -s -w" -o dcopy

# Final stage
FROM alpine:3.20.3
RUN apk add --no-cache lsblk
COPY --from=dcopy /app /
# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:50054/healthz || exit 1

CMD ["/dcopy"]
