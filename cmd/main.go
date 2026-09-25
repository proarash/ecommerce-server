package main

import (
	"github.com/proarash/ecommerce-server/internal/config"
)

// @title E-Commerce Server API
// @version 1.0
// @description E-commerce backend: catalog, cart, orders, Zibal payments, inventory, CMS, support chat and notifications. All JSON responses are wrapped in {"message": string, "data": any}.
// @host localhost:4000
// @BasePath /
// @schemes http https
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and the JWT token.
func main() {
	config := config.GetEnvConfig()
	Bootstrap(config)
}
