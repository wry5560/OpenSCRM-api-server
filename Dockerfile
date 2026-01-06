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

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o api-server .

# Production stage
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Set timezone
ENV TZ=Asia/Shanghai

# Copy binary from builder
COPY --from=builder /app/api-server .
COPY --from=builder /app/conf ./conf

# Use docker config as default
RUN if [ -f ./conf/config.docker.yaml ]; then cp ./conf/config.docker.yaml ./conf/config.yaml; fi

# Expose port
EXPOSE 9003

# Run the binary
CMD ["./api-server"]
