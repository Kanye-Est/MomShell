#Requires -Version 5.1
<#
.SYNOPSIS
    MomShell Development Environment Setup (Windows)
.DESCRIPTION
    Sets up PostgreSQL, environment variables, Go backend, and Vue frontend.
.USAGE
    powershell -ExecutionPolicy Bypass -File scripts\dev-setup.ps1
#>

$ErrorActionPreference = "Stop"

# ── Helpers ──────────────────────────────────────────────────────────

function Write-Ok    { param($msg) Write-Host "[ OK ] $msg" -ForegroundColor Green }
function Write-Info  { param($msg) Write-Host "[INFO] $msg" -ForegroundColor Cyan }
function Write-Warn  { param($msg) Write-Host "[WARN] $msg" -ForegroundColor Yellow }
function Write-Fail  { param($msg) Write-Host "[FAIL] $msg" -ForegroundColor Red }

function Test-Command { param($cmd) $null -ne (Get-Command $cmd -ErrorAction SilentlyContinue) }

function Read-Default {
    param([string]$Prompt, [string]$Default)
    if ($Default) {
        $input = Read-Host "  $Prompt [$Default]"
    } else {
        $input = Read-Host "  $Prompt"
    }
    if ([string]::IsNullOrWhiteSpace($input)) { return $Default } else { return $input }
}

# Detect if winget or choco is available
$PM = $null
if (Test-Command "winget") { $PM = "winget" }
elseif (Test-Command "choco") { $PM = "choco" }

function Try-Install {
    param([string]$Cmd, [string]$WingetId, [string]$ChocoId, [string]$DisplayName, [string]$ManualUrl)
    if (-not $PM) {
        Write-Warn "No package manager (winget/choco) found. Install $DisplayName manually: $ManualUrl"
        return $false
    }
    $answer = Read-Host "  Install $DisplayName via ${PM}? [Y/n]"
    if ($answer -match '^[nN]') { return $false }
    Write-Info "Installing $DisplayName..."
    if ($PM -eq "winget") {
        winget install --id $WingetId -e --accept-package-agreements --accept-source-agreements
    } else {
        choco install $ChocoId -y
    }
    # Refresh PATH so newly installed commands are found
    $env:Path = [System.Environment]::GetEnvironmentVariable("Path", "Machine") + ";" +
                [System.Environment]::GetEnvironmentVariable("Path", "User")
    if (Test-Command $Cmd) {
        Write-Ok "$DisplayName installed"
        return $true
    } else {
        Write-Fail "Installation may require restarting the terminal"
        return $false
    }
}

# ── Resolve project root ─────────────────────────────────────────────

$ScriptDir  = Split-Path -Parent $MyInvocation.MyCommand.Definition
$ProjectRoot = Split-Path -Parent $ScriptDir
Set-Location $ProjectRoot

# ── Banner ────────────────────────────────────────────────────────────

Write-Host ""
Write-Host "+==============================================+" -ForegroundColor Cyan
Write-Host "|       MomShell Development Setup (Windows)   |" -ForegroundColor Cyan
Write-Host "+==============================================+" -ForegroundColor Cyan
Write-Host ""

# ============================================
# 1. Check dependencies
# ============================================
Write-Host "=== 1. Check System Dependencies ===" -ForegroundColor Cyan

$Failed = $false

# --- Go ---
if (Test-Command "go") {
    $goVer = (go version) -replace 'go version ',''
    Write-Ok "Go $goVer"
} else {
    Write-Fail "Go not installed"
    if (-not (Try-Install "go" "GoLang.Go" "golang" "Go" "https://go.dev/dl/")) { $Failed = $true }
}

# --- Node ---
if (Test-Command "node") {
    Write-Ok "Node $(node -v)"
} else {
    Write-Fail "Node not installed"
    if (-not (Try-Install "node" "OpenJS.NodeJS.LTS" "nodejs-lts" "Node.js" "https://nodejs.org/")) { $Failed = $true }
}

# --- npm ---
if (Test-Command "npm") {
    Write-Ok "npm $(npm -v)"
} else {
    Write-Fail "npm not installed (should come with Node)"
    $Failed = $true
}

# --- git ---
if (Test-Command "git") {
    $gitVer = (git --version) -replace 'git version ',''
    Write-Ok "git $gitVer"
} else {
    Write-Fail "git not installed"
    if (-not (Try-Install "git" "Git.Git" "git" "Git" "https://git-scm.com/")) { $Failed = $true }
}

