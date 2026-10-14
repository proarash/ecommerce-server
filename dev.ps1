param([switch]$Seed)

# Server Configuration
$env:ENV = "dev"
$env:PORT = "4000"
$env:DOMAIN = "localhost"
$env:CORS_ORIGINS = "http://localhost:3000"

# Database
$env:DSN = "host=localhost user=postgres password=postgres dbname=ecommerce port=5432 sslmode=disable TimeZone=Asia/Tehran"

# Auth / JWT
$env:JWT_SECRET = "your-super-secret-jwt-key-here"

# Initial Admin Seed
$env:ADMIN_NAME = "Admin"
$env:ADMIN_MOBILE = "09120000000"
$env:ADMIN_PASSWORD = "Admin!123"

# Telegram Notification Bot
$env:TELEGRAM_BOT_TOKEN = ""
$env:TELEGRAM_ADMIN_CHAT_ID = ""

# Media Uploads
$env:UPLOAD_DIR = "uploads"

# Zibal Payment Gateway
$env:ZIBAL_MERCHANT = "zibal"
$env:ZIBAL_CALLBACK_URL = "http://localhost:4000/payment/callback"
$env:CLIENT_PAYMENT_REDIRECT_URL = "http://localhost:3000/payment/result"

if ($Seed) {
    Write-Host "Environment variables set. Running seeds..." -ForegroundColor Cyan
    go run ./cmd/seed
    exit $LASTEXITCODE
}

Write-Host "Environment variables set. Starting air..." -ForegroundColor Cyan
air
