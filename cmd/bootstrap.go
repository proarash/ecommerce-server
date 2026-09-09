package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/proarash/ecommerce-server/internal/auth"
	"github.com/proarash/ecommerce-server/internal/user"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Bootstrap(env string) bool {

	db, err := gorm.Open(postgres.Open(""), &gorm.Config{})
	if err != nil {
		log.Println(err)
	}
	db.AutoMigrate(&user.User{})
	if env == "" {
		env = "dev"
	}

	router := gin.Default()
	router.SetTrustedProxies([]string{"127.0.0.1"})

	auth.NewAuthHandler(auth.NewAuthRepo(db)).RegisterRoutes(router)

	router.Run(":4000")

	return true
}
