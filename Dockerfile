# ---- build ----
FROM golang:1.27.1-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/server ./cmd
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/seed ./cmd/seed

# ---- run ----
FROM alpine:3.24

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

COPY --from=builder /out/server /usr/local/bin/server
COPY --from=builder /out/seed /usr/local/bin/seed

WORKDIR /app

EXPOSE 4000

ENTRYPOINT ["sh", "-c", "export ENV=\"${ENV:-dev}\" PORT=\"${PORT:-4000}\" DOMAIN=\"${DOMAIN:-localhost}\" DSN=\"${DSN:-}\" JWT_SECRET=\"${JWT_SECRET:-}\" SITE_NAME=\"${SITE_NAME:-Shop}\" SITE_URL=\"${SITE_URL:-http://localhost:3000}\" ADMIN_NAME=\"${ADMIN_NAME:-Admin}\" ADMIN_MOBILE=\"${ADMIN_MOBILE:-}\" ADMIN_PASSWORD=\"${ADMIN_PASSWORD:-}\" TELEGRAM_BOT_TOKEN=\"${TELEGRAM_BOT_TOKEN:-}\" TELEGRAM_ADMIN_CHAT_ID=\"${TELEGRAM_ADMIN_CHAT_ID:-}\" UPLOAD_DIR=\"${UPLOAD_DIR:-/app/uploads}\" ZIBAL_MERCHANT=\"${ZIBAL_MERCHANT:-zibal}\" ZIBAL_CALLBACK_URL=\"${ZIBAL_CALLBACK_URL:-}\" CLIENT_PAYMENT_REDIRECT_URL=\"${CLIENT_PAYMENT_REDIRECT_URL:-}\" CORS_ORIGINS=\"${CORS_ORIGINS:-}\" && exec server \"$@\"", "--"]
