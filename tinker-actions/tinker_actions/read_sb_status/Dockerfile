FROM golang:1.15-alpine as readsbstat

COPY . /go/src/github.com/tinkerbell/hub/actions/read_sb_status
WORKDIR /go/src/github.com/tinkerbell/hub/actions/read_sb_status

ENV GO111MODULE=on

RUN go build -o main .

# Build the final image
FROM alpine:3.19

COPY --from=readsbstat /go/src/github.com/tinkerbell/hub/actions/read_sb_status/run_sb.sh .
COPY --from=readsbstat /go/src/github.com/tinkerbell/hub/actions/read_sb_status/main .

# Command to run the executable
CMD ["/bin/sh", "./run_sb.sh"]
