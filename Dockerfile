# --- Build stage ---
FROM golang:1.25 AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/openapi-mcp ./cmd/openapi-mcp

# --- Runtime stage ---
FROM alpine:latest
RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /app/bin/openapi-mcp ./openapi-mcp
COPY ./specs ./specs
ENTRYPOINT ["/app/openapi-mcp"]
