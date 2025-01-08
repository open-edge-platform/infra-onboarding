FROM golang:1.20-alpine AS dev
COPY . /src/
WORKDIR /src
RUN CGO_ENABLED=0 go build -a -ldflags '-s -w -extldflags "-static"' -o /hook-docker

#use custom docker
FROM hook_dind:latest
RUN echo "http://dl-cdn.alpinelinux.org/alpine/edge/community" >> /etc/apk/repositories
RUN apk add --no-cache kexec-tools binutils && rm -rf /var/cache/apk/*
# Won't use docker-buildx nor docker-compose
RUN rm -rf /usr/local/libexec/docker/cli-plugins
# Strip some large binaries
RUN strip /usr/local/bin/docker /usr/local/bin/dockerd /usr/local/bin/docker-proxy /usr/local/bin/runc
# Purge binutils package after stripping
RUN apk del binutils
COPY --from=dev /hook-docker .

ENTRYPOINT ["/hook-docker"]
