# Multi-stage Dockerfile for Parallax Backend Gateway
FROM golang:1.26-alpine AS builder

WORKDIR /src/backend
COPY backend/go.mod ./
COPY backend/ ./

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /bin/parallax-gateway ./cmd/gateway

FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /bin/parallax-gateway /app/parallax-gateway

ENV PARALLAX_ADDR=:8080
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD wget --quiet --tries=1 --spider http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/parallax-gateway"]
