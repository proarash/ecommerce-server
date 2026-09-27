#!/usr/bin/env bash
# Linux dev runner. Copy to dev.sh (gitignored), fill in real values, then: ./dev.sh
set -e

export ENV='dev'
export PORT='4000'
export DOMAIN='localhost'
export DSN='host=localhost user=postgres password=postgres dbname=ecommerce port=5432 sslmode=disable TimeZone=Asia/Tehran'
export JWT_SECRET='change-me'
export ADMIN_NAME='Admin'
export ADMIN_MOBILE='09120000000'
export ADMIN_PASSWORD='change-me'
export TELEGRAM_BOT_TOKEN=''
export TELEGRAM_ADMIN_CHAT_ID=''
export MINIO_ENDPOINT='localhost:9000'
export MINIO_ACCESS_KEY='minioadmin'
export MINIO_SECRET_KEY='minioadmin'
export MINIO_BUCKET='ecommerce'
export MINIO_USE_SSL='false'
export MINIO_PUBLIC_URL=''
export ZIBAL_MERCHANT='zibal'
export ZIBAL_CALLBACK_URL='http://localhost:4000/api/payment/callback'
export CLIENT_PAYMENT_REDIRECT_URL='http://localhost:3000/payment/result'
export CORS_ORIGINS='http://localhost:3000'

air
