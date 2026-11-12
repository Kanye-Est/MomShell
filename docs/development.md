# Development Guide

## Prerequisites

- Go 1.25+
- Node.js 24+
- PostgreSQL
- Git
- pre-commit (optional, for git hooks)

See [Getting Started](getting-started.md) for installation links.

## Setup

### Automated (Recommended)

```bash
git lfs install
git clone https://github.com/Kanye-Est/MomShell.git
cd MomShell
./scripts/dev-setup.sh
```

### Manual

```bash
cp .env.example .env   # Edit with your config

cd backend && go mod download && cd ..
cd frontend && npm ci --ignore-scripts && cd ..

# Optional: install git hooks
pre-commit install
```

## Running Locally

Start the database configured in the project `.env` first.

### Using Make

```bash
make dev-backend    # Terminal 1 — Go server on :8000
make dev-frontend   # Terminal 2 — Vite dev server on :5173
make dev-tmux       # Or both in tmux
```

### Manual Commands

```bash
# Backend
cd backend && go run cmd/server/main.go

# Frontend
cd frontend && npm run dev
```

## Common Commands

```bash
make lint          # go vet + eslint
make format        # go fmt
make typecheck     # go build + vue-tsc
make check         # lint + typecheck

make build-backend   # Build Go binary
make build-frontend  # Vite production build
```

To regenerate the sand texture, run `npm run render-sand` from `frontend`. Set `PUPPETEER_EXECUTABLE_PATH` to a local Chromium executable, or first run `npx puppeteer browsers install chrome`. Pass an output path after `--` to render a separate file.

## Contributing

See [CONTRIBUTING.md](../CONTRIBUTING.md) for code standards, commit conventions, and PR workflow.

---

[Back to Documentation Index](README.md) | [Back to main README](../README.md)
