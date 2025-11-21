# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the example service
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o example-service ./cmd/example-service

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/example-service .

# Expose port
EXPOSE 8080

# Run the service
CMD ["./example-service"]

