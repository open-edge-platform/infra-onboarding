FROM golang:1.23.2-alpine3.20 as readsbstat

COPY . /go/src/github.com/tinkerbell/hub/actions/read_sb_status
WORKDIR /go/src/github.com/tinkerbell/hub/actions/read_sb_status

ENV GO111MODULE=on

RUN go build -o main .

# Build the final image
FROM alpine:3.20.3

COPY --from=readsbstat /go/src/github.com/tinkerbell/hub/actions/read_sb_status/run_sb.sh .
COPY --from=readsbstat /go/src/github.com/tinkerbell/hub/actions/read_sb_status/main .

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:50054/healthz || exit 1

# Command to run the executable
CMD ["/bin/sh", "./run_sb.sh"]
