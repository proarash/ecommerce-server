package config

import (
	"os"
	"strings"
)

type EnvConfig struct {
	Env                 string
	Dsn                 string
	JwtSecret           string
	Domain              string
	Port                string
	TelegramBotToken    string
	TelegramAdminChatID string
	UploadDir           string
	ZibalMerchant       string
	ZibalCallbackURL    string
	ClientPaymentURL    string
	AdminName           string
	AdminMobile         string
	AdminPassword       string
	CorsOrigins         []string
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func splitList(v string) []string {
	var out []string
	for p := range strings.SplitSeq(v, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
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
		UploadDir:           getEnv("UPLOAD_DIR", "uploads"),
		ZibalMerchant:       getEnv("ZIBAL_MERCHANT", "zibal"),
		ZibalCallbackURL:    getEnv("ZIBAL_CALLBACK_URL", "http://localhost:4000/payment/callback"),
		ClientPaymentURL:    getEnv("CLIENT_PAYMENT_REDIRECT_URL", "http://localhost:3000/payment/result"),
		AdminName:           getEnv("ADMIN_NAME", "Admin"),
		AdminMobile:         os.Getenv("ADMIN_MOBILE"),
		AdminPassword:       os.Getenv("ADMIN_PASSWORD"),
		CorsOrigins:         splitList(getEnv("CORS_ORIGINS", "http://localhost:3000")),
	}
}
