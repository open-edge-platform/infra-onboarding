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



FROM golang:1.23-alpine AS image2disk
RUN apk add --no-cache git ca-certificates gcc linux-headers musl-dev
WORKDIR /src/
COPY . /src/
#go.mod go.sum /src/
#RUN go mod download
#ENV GO111MODULE=on
#COPY . /src
RUN CGO_ENABLED=1 GOOS=linux go build -a -ldflags "-linkmode external -extldflags '-static' -s -w" -o image2disk

FROM alpine:3.20.3
RUN apk add --no-cache lsblk
# Add Certificates into the image, for anything that does HTTPS calls
COPY --from=image2disk /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=image2disk /src/image2disk /usr/bin/image2disk
# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:50054/healthz || exit 1
ENTRYPOINT ["sh", "-c", "nsenter -t 1 -n /usr/bin/image2disk"]
