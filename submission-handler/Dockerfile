FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod file
COPY go.mod .

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o submission-handler .

# Final stage
FROM alpine:3.18

RUN apk add --no-cache ca-certificates curl

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/submission-handler .

# Health check
HEALTHCHECK --interval=5s --timeout=5s --retries=3 \
  CMD curl -f http://localhost:8080/health || exit 1

EXPOSE 8080

CMD ["./submission-handler"]
