#!/usr/bin/env bash
#
# MomShell Development Environment Setup
# Usage: ./scripts/dev-setup.sh
#

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

info()    { echo -e "${BLUE}[INFO]${NC} $1"; }
success() { echo -e "${GREEN}[ OK ]${NC} $1"; }
warn()    { echo -e "${YELLOW}[WARN]${NC} $1"; }
fail()    { echo -e "${RED}[FAIL]${NC} $1"; }

check() { command -v "$1" &>/dev/null; }

# Detect package manager
detect_pm() {
    if check pacman; then
        PM="pacman"; PM_INSTALL="sudo pacman -S --noconfirm"
    elif check apt-get; then
        PM="apt"; PM_INSTALL="sudo apt-get install -y"
    elif check dnf; then
        PM="dnf"; PM_INSTALL="sudo dnf install -y"
    elif check brew; then
        PM="brew"; PM_INSTALL="brew install"
    else
        PM=""; PM_INSTALL=""
    fi
}

# Ask to install a package; $1=command to verify, $2=package name(s), $3=display name
try_install() {
    local cmd="$1" pkg="$2" name="${3:-$1}"
    if [[ -n "$PM_INSTALL" ]]; then
        echo -en "  Install ${name} via ${PM}? ${YELLOW}[Y/n]${NC}: "
        read -r ans
        case "$ans" in
            [nN]*) return 1 ;;
            *) ;;
        esac
        info "Installing ${name}..."
        $PM_INSTALL $pkg
        if check "$cmd"; then
            success "${name} installed"
            return 0
        else
            fail "Installation failed"
            return 1
        fi
    else
        warn "No supported package manager found"
        return 1
    fi
}

detect_pm

echo -e "${BLUE}"
echo "╔════════════════════════════════════════════╗"
echo "║       MomShell Development Setup           ║"
echo "╚════════════════════════════════════════════╝"
echo -e "${NC}"

# ============================================
# 1. Check dependencies
# ============================================
echo -e "${BLUE}=== 1. Check System Dependencies ===${NC}"

FAILED=0

# --- Go ---
if check go; then
    success "Go $(go version | awk '{print $3}')"
else
    fail "Go not installed"
    case "$PM" in
        pacman) try_install "go" "go" "Go" || FAILED=1 ;;
        apt)    try_install "go" "golang-go" "Go" || FAILED=1 ;;
        dnf)    try_install "go" "golang" "Go" || FAILED=1 ;;
        brew)   try_install "go" "go" "Go" || FAILED=1 ;;
        *)      echo "  Install manually: https://go.dev/dl/"; FAILED=1 ;;
    esac
fi

# --- Node + npm (via nvm or package manager) ---
if check node; then
    success "Node $(node -v)"
else
    fail "Node not installed"
    if check nvm; then
        echo -en "  Install Node via nvm? ${YELLOW}[Y/n]${NC}: "
        read -r ans
        case "$ans" in
            [nN]*) FAILED=1 ;;
            *)
                info "Installing Node via nvm..."
                nvm install "$(cat "$PROJECT_ROOT/frontend/.nvmrc")"
                check node && success "Node $(node -v) installed" || FAILED=1
                ;;
        esac
    else
        case "$PM" in
            pacman) try_install "node" "nodejs npm" "Node.js" || FAILED=1 ;;
            apt)    try_install "node" "nodejs npm" "Node.js" || FAILED=1 ;;
            dnf)    try_install "node" "nodejs npm" "Node.js" || FAILED=1 ;;
            brew)   try_install "node" "node" "Node.js" || FAILED=1 ;;
            *)      echo "  Install manually: https://nodejs.org/ or use nvm"; FAILED=1 ;;
        esac
    fi
fi

if check npm; then
    success "npm $(npm -v)"
else
    fail "npm not installed (should come with Node)"
    FAILED=1
fi

# --- git ---
if check git; then
    success "git $(git --version | awk '{print $3}')"
