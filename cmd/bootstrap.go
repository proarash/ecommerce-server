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
	"github.com/proarash/ecommerce-server/internal/discount"
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
	"github.com/proarash/ecommerce-server/internal/wallet"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Bootstrap(envConfig *config.EnvConfig) {
	if envConfig.JwtSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}

	// TODO: remove DisableAutomaticPing once PostgreSQL is available (it lets the server start without a database)
	db, err := gorm.Open(postgres.Open(envConfig.Dsn), &gorm.Config{TranslateError: true, DisableAutomaticPing: true})
	if err != nil {
		panic(err)
	}
	if err := db.AutoMigrate(
		&media.Media{},
		&wallet.Wallet{},
		&wallet.WalletTransaction{},
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
		&discount.Discount{},
		&auth.RefreshToken{},
	); err != nil {
		panic(err)
	}
	seed.Run(db, envConfig)
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
	router.Use(middleware.Cors(envConfig.CorsOrigins), middleware.ApiResponseMiddleware)
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	cookies := middleware.CookieWriter{Domain: envConfig.Domain, Secure: envConfig.Env == "production"}
	staffStore := staff.NewStore(db)
	userStore := user.NewStore(db)
	authRepo := auth.NewAuthRepo(db, staffStore, userStore, envConfig.JwtSecret)
	authMW := middleware.Auth(envConfig.JwtSecret, cookies, authRepo)
	api := router.Group("")
	protected := api.Group("", authMW)
	customer := api.Group("", authMW, middleware.RequireCustomer())
	storekeeper := api.Group("", authMW, middleware.RequireRoles(staff.RoleStorekeeper))
	// mediaGroup := api.Group("", authMW, middleware.RequireRoles(staff.RoleStorekeeper, staff.RoleMarketer, staff.RoleSupport))
	accountant := api.Group("", authMW, middleware.RequireRoles(staff.RoleAccountant))
	marketer := api.Group("", authMW, middleware.RequireRoles(staff.RoleMarketer))
	support := api.Group("/support", authMW, middleware.RequireRoles(staff.RoleSupport))
	discountReader := api.Group("", authMW, middleware.RequireRoles(staff.RoleAccountant, staff.RoleMarketer, staff.RoleSupport))
	discountManager := api.Group("", authMW, middleware.RequireRoles(staff.RoleAccountant, staff.RoleMarketer))
	adminGroup := api.Group("/admin", authMW, middleware.RequireRoles(staff.RoleAdmin))
	ws := router.Group("/ws", authMW)


	hub := chat.NewHub()
	chatService := chat.NewService(db, hub)
	telegram := notification.NewTelegramClient(envConfig.TelegramBotToken, envConfig.TelegramAdminChatID)
	notificationStore := notification.NewStore(db)
	notifier := notification.NewService(db, notificationStore, chatService, telegram)
	chatService.OnRoomCreated = notifier.OnChatRoomCreated

	financeStore := finance.NewStore(db, notifier)
	walletStore := wallet.NewStore(db)
	discountStore := discount.NewStore(db)
	// TODO: MinIO disabled for now
	// mediaClient := media.NewClient(context.Background(), envConfig)

	auth.NewAuthHandler(authRepo, cookies, envConfig.JwtSecret).RegisterRoutes(api, adminGroup)
	user.NewHandler(user.NewService(userStore)).RegisterRoutes(customer)
	staff.NewHandler(staffStore).RegisterRoutes(protected)
	// media.NewHandler(media.NewStore(db), mediaClient).RegisterRoutes(mediaGroup)
	product.NewHandler(product.NewStore(db)).RegisterRoutes(api, storekeeper)
	inventory.NewHandler(inventory.NewStore(db)).RegisterRoutes(storekeeper)
	cart.NewHandler(cart.NewStore(db, financeStore, discountStore)).RegisterRoutes(customer)
	finance.NewHandler(financeStore).RegisterRoutes(accountant, customer, support)
	payment.NewHandler(payment.NewStore(db), payment.NewClient(envConfig.ZibalMerchant, envConfig.ZibalCallbackURL), financeStore, walletStore, notifier, envConfig.ClientPaymentURL).RegisterRoutes(api, customer, accountant)
	cms.NewHandler(cms.NewStore(db)).RegisterRoutes(api, marketer)
	chat.NewHandler(hub, chatService).RegisterRoutes(ws, customer, support)
	notification.NewHandler(notificationStore, notifier).RegisterRoutes(adminGroup, customer)
	admin.NewHandler(db, staffStore).RegisterRoutes(adminGroup)
	wallet.NewHandler(walletStore).RegisterRoutes(protected, accountant)
	discount.NewHandler(discountStore, userStore).RegisterRoutes(discountReader, discountManager, customer)

	log.Println("server listening on :" + envConfig.Port)
	if err := router.Run(":" + envConfig.Port); err != nil {
		log.Fatal(err)
	}
}
