package main

import (
	"context"
	"log"

	"github.com/gin-gonic/gin"
	_ "github.com/proarash/ecommerce-server/docs"
	"github.com/proarash/ecommerce-server/internal/admin"
	"github.com/proarash/ecommerce-server/internal/auth"
	"github.com/proarash/ecommerce-server/internal/cart"
	"github.com/proarash/ecommerce-server/internal/chat"
	"github.com/proarash/ecommerce-server/internal/cms"
	"github.com/proarash/ecommerce-server/internal/config"
	"github.com/proarash/ecommerce-server/internal/database"
	"github.com/proarash/ecommerce-server/internal/discount"
	"github.com/proarash/ecommerce-server/internal/finance"
	"github.com/proarash/ecommerce-server/internal/inventory"
	"github.com/proarash/ecommerce-server/internal/media"
	"github.com/proarash/ecommerce-server/internal/middleware"
	"github.com/proarash/ecommerce-server/internal/notification"
	"github.com/proarash/ecommerce-server/internal/payment"
	"github.com/proarash/ecommerce-server/internal/product"
	"github.com/proarash/ecommerce-server/internal/user"
	"github.com/proarash/ecommerce-server/internal/wallet"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func Bootstrap(envConfig *config.EnvConfig) {
	if envConfig.JwtSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}

	db, err := database.Open(envConfig.Dsn)
	if err != nil {
		panic(err)
	}
	if err := database.Migrate(db); err != nil {
		panic(err)
	}
	if err := wallet.NewStore(db).Backfill(context.Background()); err != nil {
		panic(err)
	}

	switch envConfig.Env {
	case "production":
		gin.SetMode(gin.ReleaseMode)
	default:
		gin.SetMode(gin.DebugMode)
	}

	router := gin.Default()
	router.SetTrustedProxies([]string{"127.0.0.1"})
	router.Use(middleware.Cors(envConfig.CorsOrigins, envConfig.Env != "production"), middleware.ApiResponseMiddleware)
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	userStore := user.NewStore(db)
	authRepo := auth.NewAuthRepo(db, userStore, envConfig.JwtSecret)
	cookies := middleware.CookieWriter{Domain: envConfig.Domain, Secure: envConfig.Env == "production"}
	authMW := middleware.Auth(envConfig.JwtSecret, authRepo, cookies)
	api := router.Group("")
	protected := api.Group("", authMW)
	storekeeper := api.Group("", authMW, middleware.RequireRoles(user.RoleStorekeeper))
	accountant := api.Group("", authMW, middleware.RequireRoles(user.RoleAccountant))
	marketer := api.Group("", authMW, middleware.RequireRoles(user.RoleMarketer))
	support := api.Group("/support", authMW, middleware.RequireRoles(user.RoleSupport))
	discountReader := api.Group("", authMW, middleware.RequireRoles(user.RoleAccountant, user.RoleMarketer, user.RoleSupport))
	discountManager := api.Group("", authMW, middleware.RequireRoles(user.RoleAccountant, user.RoleMarketer))
	adminGroup := api.Group("/admin", authMW, middleware.RequireRoles(user.RoleAdmin))
	ws := router.Group("/ws", middleware.AuthWithQueryToken(envConfig.JwtSecret, authRepo, cookies))

	hub := chat.NewHub()
	chatService := chat.NewService(db, hub)
	telegram := notification.NewTelegramClient(envConfig.TelegramBotToken, envConfig.TelegramAdminChatID)
	notificationStore := notification.NewStore(db)
	notifier := notification.NewService(db, notificationStore, chatService, telegram)
	chatService.OnRoomCreated = notifier.OnChatRoomCreated

	financeStore := finance.NewStore(db, notifier)
	walletStore := wallet.NewStore(db)
	discountStore := discount.NewStore(db)
	mediaService := media.NewService(media.NewStore(db), envConfig.UploadDir)

	auth.NewAuthHandler(authRepo, user.NewService(userStore), envConfig.JwtSecret, cookies).RegisterRoutes(api, protected, adminGroup)
	media.NewHandler(mediaService).RegisterRoutes(api, storekeeper)
	product.NewHandler(product.NewStore(db)).RegisterRoutes(api, storekeeper)
	inventory.NewHandler(inventory.NewStore(db)).RegisterRoutes(storekeeper)
	cart.NewHandler(cart.NewStore(db, financeStore, discountStore)).RegisterRoutes(protected)
	finance.NewHandler(financeStore).RegisterRoutes(accountant, protected, support)
	payment.NewHandler(payment.NewStore(db), payment.NewClient(envConfig.ZibalMerchant, envConfig.ZibalCallbackURL), financeStore, walletStore, notifier, envConfig.ClientPaymentURL).RegisterRoutes(api, protected, accountant)
	cms.NewHandler(cms.NewStore(db)).RegisterRoutes(api, marketer)
	chat.NewHandler(hub, chatService).RegisterRoutes(ws, protected, support)
	notification.NewHandler(notificationStore, notifier).RegisterRoutes(adminGroup, protected)
	admin.NewHandler(db, userStore).RegisterRoutes(adminGroup)
	wallet.NewHandler(walletStore).RegisterRoutes(protected, accountant)
	discount.NewHandler(discountStore, userStore).RegisterRoutes(discountReader, discountManager, protected)

	log.Println("server listening on :" + envConfig.Port)
	if err := router.Run(":" + envConfig.Port); err != nil {
		log.Fatal(err)
	}
}
