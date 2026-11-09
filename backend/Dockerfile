FROM golang:1.23-alpine AS builder

WORKDIR /app
RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/api

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata wget \
    && adduser -D -u 10001 appuser

WORKDIR /app
COPY --from=builder /server /app/server
COPY migrations /app/migrations

RUN mkdir -p /app/data/receipts && chown -R appuser:appuser /app

USER appuser

ENV PORT=8080
ENV MIGRATIONS_PATH=/app/migrations
ENV RECEIPTS_DIR=/app/data/receipts

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -qO- http://127.0.0.1:8080/health || exit 1

CMD ["/app/server"]
