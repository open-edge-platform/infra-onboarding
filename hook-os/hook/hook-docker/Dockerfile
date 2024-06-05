FROM golang:1.20-alpine as dev
COPY . /src/
WORKDIR /src
RUN CGO_ENABLED=0 go build -a -ldflags '-s -w -extldflags "-static"' -o /hook-docker

#use custom docker
FROM hook_dind:latest
RUN echo "http://dl-cdn.alpinelinux.org/alpine/edge/community" >> /etc/apk/repositories
RUN apk --no-cache add kexec-tools
COPY --from=dev /hook-docker .
ENTRYPOINT ["/hook-docker"]
