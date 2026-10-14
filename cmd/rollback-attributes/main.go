package main

import (
	"log"

	"github.com/proarash/ecommerce-server/internal/config"
	"github.com/proarash/ecommerce-server/internal/database"
)

func main() {
	cfg := config.GetEnvConfig()
	db, err := database.Open(cfg.Dsn)
	if err != nil {
		log.Fatal(err)
	}
	if err := database.RollbackAttributeValues(db); err != nil {
		log.Fatal(err)
	}
	log.Println("rollback: attributes restored to the legacy schema")
}
