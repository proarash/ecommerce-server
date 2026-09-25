package config

import "os"

type EnvConfig struct {
	Env                 string
	Dsn                 string
	JwtSecret           string
	Domain              string
	Port                string
	TelegramBotToken    string
	TelegramAdminChatID string
	MinioEndpoint       string
	MinioAccessKey      string
	MinioSecretKey      string
	MinioBucket         string
	MinioUseSSL         string
	MinioPublicURL      string
	ZibalMerchant       string
	ZibalCallbackURL    string
	ClientPaymentURL    string
	AdminName           string
	AdminMobile         string
	AdminPassword       string
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func GetEnvConfig() *EnvConfig {
	return &EnvConfig{
		Env:                 os.Getenv("ENV"),
		Dsn:                 os.Getenv("DSN"),
		JwtSecret:           os.Getenv("JWT_SECRET"),
		Domain:              getEnv("DOMAIN", "localhost"),
		Port:                getEnv("PORT", "4000"),
		TelegramBotToken:    os.Getenv("TELEGRAM_BOT_TOKEN"),
		TelegramAdminChatID: os.Getenv("TELEGRAM_ADMIN_CHAT_ID"),
		MinioEndpoint:       os.Getenv("MINIO_ENDPOINT"),
		MinioAccessKey:      os.Getenv("MINIO_ACCESS_KEY"),
		MinioSecretKey:      os.Getenv("MINIO_SECRET_KEY"),
		MinioBucket:         getEnv("MINIO_BUCKET", "ecommerce"),
		MinioUseSSL:         os.Getenv("MINIO_USE_SSL"),
		MinioPublicURL:      os.Getenv("MINIO_PUBLIC_URL"),
		ZibalMerchant:       getEnv("ZIBAL_MERCHANT", "zibal"),
		ZibalCallbackURL:    getEnv("ZIBAL_CALLBACK_URL", "http://localhost:4000/api/payment/callback"),
		ClientPaymentURL:    getEnv("CLIENT_PAYMENT_REDIRECT_URL", "http://localhost:3000/payment/result"),
		AdminName:           getEnv("ADMIN_NAME", "Admin"),
		AdminMobile:         os.Getenv("ADMIN_MOBILE"),
		AdminPassword:       os.Getenv("ADMIN_PASSWORD"),
	}
}
