# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# Use a temporary stage to extract the fixed version of liblzma5
FROM debian:bookworm AS liblzma-extractor
RUN apt-get update && apt-get install -y --no-install-recommends liblzma5=5.4.1-1
# Copy the fixed library to a temporary location
RUN mkdir /fixed-libs && cp /usr/lib/x86_64-linux-gnu/liblzma.so.5.4.1 /fixed-libs/liblzma.so.5.4.1
RUN awk '/^Package: liblzma5$/,/^$/ {print}' /var/lib/dpkg/status > /fixed-libs/liblzma5

# Use an official Ubuntu base image
FROM fluent/fluent-bit:3.2.9 AS fluent-bit-uos

# Replace the vulnerable liblzma5 with the fixed version
COPY --from=liblzma-extractor /fixed-libs/liblzma.so.5.4.1 /lib/x86_64-linux-gnu/liblzma.so.5.4.1
COPY --from=liblzma-extractor /fixed-libs/liblzma5 /var/lib/dpkg/status.d/liblzma5

COPY --from=busybox:stable-uclibc /bin/sh /bin/sh
COPY --from=busybox:stable-uclibc /bin/cat /bin/cat

COPY fluentbit_run.sh /fluentbit_run.sh
# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:50054/healthz || exit 1

ENTRYPOINT ["/bin/sh", "-c", "/fluentbit_run.sh"]

