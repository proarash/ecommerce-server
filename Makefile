BINARY  := ecommerce-server
PKG     := ./cmd
BIN_DIR := bin
GO      ?= go

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | \
		awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

.PHONY: run
run: ## Run the server
	$(GO) run $(PKG)

.PHONY: dev
dev: ## Run with hot reload (air)
	@command -v air >/dev/null || { echo "air not installed: make tools"; exit 1; }
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
