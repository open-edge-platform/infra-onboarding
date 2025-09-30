# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# NOTICE: This file has been modified by Intel Corporation.
# Original file can be found at https://github.com/tinkerbell/actions.

# Build stream
FROM golang:1.24.6-alpine3.21 AS writefile
RUN apk add --no-cache git ca-certificates gcc musl-dev 
COPY . /go/src/github.com/tinkerbell/hub/actions/writefile/v1
COPY pkg /go/src/github.com/tinkerbell/hub/actions/pkg
WORKDIR /go/src/github.com/tinkerbell/hub/actions/writefile/v1
ENV GO111MODULE=on
RUN --mount=type=cache,sharing=locked,id=gomod,target=/go/pkg/mod/cache \
    --mount=type=cache,sharing=locked,id=goroot,target=/root/.cache/go-build \
    CGO_ENABLED=1 GOOS=linux go build -a -ldflags "-linkmode external -extldflags '-static' -s -w" -o writefile


# Build final image
FROM alpine:3.21.3
RUN apk add --no-cache lsblk
# Add Certificates into the image, for anything that does HTTPS calls
COPY --from=writefile /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=writefile /go/src/github.com/tinkerbell/hub/actions/writefile/v1/writefile .

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:50054/healthz || exit 1
ENTRYPOINT ["/writefile"]
