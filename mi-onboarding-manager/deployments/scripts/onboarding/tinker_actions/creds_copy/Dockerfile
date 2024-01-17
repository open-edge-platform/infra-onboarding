# syntax=docker/dockerfile:experimental

FROM golang:1.21-alpine as dcopy
RUN apk add --no-cache git ca-certificates gcc linux-headers musl-dev
WORKDIR /app
COPY . .
ENV GO111MODULE=on
RUN --mount=type=cache,sharing=locked,id=gomod,target=/go/pkg/mod/cache \
    --mount=type=cache,sharing=locked,id=goroot,target=/root/.cache/go-build \
    CGO_ENABLED=1 GOOS=linux go build -a -ldflags "-linkmode external -extldflags '-static' -s -w" -o dcopy

# Final stage
FROM scratch
COPY --from=dcopy /app /
CMD ["/dcopy"]

