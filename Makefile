.PHONY: install install-backend install-frontend dev dev-backend dev-frontend dev-tmux \
        lint lint-backend lint-frontend format format-backend format-frontend \
        typecheck typecheck-backend typecheck-frontend check build-frontend build-backend \
        docker-up docker-down docker-logs docker-build \
        postgres-up postgres-down postgres-logs db-reset deps-lock deps-update clean clean-all help

# Database config (must match dev-setup.sh / .env)
DB_USER ?= momshell
DB_PASS ?= momshell
DB_NAME ?= momshell
export DB_USER DB_PASS DB_NAME
DOCKER_COMPOSE = docker compose --env-file .env -f deploy/docker-compose.yml

# Colors for terminal output
CYAN := \033[36m
GREEN := \033[32m
YELLOW := \033[33m
RESET := \033[0m

##@ Setup

install: install-backend install-frontend ## Install all dependencies
	@echo "$(GREEN)All dependencies installed$(RESET)"

install-backend: ## Install backend dependencies
	@echo "$(CYAN)Installing backend dependencies...$(RESET)"
	cd backend && go mod download

install-frontend: ## Install frontend dependencies
	@echo "$(CYAN)Installing frontend dependencies...$(RESET)"
	cd frontend && npm ci --ignore-scripts

##@ Development

dev: ## Start both backend and frontend (requires tmux or run in separate terminals)
	@echo "$(YELLOW)Starting development servers...$(RESET)"
	@echo "$(CYAN)Run these commands in separate terminals:$(RESET)"
	@echo "  make dev-backend"
	@echo "  make dev-frontend"
	@echo ""
	@echo "$(CYAN)Or use: make dev-tmux (requires tmux)$(RESET)"

dev-tmux: ## Start both servers in tmux split panes
	@command -v tmux >/dev/null 2>&1 || { echo "tmux is required but not installed."; exit 1; }
	tmux new-session -d -s momshell 'make dev-backend' \; \
		split-window -h 'make dev-frontend' \; \
		attach

dev-backend: ## Start backend development server using the configured database
	@echo "$(CYAN)Starting backend server...$(RESET)"
	cd backend && go run cmd/server/main.go

dev-frontend: ## Start frontend development server
	@echo "$(CYAN)Starting frontend server on http://localhost:5173$(RESET)"
	cd frontend && npm run dev

##@ Code Quality

lint: lint-backend lint-frontend ## Run all linters
	@echo "$(GREEN)All linting passed$(RESET)"

lint-backend: ## Run backend linter (go vet)
	@echo "$(CYAN)Linting backend...$(RESET)"
	cd backend && go vet ./... && golangci-lint run ./...

lint-frontend: ## Run frontend linter (eslint)
	@echo "$(CYAN)Linting frontend...$(RESET)"
	cd frontend && npm run lint

format: format-backend ## Format all code
	@echo "$(GREEN)All code formatted$(RESET)"

format-backend: ## Format backend code (go fmt)
	@echo "$(CYAN)Formatting backend...$(RESET)"
	cd backend && go fmt ./...

typecheck: typecheck-backend typecheck-frontend ## Run all type checkers
	@echo "$(GREEN)All type checks passed$(RESET)"

typecheck-backend: ## Run backend type check (compile)
	@echo "$(CYAN)Type checking backend...$(RESET)"
	cd backend && go build ./...

typecheck-frontend: ## Run frontend type checker (vue-tsc)
	@echo "$(CYAN)Type checking frontend...$(RESET)"
	cd frontend && npm run typecheck

check: lint typecheck ## Run all checks (lint + typecheck)
	@echo "$(GREEN)All checks passed$(RESET)"

##@ Build

build-frontend: ## Build frontend for production
	@echo "$(CYAN)Building frontend...$(RESET)"
	cd frontend && npm run build

build-backend: ## Build backend binary
	@echo "$(CYAN)Building backend...$(RESET)"
	cd backend && go build -o bin/server cmd/server/main.go

##@ Docker

docker-up: ## Start all services (app + postgres)
	@echo "$(CYAN)Starting Docker containers...$(RESET)"
	$(DOCKER_COMPOSE) up -d --build --wait

docker-down: ## Stop all services
	@echo "$(CYAN)Stopping Docker containers...$(RESET)"
	$(DOCKER_COMPOSE) down

docker-logs: ## Show Docker logs
	$(DOCKER_COMPOSE) logs -f

docker-build: ## Build application Docker image
	@echo "$(CYAN)Building Docker image...$(RESET)"
	docker build -t momshell .

##@ Database

postgres-up: ## Start local PostgreSQL (systemd)
	@echo "$(CYAN)Ensuring local PostgreSQL is running on localhost:5432...$(RESET)"
	@if pg_isready -h localhost -q; then \
		echo "$(GREEN)PostgreSQL is already running$(RESET)"; \
	else \
		echo "$(YELLOW)Starting PostgreSQL via systemctl...$(RESET)"; \
		sudo systemctl start postgresql && pg_isready -h localhost -t 5; \
	fi

postgres-down: ## Stop local PostgreSQL (systemd)
	@echo "$(CYAN)Stopping local PostgreSQL...$(RESET)"
	@sudo systemctl stop postgresql
	@echo "$(GREEN)PostgreSQL stopped$(RESET)"

postgres-logs: ## Show local PostgreSQL logs
	@journalctl -u postgresql -f

db-reset: postgres-up ## Reset PostgreSQL schema (drop and recreate public schema)
	@echo "$(YELLOW)Resetting database...$(RESET)"
	PGPASSWORD="$$DB_PASS" psql -h localhost -U "$$DB_USER" -d "$$DB_NAME" -v ON_ERROR_STOP=1 -c "DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA public;"
	@echo "$(GREEN)Database reset. It will be recreated on next server start.$(RESET)"

##@ Dependencies

deps-lock: ## Sync backend dependencies
	@echo "$(CYAN)Locking dependencies...$(RESET)"
	cd backend && go mod tidy
	@echo "$(GREEN)Dependencies locked$(RESET)"

deps-update: ## Update all dependencies
	@echo "$(CYAN)Updating backend dependencies...$(RESET)"
	cd backend && go get -u ./...
	cd backend && go mod tidy
	@echo "$(CYAN)Updating frontend dependencies...$(RESET)"
	cd frontend && npm update
	@echo "$(GREEN)All dependencies updated$(RESET)"

##@ Cleanup

clean: ## Clean all caches and temporary files
	@echo "$(CYAN)Cleaning caches...$(RESET)"
	rm -rf backend/bin backend/.cache
	rm -rf frontend/node_modules/.cache
	@echo "$(GREEN)Caches cleaned$(RESET)"

clean-all: clean ## Clean everything including node_modules
	@echo "$(YELLOW)Removing node_modules...$(RESET)"
	rm -rf frontend/node_modules
	@echo "$(GREEN)All cleaned$(RESET)"

##@ Help

help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "\n$(CYAN)Usage:$(RESET)\n  make $(GREEN)<target>$(RESET)\n"} \
		/^[a-zA-Z_-]+:.*?##/ { printf "  $(GREEN)%-18s$(RESET) %s\n", $$1, $$2 } \
		/^##@/ { printf "\n$(YELLOW)%s$(RESET)\n", substr($$0, 5) }' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help