else
    fail "git not installed"
    case "$PM" in
        pacman) try_install "git" "git" || FAILED=1 ;;
        apt)    try_install "git" "git" || FAILED=1 ;;
        dnf)    try_install "git" "git" || FAILED=1 ;;
        brew)   try_install "git" "git" || FAILED=1 ;;
        *)      echo "  Install manually: https://git-scm.com/"; FAILED=1 ;;
    esac
fi

# --- PostgreSQL ---
if check psql; then
    success "PostgreSQL client installed"
else
    fail "psql not installed"
    case "$PM" in
        pacman) try_install "psql" "postgresql" "PostgreSQL" || FAILED=1 ;;
        apt)    try_install "psql" "postgresql postgresql-client" "PostgreSQL" || FAILED=1 ;;
        dnf)    try_install "psql" "postgresql-server postgresql" "PostgreSQL" || FAILED=1 ;;
        brew)   try_install "psql" "postgresql" "PostgreSQL" || FAILED=1 ;;
        *)      echo "  Install manually: https://www.postgresql.org/download/"; FAILED=1 ;;
    esac
fi

if [[ "$FAILED" -ne 0 ]]; then
    echo ""
    fail "Some required dependencies are missing. Please install them and re-run this script."
    exit 1
fi

# ============================================
# 2. PostgreSQL
# ============================================
echo ""
echo -e "${BLUE}=== 2. Setup PostgreSQL ===${NC}"

DB_NAME="momshell"
DB_USER="momshell"
DB_PASS="momshell"

if [[ -f .env || -n "${DATABASE_URL:-}" ]]; then
    success "Using the configured database"
else
    # Check if PostgreSQL service is running
    if pg_isready -h localhost -q 2>/dev/null; then
        success "PostgreSQL is running"
    else
        warn "PostgreSQL is not running, attempting to start..."
        if [[ "$OSTYPE" == darwin* ]]; then
            brew services start postgresql 2>/dev/null || brew services start "$(brew list --formula | grep -m1 '^postgresql@')" 2>/dev/null || true
        else
            sudo systemctl start postgresql 2>/dev/null || true
        fi
        if pg_isready -h localhost -q 2>/dev/null; then
            success "PostgreSQL started"
        else
            fail "Could not start PostgreSQL"
            if [[ "$OSTYPE" == darwin* ]]; then
                echo "  Try: brew services start postgresql"
            else
                echo "  Try: sudo systemctl start postgresql"
                echo "  Arch first-time: sudo -u postgres initdb -D /var/lib/postgres/data"
            fi
            exit 1
        fi
    fi

    # Determine how to run psql as superuser
    # macOS Homebrew: psql runs as the current user who is already a superuser
    # Linux: need sudo -u postgres
    if [[ "$OSTYPE" == darwin* ]]; then
        PG_SUDO="psql -d postgres"
    else
        PG_SUDO="sudo -u postgres psql"
    fi

    # Create database user if not exists
    if $PG_SUDO -tAc "SELECT 1 FROM pg_roles WHERE rolname='$DB_USER'" 2>/dev/null | grep -q 1; then
        success "Database user '$DB_USER' exists"
    else
        info "Creating database user '$DB_USER'..."
        $PG_SUDO -c "CREATE USER $DB_USER WITH PASSWORD '$DB_PASS';" 2>/dev/null
        success "Database user '$DB_USER' created"
    fi

    # Create database if not exists
    if $PG_SUDO -tAc "SELECT 1 FROM pg_database WHERE datname='$DB_NAME'" 2>/dev/null | grep -q 1; then
        success "Database '$DB_NAME' exists"
    else
        info "Creating database '$DB_NAME'..."
        $PG_SUDO -c "CREATE DATABASE $DB_NAME OWNER $DB_USER;" 2>/dev/null
        success "Database '$DB_NAME' created"
    fi

    # Test connection
    if PGPASSWORD="$DB_PASS" psql -h localhost -U "$DB_USER" -d "$DB_NAME" -c "SELECT 1" &>/dev/null; then
        success "Database connection verified"
    else
        warn "Could not connect via localhost, check pg_hba.conf allows password auth"
        echo "  Check the existing role password or enter your connection URL in the next step."
    fi
