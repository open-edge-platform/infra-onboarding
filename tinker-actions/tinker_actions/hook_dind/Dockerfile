#####################################################################################
# INTEL CONFIDENTIAL                                                                #
# Copyright (C) 2023 Intel Corporation                                              #
# This software and the related documents are Intel copyrighted materials,          #
# and your use of them is governed by the express license under which they          #
# were provided to you ("License"). Unless the License provides otherwise,          #
# you may not use, modify, copy, publish, distribute, disclose or transmit          #
# this software or the related documents without Intel's prior written permission.  #
# This software and the related documents are provided as is, with no express       #
# or implied warranties, other than those that are expressly stated in the License. #
#####################################################################################

FROM alpine:3.20.3

RUN apk add --no-cache \
	ca-certificates   \
# DOCKER_HOST=ssh://... -- https://github.com/docker/cli/pull/1014
    openssh-client

# ensure that nsswitch.conf is set up for Go's "netgo" implementation (which Docker explicitly uses)
# - https://github.com/moby/moby/blob/v24.0.6/hack/make.sh#L111
# - https://github.com/golang/go/blob/go1.19.13/src/net/conf.go#L227-L303
# - docker run --rm debian:stretch grep '^hosts:' /etc/nsswitch.conf
RUN [ -e /etc/nsswitch.conf ] && grep '^hosts: files dns' /etc/nsswitch.conf

# pre-add a "docker" group for socket usage
RUN set -eux ; \
addgroup -g 2375 -S docker

RUN set -eux; \
	apk add --no-cache iptables-legacy \
	ip6tables \
	iptables ; \
    \
		# set up a symlink farm we can use PATH to switch to legacy with
	mkdir -p /usr/local/sbin/.iptables-legacy ; \
	# https://git.alpinelinux.org/aports/tree/main/iptables/APKBUILD?id=b215d54de159eacafecb13c68dfadce6eefd9ec9#n73
	for f in \
	iptables \
	iptables-save \
	iptables-restore \
	ip6tables \
	ip6tables-save \
	ip6tables-restore \
	; do \
	# "iptables-save" -> "iptables-legacy-save", "ip6tables" -> "ip6tables-legacy", etc.
	# https://pkgs.alpinelinux.org/contents?branch=v3.19&name=iptables-legacy&arch=x86_64
	b="/sbin/${f/tables/tables-legacy}"; \
	"$b" --version; \
	ln -svT "$b" "/usr/local/sbin/.iptables-legacy/$f"; \
	done; \
	# verify it works (and gets us legacy)
	export PATH="/usr/local/sbin/.iptables-legacy:$PATH"; \
	iptables --version | grep legacy ;\
	\
	addgroup -S dockremap; \
	adduser -S -G dockremap dockremap; \
	echo 'dockremap:165536:65536' >> /etc/subuid; \
	echo 'dockremap:165536:65536' >> /etc/subgid

ENV DOCKER_VERSION 24.0.9
ENV DIND_COMMIT 65cfcc28ab37cb75e1560e4b4738719c07c6618e
RUN set -eux ;\
		apk add --no-cache wget \
			btrfs-progs \
			e2fsprogs \
			e2fsprogs-extra \
			openssl \
			pigz \
			shadow-uidmap \
			xfsprogs \
			xz \
			zfs ; \
		\
		apkArch="$(apk --print-arch)"; \
		case "$apkArch" in \
			'x86_64') \
				url='https://download.docker.com/linux/static/stable/x86_64/docker-24.0.9.tgz'; \
				;; \
			'armhf') \
				url='https://download.docker.com/linux/static/stable/armel/docker-24.0.9.tgz'; \
				;; \
			'armv7') \
				url='https://download.docker.com/linux/static/stable/armhf/docker-24.0.9.tgz'; \
				;; \
			'aarch64') \
				url='https://download.docker.com/linux/static/stable/aarch64/docker-24.0.9.tgz'; \
				;; \
			*) echo >&2 "error: unsupported 'docker.tgz' architecture ($apkArch)"; exit 1 ;; \
		esac; \
		\
		wget -O 'docker.tgz' "$url" ; \
		apk add --no-cache binutils ;  \
		tar --extract \
			--file docker.tgz \
			--strip-components 1 \
			--directory /usr/local/bin/ \
			--no-same-owner \
			; \
		rm docker.tgz; \
		\
		docker --version;  \
		dockerd --version; \
		containerd --version; \
		ctr --version; \
		runc --version; \
		wget -O /usr/local/bin/dind "https://raw.githubusercontent.com/docker/docker/${DIND_COMMIT}/hack/dind"; \
		chmod +x /usr/local/bin/dind; \
		strip /usr/local/bin/docker /usr/local/bin/dockerd /usr/local/bin/containerd /usr/local/bin/docker-proxy /usr/local/bin/runc; \
		apk del binutils wget

COPY modprobe.sh /usr/local/bin/modprobe
COPY docker-entrypoint.sh /usr/local/bin/

# https://github.com/docker-library/docker/pull/166
#   dockerd-entrypoint.sh uses DOCKER_TLS_CERTDIR for auto-generating TLS certificates
#   docker-entrypoint.sh uses DOCKER_TLS_CERTDIR for auto-setting DOCKER_TLS_VERIFY and DOCKER_CERT_PATH
# (For this to work, at least the "client" subdirectory of this path needs to be shared between the client and server containers via a volume, "docker cp", or other means of data sharing.)
ENV DOCKER_TLS_CERTDIR=/certs
# also, ensure the directory pre-exists and has wide enough permissions for "dockerd-entrypoint.sh" to create subdirectories, even when run in "rootless" mode
RUN mkdir /certs /certs/client && chmod 1777 /certs /certs/client
# (doing both /certs and /certs/client so that if Docker does a "copy-up" into a volume defined on /certs/client, it will "do the right thing" by default in a way that still works for rootless users)

COPY dockerd-entrypoint.sh /usr/local/bin/

VOLUME /var/lib/docker
EXPOSE 2375 2376

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:50054/healthz || exit 1
ENTRYPOINT ["dockerd-entrypoint.sh"]
CMD []
