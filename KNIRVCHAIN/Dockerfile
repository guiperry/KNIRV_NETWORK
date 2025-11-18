# Use the official Go image as a builder
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum to leverage Docker layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code for the oracle
COPY . .

# Build the binary with testnet and no_webview tags for a headless server.
# CGO_ENABLED=0 creates a static binary.
RUN CGO_ENABLED=0 go build -tags "testnet,no_webview" -o /knirv-oracle .

# --- Final Stage ---
# Use a minimal base image for the final container
FROM alpine:latest

# Copy the pre-built binary from the builder stage
COPY --from=builder /knirv-oracle /usr/local/bin/

# Expose the port the service runs on
EXPOSE 1317

# Command to run the service. Arguments are passed here.
CMD ["knirv-oracle", "--testnet", "--disable-p2p", "--port", "1317"]