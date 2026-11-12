# Docker Deployment

## Prerequisites

Install Docker Engine, Buildx and the Compose plugin, then start the Docker daemon. [Docker installation guide](https://docs.docker.com/engine/install/)

## Quick Start (Docker Compose)

```bash
# 1. Configure environment
cp .env.example .env
# Set POSTGRES_PASSWORD and JWT_SECRET_KEY in .env.
# Set OPENAI_API_KEY to enable AI features.

# 2. Start all services
make docker-up

# 3. Access
# App           http://localhost:7860
# Admin panel   http://localhost:7860/admin
```

## Architecture

A single Docker image contains both frontend and backend:

```
Browser → Nginx (:7860)
            ├── /            → static files (Vue SPA)
            ├── /api/        → proxy → Go backend (:8000)
            ├── /uploads/    → proxy → Go backend (:8000)
            └── /admin       → proxy → Go backend (:8000)
```

`docker-compose.yml` (in `deploy/`) runs two containers:

| Service | Image | Port |
|---------|-------|------|
| **app** | Nginx + Go binary (single image) | 7860 (published) |
| **postgres** | PostgreSQL 16 | 5432 (internal) |

The root `Dockerfile` uses multi-stage builds:
1. **frontend-builder** — Node 24, `npm ci --ignore-scripts` and `npm run build` produce `dist/`
2. **backend-builder** — Go 1.25, `go build` → binary
3. **final** — Nginx Alpine + Go binary + entrypoint script

## ModelScope Standalone Deployment

For single-container platforms like ModelScope Studios, the Dockerfile also supports embedded PostgreSQL:

```bash
docker build -t momshell .
docker run -d -p 7860:7860 --env-file .env -e DATABASE_URL= \
  -v momshell-db:/var/lib/postgresql/data \
  -v momshell-uploads:/app/uploads momshell
```

The entrypoint initializes the embedded database when `DATABASE_URL` is empty. Supply an external PostgreSQL connection string in `DATABASE_URL` to use a separate database in standalone deployments.

## Make Commands

```bash
make docker-build    # Build Docker image
make docker-up       # Start all services (app + postgres)
make docker-down     # Stop all services
make docker-logs     # View logs
```

## Configuration

Key environment variables for deployment:

| Variable | Description |
|----------|-------------|
| `DATABASE_URL` | Connection string for local development or standalone deployments. Compose configures its own PostgreSQL connection from `POSTGRES_*`. |
| `POSTGRES_USER` | PostgreSQL container user (default: momshell) |
| `POSTGRES_PASSWORD` | PostgreSQL container password (default: momshell) |
| `POSTGRES_DB` | PostgreSQL container database (default: momshell) |
| `JWT_SECRET_KEY` | Secure random secret |
| `OPENAI_API_KEY` | LLM API key |
| `PORT` | Backend server port (default: 8000, internal) |

Nginx serves the app and proxies API requests on the same origin by default.

See [Configuration](configuration.md) for the full reference.

## Data Persistence

Compose stores the database in `pgdata` and uploaded files in `uploads`. Both volumes persist across `make docker-down` and `make docker-up`.

For an older combined container, copy its `/uploads` directory before recreating the container. Restore the files into the new volume mounted at `/app/uploads`.

---

[Back to Documentation Index](README.md) | [Back to main README](../README.md)