fi

# ============================================
# 3. Environment variables
# ============================================
echo ""
echo -e "${BLUE}=== 3. Configure Environment Variables ===${NC}"

# Helper: prompt for a value with a default; empty input keeps the default
# Usage: ask "PROMPT" "DEFAULT" -> sets REPLY
ask() {
    local prompt="$1" default="$2"
    if [[ -n "$default" ]]; then
        echo -en "  ${prompt} ${YELLOW}[${default}]${NC}: "
    else
        echo -en "  ${prompt}: "
    fi
    read -r REPLY
    [[ -z "$REPLY" ]] && REPLY="$default" || true
}

if [[ -f .env ]]; then
    success ".env already exists, skipping interactive config"
else
    info "Creating .env — press Enter to accept defaults, leave blank to skip"
    echo ""

    # Database (auto-filled from step 2)
    echo -e "  ${BLUE}-- Database --${NC}"
    DATABASE_URL="${DATABASE_URL:-postgres://$DB_USER:$DB_PASS@localhost:5432/$DB_NAME?sslmode=disable}"
    ask "DATABASE_URL" "$DATABASE_URL"
    DATABASE_URL="$REPLY"

    # JWT
    echo -e "  ${BLUE}-- JWT --${NC}"
    JWT_SECRET=$(openssl rand -hex 32 2>/dev/null || head -c 64 /dev/urandom | xxd -p | tr -d '\n' | head -c 64)
    ask "JWT_SECRET_KEY" "$JWT_SECRET"
    JWT_SECRET_KEY="$REPLY"

    ask "JWT_ACCESS_TOKEN_EXPIRE_MINUTES" "30"
    JWT_ACCESS_TOKEN_EXPIRE_MINUTES="$REPLY"

    ask "JWT_REFRESH_TOKEN_EXPIRE_DAYS" "7"
    JWT_REFRESH_TOKEN_EXPIRE_DAYS="$REPLY"

    # OpenAI
    echo -e "  ${BLUE}-- OpenAI Compatible API --${NC}"
    ask "OPENAI_API_KEY" ""
    OPENAI_API_KEY="$REPLY"

    ask "OPENAI_BASE_URL" "https://api-inference.modelscope.cn/v1"
    OPENAI_BASE_URL="$REPLY"

    ask "OPENAI_MODEL" "Qwen/Qwen3-235B-A22B"
    OPENAI_MODEL="$REPLY"

    ask "IMAGE_MODEL (image generation)" "Tongyi-MAI/Z-Image-Turbo"
    IMAGE_MODEL="$REPLY"

    # Firecrawl
    echo -e "  ${BLUE}-- Firecrawl (Web Search) --${NC}"
    ask "FIRECRAWL_API_KEY" ""
    FIRECRAWL_API_KEY="$REPLY"

    # Server
    echo -e "  ${BLUE}-- Server --${NC}"
    ask "PORT" "8000"
    PORT="$REPLY"

    # Admin
    echo -e "  ${BLUE}-- Initial Admin (optional) --${NC}"
    ask "ADMIN_USERNAME" ""
    ADMIN_USERNAME="$REPLY"

    ask "ADMIN_EMAIL" ""
    ADMIN_EMAIL="$REPLY"

    ask "ADMIN_PASSWORD" ""
    ADMIN_PASSWORD="$REPLY"

    # Write .env
    (umask 077 && cat > .env) <<ENVEOF
# MomShell Environment Configuration

# ==================== Database ====================
POSTGRES_USER=$DB_USER
POSTGRES_PASSWORD=$DB_PASS
POSTGRES_DB=$DB_NAME
DATABASE_URL=$DATABASE_URL

