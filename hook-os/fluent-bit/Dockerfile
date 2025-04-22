# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# For multi-arch builds - assumption is running on an AMD64 host
FROM multiarch/qemu-user-static:x86_64-arm AS qemu-arm32
FROM multiarch/qemu-user-static:x86_64-aarch64 AS qemu-arm64

FROM debian:bookworm-slim AS deb-extractor
COPY --from=qemu-arm32 /usr/bin/qemu-arm-static /usr/bin/
COPY --from=qemu-arm64 /usr/bin/qemu-aarch64-static /usr/bin/

# Taken from https://github.com/fluent/fluent-bit/blob/v3.2.9/dockerfiles/Dockerfile
# to address CVE-2025-31115 for liblzma5 package
WORKDIR /tmp
SHELL ["/bin/bash", "-o", "pipefail", "-c"]
RUN echo "deb http://deb.debian.org/debian bookworm-backports main" >> /etc/apt/sources.list && \
    apt-get update && \
    apt-get download \
    libssl3 \
    libcurl4 \
    libnghttp2-14 \
    librtmp1 \
    libssh2-1 \
    libpsl5 \
    libbrotli1 \
    libsasl2-2 \
    pkg-config \
    libpq5 \
    libsystemd0/bookworm-backports \
    zlib1g \
    ca-certificates \
    libatomic1 \
    libgcrypt20 \
    libzstd1 \
    liblz4-1 \
    libgssapi-krb5-2 \
    libldap-2.5 \
    libgpg-error0 \
    libkrb5-3 \
    libk5crypto3 \
    libcom-err2 \
    libkrb5support0 \
    libgnutls30 \
    libkeyutils1 \
    libp11-kit0 \
    libidn2-0 \
    libunistring2 \
    libtasn1-6 \
    libnettle8 \
    libhogweed6 \
    libgmp10 \
    libffi8 \
    liblzma5 \
    libyaml-0-2 \
    libcap2 \
    && \
    mkdir -p /dpkg/var/lib/dpkg/status.d/ && \
    for deb in *.deb; do \
    package_name=$(dpkg-deb -I "${deb}" | awk '/^ Package: .*$/ {print $2}'); \
    echo "Processing: ${package_name}"; \
    dpkg --ctrl-tarfile "$deb" | tar -Oxf - ./control > "/dpkg/var/lib/dpkg/status.d/${package_name}"; \
    dpkg --extract "$deb" /dpkg || exit 10; \
    done

# Remove unnecessary files extracted from deb packages like man pages and docs etc.
RUN find /dpkg/ -type d -empty -delete && \
    rm -r /dpkg/usr/share/doc/


# Use an official Ubuntu base image
FROM fluent/fluent-bit:3.2.9 AS fluent-bit-uos

# Replace the vulnerable liblzma5 with the fixed version
COPY --from=deb-extractor /dpkg /

COPY --from=busybox:stable-uclibc /bin/sh /bin/sh
COPY --from=busybox:stable-uclibc /bin/cat /bin/cat

COPY fluentbit_run.sh /fluentbit_run.sh
# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:50054/healthz || exit 1

ENTRYPOINT ["/bin/sh", "-c", "/fluentbit_run.sh"]

