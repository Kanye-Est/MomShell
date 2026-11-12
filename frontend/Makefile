.PHONY: dev build preview lint typecheck check install clean docker-build docker-run help

CYAN  := \033[36m
GREEN := \033[32m
RESET := \033[0m

##@ Development

dev: ## Start Vite dev server
	@echo "$(CYAN)Starting dev server on http://localhost:5173$(RESET)"
	npx vite

preview: build ## Preview production build locally
	@echo "$(CYAN)Starting preview server...$(RESET)"
	npx vite preview

##@ Code Quality

lint: ## Run ESLint
	@echo "$(CYAN)Linting...$(RESET)"
	npx eslint .
	@echo "$(GREEN)Lint passed$(RESET)"

typecheck: ## Run vue-tsc type checker
	@echo "$(CYAN)Type checking...$(RESET)"
	npx vue-tsc --noEmit -p tsconfig.node.json
	@echo "$(GREEN)Typecheck passed$(RESET)"

check: lint typecheck ## Run all checks (lint + typecheck)
	@echo "$(GREEN)All checks passed$(RESET)"

##@ Build

install: ## Install npm dependencies
	@echo "$(CYAN)Installing dependencies...$(RESET)"
	npm ci --ignore-scripts
	@echo "$(GREEN)Dependencies installed$(RESET)"

build: ## Build for production
	@echo "$(CYAN)Building...$(RESET)"
	npm run build
	@echo "$(GREEN)Build output in dist/$(RESET)"

##@ Docker

docker-build: ## Build Docker image
	@echo "$(CYAN)Building Docker image...$(RESET)"
	docker build -t momshell-frontend .
	@echo "$(GREEN)Image built: momshell-frontend$(RESET)"

docker-run: ## Run Docker container
	@echo "$(CYAN)Starting container on http://localhost:80$(RESET)"
	docker run --rm --add-host=host.docker.internal:host-gateway -p 80:80 momshell-frontend

##@ Cleanup

clean: ## Remove build output and caches
	@echo "$(CYAN)Cleaning...$(RESET)"
	rm -rf dist node_modules/.cache
	@echo "$(GREEN)Cleaned$(RESET)"

##@ Help

help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "\n$(CYAN)Usage:$(RESET)\n  make $(GREEN)<target>$(RESET)\n"} \
		/^[a-zA-Z_-]+:.*?##/ { printf "  $(GREEN)%-16s$(RESET) %s\n", $$1, $$2 } \
		/^##@/ { printf "\n$(CYAN)%s$(RESET)\n", substr($$0, 5) }' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help
