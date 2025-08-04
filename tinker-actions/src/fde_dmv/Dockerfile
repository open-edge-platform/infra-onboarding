# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

FROM debian:12.10

# Update perl-base to the fixed version (CVE-2024-56406)
RUN apt-get update && apt-get install -y --no-install-recommends perl-base=5.36.0-7+deb12u2

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
