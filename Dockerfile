#Build stage
# FROM golang:1.23-alpine AS builder

# WORKDIR /app

# # Copy go mod files first for better caching
# COPY go.mod go.sum ./
# RUN go mod download

# # Copy source code
# COPY . .

# # Build the binary
# RUN CGO_ENABLED=0 GOOS=linux go build -o /orderly ./cmd/orderly

# # Runtime stage
# FROM alpine:3.20

# WORKDIR /app

# # Install ca-certificates for HTTPS (if needed for external calls)
# RUN apk add --no-cache ca-certificates

# # Copy binary from builder
# COPY --from=builder /orderly /orderly

# # Expose port
# EXPOSE 8080

# # Run the binary
# ENTRYPOINT ["/orderly"]