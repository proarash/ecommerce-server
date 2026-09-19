package config

import "os"

type EnvConfig struct {
	Env       string
	Dsn       string
	JwtSecret string
	Domain    string
}

func GetEnvConfig() *EnvConfig {
	env := os.Getenv("ENV")
	dsn := os.Getenv("DSN")
	jwtSecret := os.Getenv("JWT_SECRET")
	domain := os.Getenv("DOMAIN")

	return &EnvConfig{
		Env:       env,
		Dsn:       dsn,
		JwtSecret: jwtSecret,
		Domain:    domain,
	}
}
