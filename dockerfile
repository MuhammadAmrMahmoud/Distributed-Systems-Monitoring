# =========================
# Build stage
# =========================
FROM golang:1.25.1-alpine AS builder

# Install git (for go modules if needed)
RUN apk add --no-cache git

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o app ./  

# =========================
# Runtime stage
# =========================
FROM gcr.io/distroless/base-debian12

WORKDIR /app

# Copy binary and config
COPY --from=builder /app/app /app/app
COPY --from=builder /app/config.json /app/config.json


# Expose Gin port
EXPOSE 8080

# Run app
ENTRYPOINT ["/app/app"]