# --- PostgreSQL (psql) ---
if (Test-Command "psql") {
    Write-Ok "PostgreSQL client installed"
} else {
    Write-Fail "psql not installed"
    if (-not (Try-Install "psql" "PostgreSQL.PostgreSQL" "postgresql" "PostgreSQL" "https://www.postgresql.org/download/")) { $Failed = $true }
}

if ($Failed) {
    Write-Host ""
    Write-Fail "Some required dependencies are missing. Install them and re-run this script."
    exit 1
}

# ============================================
# 2. PostgreSQL
# ============================================
Write-Host ""
Write-Host "=== 2. Setup PostgreSQL ===" -ForegroundColor Cyan

$DB_NAME = "momshell"
$DB_USER = "momshell"
$DB_PASS = "momshell"

$envFile = Join-Path $ProjectRoot ".env"
if ((Test-Path $envFile -PathType Leaf) -or $env:DATABASE_URL) {
    Write-Ok "Using the configured database"
} else {
    # Check if PostgreSQL is running
    $pgReady = $false
    try {
        $null = pg_isready 2>$null
        if ($LASTEXITCODE -eq 0) { $pgReady = $true }
    } catch {}

    if ($pgReady) {
        Write-Ok "PostgreSQL is running"
    } else {
        Write-Warn "PostgreSQL is not running, attempting to start..."
        # Try starting via Windows service
        $pgService = Get-Service -Name "postgresql*" -ErrorAction SilentlyContinue | Select-Object -First 1
        if ($pgService) {
            try {
                Start-Service $pgService.Name -ErrorAction Stop
                Start-Sleep -Seconds 2
                Write-Ok "PostgreSQL service '$($pgService.Name)' started"
            } catch {
                Write-Fail "Could not start PostgreSQL service. Run this script as Administrator, or start PostgreSQL manually."
                Write-Host "  Try: Start-Service $($pgService.Name)" -ForegroundColor Yellow
                exit 1
            }
        } else {
            # Try pg_ctl with default data directory
            $pgData = $null
            if ($env:PGDATA) { $pgData = $env:PGDATA }
            elseif (Test-Path "$env:PROGRAMFILES\PostgreSQL") {
                $pgDir = Get-ChildItem "$env:PROGRAMFILES\PostgreSQL" -Directory | Sort-Object Name -Descending | Select-Object -First 1
                if ($pgDir -and (Test-Path "$($pgDir.FullName)\data")) {
                    $pgData = "$($pgDir.FullName)\data"
                }
            }
            if ($pgData -and (Test-Command "pg_ctl")) {
                Write-Info "Starting PostgreSQL with pg_ctl..."
                pg_ctl start -D $pgData -l "$pgData\server.log"
                Start-Sleep -Seconds 3
            } else {
                Write-Fail "Could not find or start PostgreSQL. Please start it manually."
                Write-Host "  Common locations: C:\Program Files\PostgreSQL\<version>\data" -ForegroundColor Yellow
                exit 1
            }
        }

        # Verify it's now running
        try {
            $null = pg_isready 2>$null
            if ($LASTEXITCODE -ne 0) { throw "not ready" }
            Write-Ok "PostgreSQL is running"
        } catch {
            Write-Fail "PostgreSQL still not responding. Please start it manually and re-run."
            exit 1
        }
    }

    # On Windows, PostgreSQL installer creates a 'postgres' superuser by default.
    # Connect as postgres to create the app user/database.
    $env:PGPASSWORD = ""

    # Helper to run psql as the postgres superuser
    function Invoke-PgSuperuser {
        param([string]$Sql)
        # Try connecting as postgres user (default installer superuser)
        $env:PGPASSWORD = "postgres"
        $result = psql -w -h localhost -U postgres -tAc $Sql 2>$null
        if ($LASTEXITCODE -ne 0) {
            # Try without password (peer/trust auth)
            $env:PGPASSWORD = ""
            $result = psql -w -h localhost -U postgres -tAc $Sql 2>$null
        }
        return $result
    }

    function Invoke-PgSuperuserCmd {
        param([string]$Sql)
        $env:PGPASSWORD = "postgres"
        psql -w -h localhost -U postgres -c $Sql 2>$null
        if ($LASTEXITCODE -ne 0) {
            $env:PGPASSWORD = ""
            psql -w -h localhost -U postgres -c $Sql 2>$null
        }
    }

    # Create user
    $userExists = Invoke-PgSuperuser "SELECT 1 FROM pg_roles WHERE rolname='$DB_USER'"
    if ($userExists -match "1") {
        Write-Ok "Database user '$DB_USER' exists"
    } else {
        Write-Info "Creating database user '$DB_USER'..."
        Invoke-PgSuperuserCmd "CREATE USER $DB_USER WITH PASSWORD '$DB_PASS';"
        if ($LASTEXITCODE -eq 0) {
            Write-Ok "Database user '$DB_USER' created"
        } else {
            Write-Warn "Could not create user. You may need to set the postgres superuser password."
            Write-Host "  Try: psql -w -h localhost -U postgres -c `"CREATE USER $DB_USER WITH PASSWORD '$DB_PASS';`"" -ForegroundColor Yellow
        }
    }

    # Create database
    $dbExists = Invoke-PgSuperuser "SELECT 1 FROM pg_database WHERE datname='$DB_NAME'"
    if ($dbExists -match "1") {
        Write-Ok "Database '$DB_NAME' exists"
    } else {
        Write-Info "Creating database '$DB_NAME'..."
        Invoke-PgSuperuserCmd "CREATE DATABASE $DB_NAME OWNER $DB_USER;"
        if ($LASTEXITCODE -eq 0) {
            Write-Ok "Database '$DB_NAME' created"
        } else {
            Write-Warn "Could not create database. Create it manually."
            Write-Host "  Try: psql -w -h localhost -U postgres -c `"CREATE DATABASE $DB_NAME OWNER $DB_USER;`"" -ForegroundColor Yellow
        }
    }

    # Test connection
    $env:PGPASSWORD = $DB_PASS
    try {
        $null = psql -w -h localhost -U $DB_USER -d $DB_NAME -c "SELECT 1" 2>$null
        if ($LASTEXITCODE -eq 0) {
            Write-Ok "Database connection verified"
        } else {
            throw "connection failed"
        }
    } catch {
        Write-Warn "Could not connect via localhost. Check pg_hba.conf allows password auth."
        Write-Host "  Add to pg_hba.conf: host all all 127.0.0.1/32 md5" -ForegroundColor Yellow
    }
    $env:PGPASSWORD = ""
}

