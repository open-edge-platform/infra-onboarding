# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

FROM debian:bookworm-slim AS builder

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        dnsmasq=2.90-4~deb12u1 && \
    rm -rf /var/lib/apt/lists/*

FROM gcr.io/distroless/base-debian12:latest

COPY --from=builder /lib /lib
COPY --from=builder /usr/sbin/dnsmasq /usr/sbin/dnsmasq

EXPOSE 67/udp 69/udp

ENTRYPOINT ["/usr/sbin/dnsmasq"]
