FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod .

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o telemetry-ingester .

FROM alpine:3.18

RUN apk add --no-cache ca-certificates curl

WORKDIR /app

COPY --from=builder /app/telemetry-ingester .

HEALTHCHECK --interval=5s --timeout=5s --retries=3 \
  CMD curl -f http://localhost:8082/health || exit 1

EXPOSE 8082

CMD ["./telemetry-ingester"]
