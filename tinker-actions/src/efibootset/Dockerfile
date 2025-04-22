# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

FROM debian:12.10

# Update perl-base to the fixed version (CVE-2024-56406)
RUN apt-get update && apt-get install -y perl-base=5.36.0-7+deb12u2

RUN apt update && apt install -y parted grub-common udev grub-pc-bin grub-efi-amd64-bin dosfstools wget iproute2

COPY efibootset.sh /

RUN chmod +x /efibootset.sh


# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:50054/healthz || exit 1

CMD ["/bin/bash", "/efibootset.sh"]
