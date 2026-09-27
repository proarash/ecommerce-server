@echo off
rem Windows dev runner. Copy to dev.bat (gitignored), fill in real values, then: .\dev.bat
rem Note: a literal % in a value must be written as %%.
setlocal

set "ENV=dev"
set "PORT=4000"
set "DOMAIN=localhost"
set "DSN=host=localhost user=postgres password=postgres dbname=ecommerce port=5432 sslmode=disable TimeZone=Asia/Tehran"
set "JWT_SECRET=change-me"
set "ADMIN_NAME=Admin"
set "ADMIN_MOBILE=09120000000"
set "ADMIN_PASSWORD=change-me"
set "TELEGRAM_BOT_TOKEN="
set "TELEGRAM_ADMIN_CHAT_ID="
set "MINIO_ENDPOINT=localhost:9000"
set "MINIO_ACCESS_KEY=minioadmin"
set "MINIO_SECRET_KEY=minioadmin"
set "MINIO_BUCKET=ecommerce"
set "MINIO_USE_SSL=false"
set "MINIO_PUBLIC_URL="
set "ZIBAL_MERCHANT=zibal"
set "ZIBAL_CALLBACK_URL=http://localhost:4000/api/payment/callback"
set "CLIENT_PAYMENT_REDIRECT_URL=http://localhost:3000/payment/result"
set "CORS_ORIGINS=http://localhost:3000"

air
