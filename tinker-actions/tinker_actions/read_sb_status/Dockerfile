FROM ubuntu:22.04

RUN apt-get update && \
    apt-get install -y ca-certificates curl getty-run jq mokutil && \
    curl -O https://dl.google.com/go/go1.20.linux-amd64.tar.gz && \
    tar -xvf go1.20.linux-amd64.tar.gz && \
    mv go /usr/local
ENV GOROOT=/usr/local/go
ENV GOPATH=$HOME/go
ENV PATH=$GOPATH/bin:$GOROOT/bin:$PATH

RUN apt-get install -y ca-certificates curl getty-run jq
RUN apt-get install -y parted cryptsetup-bin tpm2-tools xxd dnsutils


# Set the working directory inside the container
WORKDIR /app
 
# Copy the local package files to the container's workspace
COPY . .
 
# Build the Go app
RUN go build -o main .
 
# Command to run the executable
CMD ["/bin/bash", "./run_sb.sh"]
