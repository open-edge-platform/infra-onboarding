#####################################################################################
# INTEL CONFIDENTIAL                                                                #
# Copyright (C) 2024 Intel Corporation                                              #
# This software and the related documents are Intel copyrighted materials,          #
# and your use of them is governed by the express license under which they          #
# were provided to you ("License"). Unless the License provides otherwise,          #
# you may not use, modify, copy, publish, distribute, disclose or transmit          #
# this software or the related documents without Intel's prior written permission.  #
# This software and the related documents are provided as is, with no express       #
# or implied warranties, other than those that are expressly stated in the License. #
#####################################################################################

# Build cexec
FROM golang:1.22-alpine as cexec
RUN apk add --no-cache git ca-certificates gcc linux-headers musl-dev
COPY . /go/src/github.com/thebsdbox/cexec/
WORKDIR /go/src/github.com/thebsdbox/cexec
ENV GO111MODULE=on
RUN --mount=type=cache,sharing=locked,id=gomod,target=/go/pkg/mod/cache \
    --mount=type=cache,sharing=locked,id=goroot,target=/root/.cache/go-build \
    CGO_ENABLED=1 GOOS=linux go build -a -ldflags "-linkmode external -extldflags '-static' -s -w" -o cexec

# Build final image
FROM scratch
COPY --from=cexec /go/src/github.com/thebsdbox/cexec/cexec .
ENTRYPOINT ["/cexec"]
