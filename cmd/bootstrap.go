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
	"github.com/proarash/ecommerce-server/internal/finance"
	"github.com/proarash/ecommerce-server/internal/inventory"
	"github.com/proarash/ecommerce-server/internal/media"
	"github.com/proarash/ecommerce-server/internal/middleware"
	"github.com/proarash/ecommerce-server/internal/notification"
	"github.com/proarash/ecommerce-server/internal/payment"
	"github.com/proarash/ecommerce-server/internal/product"
	"github.com/proarash/ecommerce-server/internal/seed"
	"github.com/proarash/ecommerce-server/internal/staff"
	"github.com/proarash/ecommerce-server/internal/user"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Bootstrap(envConfig *config.EnvConfig) bool {
	if envConfig.JwtSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}

	db, err := gorm.Open(postgres.Open(envConfig.Dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		panic(err)
	}
	if err := db.AutoMigrate(
		&media.Media{},
		&staff.StaffUser{},
		&user.User{},
		&product.Category{},
		&product.Product{},
		&product.Attribute{},
		&cart.Cart{},
		&cart.CartItem{},
		&finance.Order{},
		&finance.OrderItem{},
		&finance.PreInvoice{},
		&payment.PaymentTransaction{},
		&inventory.InventoryStock{},
		&inventory.InventoryLog{},
		&cms.BlogPost{},
		&cms.Banner{},
		&cms.SiteContent{},
		&chat.ChatRoom{},
		&chat.ChatMessage{},
		&notification.Notification{},
	); err != nil {
		panic(err)
	}
	seed.Run(db, envConfig)

	switch envConfig.Env {
	case "production":
		gin.SetMode(gin.ReleaseMode)
	default:
		gin.SetMode(gin.DebugMode)
	}

	router := gin.Default()
	router.SetTrustedProxies([]string{"127.0.0.1"})
	router.Use(middleware.Cors, middleware.ApiResponseMiddleware)
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	authMW := middleware.Auth(envConfig.JwtSecret)
	api := router.Group("/api")
	protected := api.Group("", authMW)
	customer := api.Group("", authMW, middleware.RequireCustomer())
	storekeeper := api.Group("", authMW, middleware.RequireRoles(staff.RoleStorekeeper))
	mediaGroup := api.Group("", authMW, middleware.RequireRoles(staff.RoleStorekeeper, staff.RoleMarketer, staff.RoleSupport))
	accountant := api.Group("", authMW, middleware.RequireRoles(staff.RoleAccountant))
	marketer := api.Group("", authMW, middleware.RequireRoles(staff.RoleMarketer))
	support := api.Group("/support", authMW, middleware.RequireRoles(staff.RoleSupport))
	adminGroup := api.Group("/admin", authMW, middleware.RequireRoles(staff.RoleAdmin))
	ws := router.Group("/ws", authMW)

	staffStore := staff.NewStore(db)
	userStore := user.NewStore(db)

	hub := chat.NewHub()
	chatService := chat.NewService(db, hub)
	telegram := notification.NewTelegramClient(envConfig.TelegramBotToken, envConfig.TelegramAdminChatID)
	notificationStore := notification.NewStore(db)
	notifier := notification.NewService(db, notificationStore, chatService, telegram)
	chatService.OnRoomCreated = notifier.OnChatRoomCreated

	financeStore := finance.NewStore(db, notifier)
	mediaClient := media.NewClient(context.Background(), envConfig)

	auth.NewAuthHandler(auth.NewAuthRepo(staffStore, userStore, envConfig.JwtSecret), envConfig.Domain, envConfig.Env == "production").RegisterRoutes(api)
	user.NewHandler(user.NewService(userStore)).RegisterRoutes(customer)
	staff.NewHandler(staffStore).RegisterRoutes(protected)
	media.NewHandler(media.NewStore(db), mediaClient).RegisterRoutes(mediaGroup)
	product.NewHandler(product.NewStore(db)).RegisterRoutes(api, storekeeper)
	inventory.NewHandler(inventory.NewStore(db)).RegisterRoutes(storekeeper)
	cart.NewHandler(cart.NewStore(db, financeStore)).RegisterRoutes(customer)
	finance.NewHandler(financeStore).RegisterRoutes(accountant, customer, support)
	payment.NewHandler(payment.NewStore(db), payment.NewClient(envConfig.ZibalMerchant, envConfig.ZibalCallbackURL), financeStore, notifier, envConfig.ClientPaymentURL).RegisterRoutes(api, customer, accountant)
	cms.NewHandler(cms.NewStore(db)).RegisterRoutes(api, marketer)
	chat.NewHandler(hub, chatService).RegisterRoutes(ws, customer, support)
	notification.NewHandler(notificationStore, notifier).RegisterRoutes(adminGroup, customer)
	admin.NewHandler(db, staffStore).RegisterRoutes(adminGroup)

	router.Run(":" + envConfig.Port)

	return true
}