# ============================================
# 3. Environment variables
# ============================================
Write-Host ""
Write-Host "=== 3. Configure Environment Variables ===" -ForegroundColor Cyan

if (Test-Path $envFile) {
    Write-Ok ".env already exists, skipping interactive config"
} else {
    Write-Info "Creating .env -- press Enter to accept defaults, leave blank to skip"
    Write-Host ""

    # Database
    Write-Host "  -- Database --" -ForegroundColor Cyan
    $defaultDbUrl = $env:DATABASE_URL
    if (-not $defaultDbUrl) { $defaultDbUrl = "postgres://${DB_USER}:${DB_PASS}@localhost:5432/${DB_NAME}?sslmode=disable" }
    $DATABASE_URL = Read-Default "DATABASE_URL" $defaultDbUrl

    # JWT
    Write-Host "  -- JWT --" -ForegroundColor Cyan
    # Generate random hex string for JWT secret
    $jwtBytes = New-Object byte[] 32
    [System.Security.Cryptography.RandomNumberGenerator]::Create().GetBytes($jwtBytes)
    $defaultJwt = ($jwtBytes | ForEach-Object { $_.ToString("x2") }) -join ''
    $JWT_SECRET_KEY = Read-Default "JWT_SECRET_KEY" $defaultJwt
    $JWT_ACCESS_TOKEN_EXPIRE_MINUTES = Read-Default "JWT_ACCESS_TOKEN_EXPIRE_MINUTES" "30"
    $JWT_REFRESH_TOKEN_EXPIRE_DAYS = Read-Default "JWT_REFRESH_TOKEN_EXPIRE_DAYS" "7"

    # OpenAI
    Write-Host "  -- OpenAI Compatible API --" -ForegroundColor Cyan
    $OPENAI_API_KEY = Read-Default "OPENAI_API_KEY" ""
    $OPENAI_BASE_URL = Read-Default "OPENAI_BASE_URL" "https://api-inference.modelscope.cn/v1"
    $OPENAI_MODEL = Read-Default "OPENAI_MODEL" "Qwen/Qwen3-235B-A22B"

    # Server
    Write-Host "  -- Server --" -ForegroundColor Cyan
    $PORT = Read-Default "PORT" "8000"

    # Admin
    Write-Host "  -- Initial Admin (optional) --" -ForegroundColor Cyan
    $ADMIN_USERNAME = Read-Default "ADMIN_USERNAME" ""
    $ADMIN_EMAIL = Read-Default "ADMIN_EMAIL" ""
    $ADMIN_PASSWORD = Read-Default "ADMIN_PASSWORD" ""

    # Write .env
    $envContents = @"
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

# ==================== Server ====================
PORT=$PORT

# ==================== Initial Admin (optional, created on first startup) ====================
ADMIN_USERNAME=$ADMIN_USERNAME
ADMIN_EMAIL=$ADMIN_EMAIL
ADMIN_PASSWORD=$ADMIN_PASSWORD
"@
    [System.IO.File]::WriteAllText($envFile, $envContents + [Environment]::NewLine, [System.Text.UTF8Encoding]::new($false))

    Write-Host ""
    Write-Ok ".env created"
}

