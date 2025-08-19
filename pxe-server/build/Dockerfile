# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

FROM debian:12.11-slim AS builder

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        dnsmasq=2.90-4~deb12u1 && \
    rm -rf /var/lib/apt/lists/*

# Download sources for dependent packages and create tar archive
RUN mkdir -p /sources && \
    apt-get update && \
    # Install dpkg-dev for source package handling
    apt-get install -y --no-install-recommends dpkg-dev=1.21.22 tar=1.34+dfsg-1.2+deb12u1 xz-utils=5.4.1-1 && \
    # Create source repositories
    echo "deb-src http://deb.debian.org/debian bookworm main" > /etc/apt/sources.list.d/debian-sources.list && \
    echo "deb-src http://deb.debian.org/debian-security bookworm-security main" >> /etc/apt/sources.list.d/debian-sources.list && \
    # Update with --allow-insecure-repositories to avoid signing conflicts
    apt-get update --allow-insecure-repositories || apt-get update && \
    # Define package list with versions
    echo "dnsmasq:2.90-4~deb12u1" > /tmp/target_packages.txt && \
    echo "dnsmasq-base:2.90-4~deb12u1" >> /tmp/target_packages.txt && \
    echo "libdbus-1-3:1.14.10-1~deb12u1" >> /tmp/target_packages.txt && \
    echo "libjansson4:2.14-2" >> /tmp/target_packages.txt && \
    echo "libmnl0:1.0.4-3" >> /tmp/target_packages.txt && \
    echo "libnetfilter-conntrack3:1.0.9-3" >> /tmp/target_packages.txt && \
    echo "libnfnetlink0:1.0.2-2" >> /tmp/target_packages.txt && \
    echo "libnftables1:1.0.6-2+deb12u2" >> /tmp/target_packages.txt && \
    echo "libnftnl11:1.2.4-2" >> /tmp/target_packages.txt && \
    echo "libxtables12:1.8.9-2" >> /tmp/target_packages.txt && \
    echo "netbase:6.4" >> /tmp/target_packages.txt && \
    echo "runit-helper:2.15.2" >> /tmp/target_packages.txt

# Set working directory for source download operations
WORKDIR /sources

# Download source for each package
# hadolint ignore=DL3003
RUN while IFS=: read -r package version; do \
        echo "Getting source for: $package ($version)"; \
        mkdir -p "$package" && \
        (cd "$package" && \
         apt-get source "$package=$version" 2>/dev/null || \
         apt-get source "$package" 2>/dev/null || \
         echo "No source available for $package"); \
    done < /tmp/target_packages.txt && \
    echo "Creating compressed tar archive of all sources" && \
    tar -cJf /sources.tar.xz . && \
    # Clean up apt cache
    rm -rf /var/lib/apt/lists/*

FROM gcr.io/distroless/base-debian12@sha256:cef75d12148305c54ef5769e6511a5ac3c820f39bf5c8a4fbfd5b76b4b8da843

# Copy only the required libraries for dnsmasq
COPY --from=builder /lib/x86_64-linux-gnu/libdbus-1.so.3 /lib/x86_64-linux-gnu/
COPY --from=builder /lib/x86_64-linux-gnu/libidn2.so.0 /lib/x86_64-linux-gnu/
COPY --from=builder /lib/x86_64-linux-gnu/libnetfilter_conntrack.so.3 /lib/x86_64-linux-gnu/
COPY --from=builder /lib/x86_64-linux-gnu/libnettle.so.8 /lib/x86_64-linux-gnu/
COPY --from=builder /lib/x86_64-linux-gnu/libhogweed.so.6 /lib/x86_64-linux-gnu/
COPY --from=builder /lib/x86_64-linux-gnu/libgmp.so.10 /lib/x86_64-linux-gnu/
COPY --from=builder /lib/x86_64-linux-gnu/libnftables.so.1 /lib/x86_64-linux-gnu/
COPY --from=builder /lib/x86_64-linux-gnu/libc.so.6 /lib/x86_64-linux-gnu/
COPY --from=builder /lib/x86_64-linux-gnu/libsystemd.so.0 /lib/x86_64-linux-gnu/
COPY --from=builder /lib/x86_64-linux-gnu/libunistring.so.2 /lib/x86_64-linux-gnu/
COPY --from=builder /lib/x86_64-linux-gnu/libnfnetlink.so.0 /lib/x86_64-linux-gnu/
COPY --from=builder /lib/x86_64-linux-gnu/libmnl.so.0 /lib/x86_64-linux-gnu/
COPY --from=builder /lib/x86_64-linux-gnu/libnftnl.so.11 /lib/x86_64-linux-gnu/
COPY --from=builder /lib/x86_64-linux-gnu/libxtables.so.12 /lib/x86_64-linux-gnu/
COPY --from=builder /lib/x86_64-linux-gnu/libjansson.so.4 /lib/x86_64-linux-gnu/
COPY --from=builder /lib/x86_64-linux-gnu/libcap.so.2 /lib/x86_64-linux-gnu/
COPY --from=builder /lib/x86_64-linux-gnu/libgcrypt.so.20 /lib/x86_64-linux-gnu/
COPY --from=builder /lib/x86_64-linux-gnu/liblzma.so.5 /lib/x86_64-linux-gnu/
COPY --from=builder /lib/x86_64-linux-gnu/libzstd.so.1 /lib/x86_64-linux-gnu/
COPY --from=builder /lib/x86_64-linux-gnu/liblz4.so.1 /lib/x86_64-linux-gnu/
COPY --from=builder /lib/x86_64-linux-gnu/libgpg-error.so.0 /lib/x86_64-linux-gnu/
# Copy the dynamic linker
COPY --from=builder /lib64/ld-linux-x86-64.so.2 /lib64/

COPY --from=builder /usr/sbin/dnsmasq /usr/sbin/dnsmasq
COPY --from=builder /sources.tar.xz /sources.tar.xz

EXPOSE 67/udp 69/udp

ENTRYPOINT ["/usr/sbin/dnsmasq"]
