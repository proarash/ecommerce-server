# PowerShell dev runner. Copy to dev.ps1 (gitignored), fill in real values, then: .\dev.ps1

$env:ENV = 'dev'
$env:PORT = '4000'
$env:DOMAIN = 'localhost'
$env:DSN = 'host=localhost user=postgres password=postgres dbname=ecommerce port=5432 sslmode=disable TimeZone=Asia/Tehran'
$env:JWT_SECRET = 'change-me'
$env:ADMIN_NAME = 'Admin'
$env:ADMIN_MOBILE = '09120000000'
$env:ADMIN_PASSWORD = 'change-me'
$env:TELEGRAM_BOT_TOKEN = ''
$env:TELEGRAM_ADMIN_CHAT_ID = ''
$env:MINIO_ENDPOINT = 'localhost:9000'
$env:MINIO_ACCESS_KEY = 'minioadmin'
$env:MINIO_SECRET_KEY = 'minioadmin'
$env:MINIO_BUCKET = 'ecommerce'
$env:MINIO_USE_SSL = 'false'
$env:MINIO_PUBLIC_URL = ''
$env:ZIBAL_MERCHANT = 'zibal'
$env:ZIBAL_CALLBACK_URL = 'http://localhost:4000/payment/callback'
$env:CLIENT_PAYMENT_REDIRECT_URL = 'http://localhost:3000/payment/result'
$env:CORS_ORIGINS = 'http://localhost:3000'

air
