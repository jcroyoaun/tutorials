# Build stage
FROM golang:1.22-alpine AS builder

# Install ca-certificates for HTTPS calls and git for go mod
RUN apk add --no-cache ca-certificates git

WORKDIR /app

# Copy go.mod and go.sum first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the binary with static linking for alpine
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /sentinel \
    ./cmd/sentinel

# Runtime stage - minimal image
FROM alpine:3.19

# Install ca-certificates for AWS SDK HTTPS calls
RUN apk add --no-cache ca-certificates

# Create non-root user for security (UID 10001 for k8s runAsNonRoot verification)
RUN adduser -D -u 10001 -g '' sentinel
USER 10001

WORKDIR /app

# Copy binary from builder
COPY --from=builder /sentinel /app/sentinel

# Run the binary
ENTRYPOINT ["/app/sentinel"]

