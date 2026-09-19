package main

import (
	"github.com/gin-gonic/gin"
	"github.com/proarash/ecommerce-server/internal/auth"
	"github.com/proarash/ecommerce-server/internal/config"
	"github.com/proarash/ecommerce-server/internal/middleware"
	"github.com/proarash/ecommerce-server/internal/user"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Bootstrap(envConfig *config.EnvConfig) bool {
	db, err := gorm.Open(postgres.Open(envConfig.Dsn), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	db.AutoMigrate(&user.User{})

	switch envConfig.Env {
	case "":
		gin.SetMode(gin.DebugMode)
	case "dev":
		gin.SetMode(gin.DebugMode)
	case "production":
		gin.SetMode(gin.ReleaseMode)
	default:
		gin.SetMode(gin.DebugMode)
	}

	router := gin.Default()
	router.SetTrustedProxies([]string{"127.0.0.1"})
	router.Use(middleware.ApiResponseMiddleware)

	auth.NewAuthHandler(auth.NewAuthRepo(db)).RegisterRoutes(router)

	router.Run(":4000")

	return true
}
