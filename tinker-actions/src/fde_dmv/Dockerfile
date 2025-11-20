# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

FROM debian:12.10@sha256:264982ff4d18000fa74540837e2c43ca5137a53a83f8f62c7b3803c0f0bdcd56

# Update packages to fix CVEs
# CVE-2023-31484: perl-base
# CVE-2025-4802: libc6, libc-bin
# CVE-2025-32988, CVE-2025-32990: libgnutls30
RUN apt-get update && apt-get install -y --no-install-recommends \
    perl-base=5.36.0-7+deb12u3 \
    libc6=2.36-9+deb12u11 \
    libc-bin=2.36-9+deb12u11 \
    libgnutls30=3.7.9-2+deb12u5

RUN apt-get update && apt-get install -y --no-install-recommends parted cryptsetup-bin tpm2-tools xxd
RUN apt-get update && apt-get install -y --no-install-recommends lvm2 procps

RUN apt-get update && apt-get install -y --no-install-recommends file fdisk

COPY enable_fde.sh /
COPY enable_fde_emt.sh /
COPY enable_dmv_emt.sh /
COPY os_selection.sh /

RUN chmod +x enable_fde.sh
RUN chmod +x enable_fde_emt.sh
RUN chmod +x enable_dmv_emt.sh
RUN chmod +x os_selection.sh

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:50054/healthz || exit 1

CMD ["/bin/bash", "/os_selection.sh"]
