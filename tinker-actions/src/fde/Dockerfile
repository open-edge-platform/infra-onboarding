# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

FROM debian:12.8

RUN apt-get update && apt-get install -y parted cryptsetup-bin tpm2-tools xxd
RUN apt-get update && apt-get install -y lvm2 procps

RUN apt-get update && apt-get install -y file fdisk 

COPY enable_fde.sh /
COPY enable_fde_tiber.sh /
COPY os_selection.sh /

RUN chmod +x enable_fde.sh
RUN chmod +x enable_fde_tiber.sh
RUN chmod +x os_selection.sh

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:50054/healthz || exit 1

CMD ["/bin/bash", "/os_selection.sh"]
