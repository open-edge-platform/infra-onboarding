# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

FROM caddy:2.8.4-alpine

# Copy self signed certificate and key
COPY "caddy-key.pem" "/etc/caddy/caddy-key.pem"
COPY "caddy-cert.pem" "/etc/caddy/caddy-cert.pem"

COPY caddy_run.sh /caddy_run.sh

RUN chmod +x /caddy_run.sh

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:50054/healthz || exit 1

CMD ["/bin/sh", "/caddy_run.sh"]
