FROM debian:bookworm-slim AS builder

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        dnsmasq && \
    rm -rf /var/lib/apt/lists/*

FROM gcr.io/distroless/base-debian12

COPY --from=builder /lib /lib
COPY --from=builder /usr/sbin/dnsmasq /usr/sbin/dnsmasq

EXPOSE 67/udp 69/udp

ENTRYPOINT ["/usr/sbin/dnsmasq"]
