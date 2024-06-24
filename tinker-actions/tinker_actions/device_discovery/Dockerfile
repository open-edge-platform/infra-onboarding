# Start from the official Alpine image
FROM alpine:3.19

# Install dmidecode
RUN apk update && apk add --no-cache dmidecode

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the Pre-built binary file from the local system to the Working Directory inside the container
COPY app . 

# Command to run the executable
CMD ["./app"]
