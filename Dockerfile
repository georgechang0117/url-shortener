FROM golang:1.13.15-buster as builder

# Create and change to the app directory.
WORKDIR /build

# Retrieve application dependencies.
# This allows the container build to reuse cached dependencies.
# Expecting to copy go.mod and if present go.sum.
COPY ./go.* ./
RUN go mod download

# Copy local code to the container image.
COPY ./ ./

# Build the binary.
RUN go build -mod=readonly -v -o url-shortener ./main

# Use the official Debian slim image for a lean production container.
# https://hub.docker.com/_/debian
# https://docs.docker.com/develop/develop-images/multistage-build/#use-multi-stage-builds
FROM debian:buster-slim
RUN set -x && apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
    ca-certificates && \
    rm -rf /var/lib/apt/lists/*

# Copy the binary to the production image from the builder stage.
COPY --from=builder build/url-shortener /build/url-shortener

EXPOSE 8000 80

# Run the web service on container startup.
ENTRYPOINT ["/build/url-shortener"]
