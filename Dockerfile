# Build stage
FROM golang:1.26-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

WORKDIR /workspace

# Copy go mod files
COPY go.mod ./

# Copy source code (needed for go mod tidy to resolve all dependencies)
COPY cmd/ cmd/
COPY pkg/ pkg/

# Download dependencies and populate go.sum with all transitive dependencies
RUN go mod download && go mod tidy

# Build the operator
RUN set -x && go build -v -o operator ./cmd

# Runtime stage
FROM gcr.io/distroless/static:nonroot

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /workspace/operator .


ENTRYPOINT ["/app/operator"]