# ==================== JWT ====================
JWT_SECRET_KEY=$JWT_SECRET_KEY
JWT_ACCESS_TOKEN_EXPIRE_MINUTES=$JWT_ACCESS_TOKEN_EXPIRE_MINUTES
JWT_REFRESH_TOKEN_EXPIRE_DAYS=$JWT_REFRESH_TOKEN_EXPIRE_DAYS

# ==================== OpenAI Compatible API ====================
OPENAI_API_KEY=$OPENAI_API_KEY
OPENAI_BASE_URL=$OPENAI_BASE_URL
OPENAI_MODEL=$OPENAI_MODEL
IMAGE_MODEL=$IMAGE_MODEL

# ==================== Firecrawl (Web Search) ====================
FIRECRAWL_API_KEY=$FIRECRAWL_API_KEY

# ==================== Server ====================
PORT=$PORT

# ==================== Initial Admin (optional, created on first startup) ====================
ADMIN_USERNAME=$ADMIN_USERNAME
ADMIN_EMAIL=$ADMIN_EMAIL
ADMIN_PASSWORD=$ADMIN_PASSWORD
ENVEOF

    echo ""
    success ".env created"
fi

# ============================================
# 4. Backend (Go)
# ============================================
echo ""
echo -e "${BLUE}=== 4. Initialize Backend (Go) ===${NC}"

info "Downloading Go dependencies..."
(cd "$PROJECT_ROOT/backend" && go mod download)
success "Backend dependencies installed"

# ============================================
# 5. Frontend (Vue)
# ============================================
echo ""
echo -e "${BLUE}=== 5. Initialize Frontend (Vue) ===${NC}"

info "Installing npm dependencies..."
(cd "$PROJECT_ROOT/frontend" && npm ci --ignore-scripts)
success "Frontend dependencies installed"

# ============================================
# 6. Pre-commit hooks
# ============================================
echo ""
echo -e "${BLUE}=== 6. Install Pre-commit Hooks ===${NC}"

if check pre-commit; then
    pre-commit install
    success "Pre-commit hooks installed"
else
    warn "pre-commit not installed"
    if [[ -t 0 ]] && check pip; then
        echo -en "  Install pre-commit via pip? ${YELLOW}[Y/n]${NC}: "
        read -r ans
        case "$ans" in
            [nN]*) warn "Skipping hook setup" ;;
            *)
                info "Installing pre-commit..."
                pip install --break-system-packages pre-commit 2>/dev/null || pip install pre-commit
                if check pre-commit; then
                    pre-commit install
                    success "Pre-commit hooks installed"
                else
                    warn "Installation failed, skipping hook setup"
                fi
                ;;
        esac
    else
        warn "Skipping optional hook setup"
        echo "  Install: pip install pre-commit && pre-commit install"
    fi
fi

# ============================================
# 7. Verify
# ============================================
echo ""
echo -e "${BLUE}=== 7. Verify ===${NC}"

(cd "$PROJECT_ROOT/backend" && go build ./...)
success "Backend build passed"
(cd "$PROJECT_ROOT/frontend" && npm run typecheck)
success "Frontend type check passed"

# ============================================
# Done
# ============================================
echo ""
echo -e "${GREEN}"
echo "╔════════════════════════════════════════════╗"
echo "║            Setup Complete!                 ║"
echo "╚════════════════════════════════════════════╝"
echo -e "${NC}"
echo "Next steps:"
echo ""
echo "  1. Start development servers:"
echo "     make dev-backend    # Backend  http://localhost:8000"
echo "     make dev-frontend   # Frontend http://localhost:5173"
echo "     make dev-tmux       # Or both in tmux"
echo ""
echo "  2. Admin panel: http://localhost:8000/admin"
echo ""
echo "  3. Edit .env anytime to update configuration"
echo ""
