# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.24.6-alpine3.21 as readsbstat

COPY . /go/src/github.com/tinkerbell/hub/actions/read_sb_status
WORKDIR /go/src/github.com/tinkerbell/hub/actions/read_sb_status

ENV GO111MODULE=on

RUN go build -o main .

# Build the final image
FROM alpine:3.21.3

COPY --from=readsbstat /go/src/github.com/tinkerbell/hub/actions/read_sb_status/run_sb.sh .
COPY --from=readsbstat /go/src/github.com/tinkerbell/hub/actions/read_sb_status/main .

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:50054/healthz || exit 1

# Command to run the executable
CMD ["/bin/sh", "./run_sb.sh"]
