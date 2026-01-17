# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make protobuf protobuf-dev

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN make build

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache \
    protobuf \
    ca-certificates \
    git \
    bash

# Install language-specific protoc plugins
# Go
RUN apk add --no-cache go && \
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Python
RUN apk add --no-cache python3 py3-pip && \
    pip3 install --no-cache-dir grpcio-tools

# Copy binary from builder
COPY --from=builder /build/bin/buffalo /usr/local/bin/buffalo

# Set working directory
WORKDIR /workspace

# Set entrypoint
ENTRYPOINT ["buffalo"]
CMD ["--help"]

# Labels
LABEL maintainer="massonsky"
LABEL description="Buffalo Protocol Buffer Compiler"
LABEL version="0.5.0"
