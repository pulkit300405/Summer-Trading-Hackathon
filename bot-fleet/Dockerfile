FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod .

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bot-fleet .

FROM alpine:3.18

RUN apk add --no-cache ca-certificates curl

WORKDIR /app

COPY --from=builder /app/bot-fleet .

HEALTHCHECK --interval=5s --timeout=5s --retries=3 \
  CMD curl -f http://localhost:8081/health || exit 1

EXPOSE 8081

CMD ["./bot-fleet"]
