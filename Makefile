BINARY  := ecommerce-server
PKG     := ./cmd
BIN_DIR := bin
GO      ?= go

# Default dev environment variables
export ENV                         ?= dev
export PORT                        ?= 4000
export DOMAIN                      ?= localhost
export DSN                         ?= host=localhost user=postgres password=postgres dbname=ecommerce port=5432 sslmode=disable TimeZone=Asia/Tehran
export JWT_SECRET                  ?= change-me
export ADMIN_NAME                  ?= Admin
export ADMIN_MOBILE                ?= 09120000000
export ADMIN_PASSWORD              ?= change-me
export TELEGRAM_BOT_TOKEN          ?=
export TELEGRAM_ADMIN_CHAT_ID      ?=
export MINIO_ENDPOINT              ?= localhost:9000
export MINIO_ACCESS_KEY            ?= minioadmin
export MINIO_SECRET_KEY            ?= minioadmin
export MINIO_BUCKET                ?= ecommerce
export MINIO_USE_SSL               ?= false
export MINIO_PUBLIC_URL            ?=
export ZIBAL_MERCHANT              ?= zibal
export ZIBAL_CALLBACK_URL          ?= http://localhost:4000/payment/callback
export CLIENT_PAYMENT_REDIRECT_URL ?= http://localhost:3000/payment/result
export CORS_ORIGINS                ?= http://localhost:3000

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | \
		awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

.PHONY: run
run: ## Run the server with default/exported env vars
	$(GO) run $(PKG)

.PHONY: dev
dev: ## Run with hot reload (air) using default/exported env vars
	@command -v air >/dev/null || { echo "air not installed: please install air"; exit 1; }
	air

.PHONY: build
build: ## Build the binary into bin/
	$(GO) build -v -x -trimpath -ldflags="-s -w" -o $(BIN_DIR)/$(BINARY) $(PKG)

.PHONY: test
test: ## Run tests
	$(GO) test ./...

.PHONY: fmt
fmt: ## Format the code
	$(GO) fmt ./...

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf $(BIN_DIR) tmp coverage.out

.PHONY: swag
swag: ## Generate Swagger docs
	swag init -g cmd/main.go -o ./docs --parseInternal --parseDependency --parseDepth 2
