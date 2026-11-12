# Getting Started

## Prerequisites

- [Go 1.25+](https://go.dev/dl/)
- [Node.js 24+](https://nodejs.org/) (or via [nvm](https://github.com/nvm-sh/nvm))
- [PostgreSQL](https://www.postgresql.org/)
- Git with [Git LFS](https://git-lfs.com/) for project assets

## Quick Install

```bash
git lfs install
git clone https://github.com/Kanye-Est/MomShell.git
cd MomShell
./scripts/dev-setup.sh
```

The setup script:
- Checks prerequisites (Go, Node, npm, git)
- Prepares the local PostgreSQL database on first setup
- Creates `.env` from template with auto-generated JWT secret
- Downloads Go dependencies
- Installs npm packages
- Installs pre-commit hooks

## Start the Application

Start the PostgreSQL instance configured by `DATABASE_URL` before starting the backend. On Linux with a local systemd PostgreSQL service, `make postgres-up` starts it. Use `make install` to restore dependencies in an existing checkout.

```bash
make dev-backend    # Terminal 1 — http://localhost:8000
make dev-frontend   # Terminal 2 — http://localhost:5173

# Or use tmux
make dev-tmux
```

## Access

| Service | URL |
|---------|-----|
| Frontend | http://localhost:5173 |
| Backend API | http://localhost:8000 |
| Admin Panel | http://localhost:8000/admin |

The Vite development server proxies the API, admin panel, and uploaded files to the backend. If you change `PORT` in the project `.env`, restart both servers.

## Create Admin Account

Set these in `.env` before first startup:

```
ADMIN_USERNAME=admin
ADMIN_EMAIL=admin@example.com
ADMIN_PASSWORD=your_secure_password
```

Or create additional admins via the admin panel after login.

## Next Steps

- [Configure environment variables](configuration.md)
- [Explore features](features.md)
- [Deploy with Docker](deployment.md)

---

[Back to Documentation Index](README.md) | [Back to main README](../README.md)
