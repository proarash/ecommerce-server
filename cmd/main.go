package main

import (
	"github.com/proarash/ecommerce-server/internal/config"
)

func main() {
	config := config.GetEnvConfig()
	Bootstrap(config)
}
