package main

import (
	"context"
	"log"

	"github.com/proarash/ecommerce-server/internal/config"
	"github.com/proarash/ecommerce-server/internal/database"
	"github.com/proarash/ecommerce-server/internal/seed"
	"github.com/proarash/ecommerce-server/internal/wallet"
)

func main() {
	cfg := config.GetEnvConfig()
	db, err := database.Open(cfg.Dsn)
	if err != nil {
		log.Fatal(err)
	}
	if err := database.Migrate(db); err != nil {
		log.Fatal(err)
	}
	seed.Run(db, cfg)
	if err := wallet.NewStore(db).Backfill(context.Background()); err != nil {
		log.Fatal(err)
	}
	log.Println("seed: done")
}