# ============================================
# 4. Backend (Go)
# ============================================
Write-Host ""
Write-Host "=== 4. Initialize Backend (Go) ===" -ForegroundColor Cyan

Write-Info "Downloading Go dependencies..."
Push-Location (Join-Path $ProjectRoot "backend")
go mod download
if ($LASTEXITCODE -ne 0) {
    Write-Fail "Failed to download Go dependencies"
    Pop-Location
    exit 1
}
Pop-Location
Write-Ok "Backend dependencies installed"

# ============================================
# 5. Frontend (Vue)
# ============================================
Write-Host ""
Write-Host "=== 5. Initialize Frontend (Vue) ===" -ForegroundColor Cyan

Write-Info "Installing npm dependencies..."
Push-Location (Join-Path $ProjectRoot "frontend")
npm ci --ignore-scripts
if ($LASTEXITCODE -ne 0) {
    Write-Fail "npm ci failed"
    Pop-Location
    exit 1
}
Pop-Location
Write-Ok "Frontend dependencies installed"

# ============================================
# 6. Pre-commit hooks
# ============================================
Write-Host ""
Write-Host "=== 6. Install Pre-commit Hooks ===" -ForegroundColor Cyan

if (Test-Command "pre-commit") {
    Set-Location $ProjectRoot
    pre-commit install
    Write-Ok "Pre-commit hooks installed"
} else {
    Write-Warn "pre-commit not installed"
    if (Test-Command "pip") {
        $answer = Read-Host "  Install pre-commit via pip? [Y/n]"
        if ($answer -notmatch '^[nN]') {
            Write-Info "Installing pre-commit..."
            pip install pre-commit 2>$null
            if (Test-Command "pre-commit") {
                Set-Location $ProjectRoot
                pre-commit install
                Write-Ok "Pre-commit hooks installed"
            } else {
                Write-Warn "Installation failed, skipping hook setup"
            }
        } else {
            Write-Warn "Skipping hook setup"
        }
    } else {
        Write-Warn "pip not found, skipping hook setup"
        Write-Host "  Install: pip install pre-commit; pre-commit install" -ForegroundColor Yellow
    }
}

# ============================================
# 7. Verify
# ============================================
Write-Host ""
Write-Host "=== 7. Verify ===" -ForegroundColor Cyan

Push-Location (Join-Path $ProjectRoot "backend")
$buildResult = go build ./... 2>&1
if ($LASTEXITCODE -eq 0) {
    Write-Ok "Backend build passed"
} else {
    Write-Warn "Backend build failed (database config may be missing)"
}
Pop-Location

if (Test-Path (Join-Path $ProjectRoot "frontend\node_modules")) {
    Write-Ok "Frontend node_modules installed"
} else {
    Write-Warn "Frontend node_modules not found"
}

# ============================================
# Done
# ============================================
Write-Host ""
Write-Host "+==============================================+" -ForegroundColor Green
Write-Host "|            Setup Complete!                   |" -ForegroundColor Green
Write-Host "+==============================================+" -ForegroundColor Green
Write-Host ""
Write-Host "Next steps:"
Write-Host ""
Write-Host "  1. Start development servers:"
Write-Host "     make dev-backend    # Backend  http://localhost:8000"
Write-Host "     make dev-frontend   # Frontend http://localhost:5173"
Write-Host ""
Write-Host "  2. Admin panel: http://localhost:8000/admin"
Write-Host ""
Write-Host "  3. Edit .env anytime to update configuration"
Write-Host ""
