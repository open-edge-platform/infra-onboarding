# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# Use an official Ubuntu base image
FROM fluent/fluent-bit:3.2.10 AS fluent-bit-uos

# Update liblzma5 to the fixed version (CVE-2025-31115)
RUN apt-get update && apt-get install -y --no-install-recommends liblzma5=5.4.1-1 && apt-get clean

COPY --from=busybox:stable-uclibc /bin/sh /bin/sh
COPY --from=busybox:stable-uclibc /bin/cat /bin/cat

COPY fluentbit_run.sh /fluentbit_run.sh
# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:50054/healthz || exit 1

ENTRYPOINT ["/bin/sh", "-c", "/fluentbit_run.sh"]

