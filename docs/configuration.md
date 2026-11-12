# Configuration

Environment variables for configuring MomShell. Copy `.env.example` to `.env` and edit.

## Database

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `DATABASE_URL` | PostgreSQL connection string | **Yes** | — |

Format: `postgres://user:password@host:5432/dbname?sslmode=disable`

## JWT Authentication

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `JWT_SECRET_KEY` | Secret key for JWT signing | **Yes** | — |
| `JWT_ACCESS_TOKEN_EXPIRE_MINUTES` | Access token lifetime | No | `30` |
| `JWT_REFRESH_TOKEN_EXPIRE_DAYS` | Refresh token lifetime | No | `7` |

The setup script auto-generates `JWT_SECRET_KEY`. Set a persistent secret in production to preserve sessions across restarts. A server started without one generates a temporary signing key.

## OpenAI Compatible API

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `OPENAI_API_KEY` | API key for the configured provider | For AI features | — |
| `OPENAI_BASE_URL` | API base URL | No | `https://api-inference.modelscope.cn/v1` |
| `OPENAI_MODEL` | Model name | No | `Qwen/Qwen3-235B-A22B` |
| `IMAGE_MODEL` | Model for AI image generation. Set `IMAGE_MODEL=` for a text-only provider. | No | `Tongyi-MAI/Z-Image-Turbo` when unset |
| `EMBEDDING_MODEL` | Embedding model for pgvector retrieval. Set `EMBEDDING_MODEL=` to disable retrieval. | No | `iic/nlp_gte_sentence-embedding_chinese-base` when unset |

Chat and image generation use the corresponding OpenAI-compatible endpoints of the configured provider. Memoirs display a built-in cover when AI image generation is disabled or fails. Future-letter tasks use the configured text model. Retrieval also requires the PostgreSQL `vector` extension and a 768-dimension embedding model.

## Web Search

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `FIRECRAWL_API_KEY` | Firecrawl API key for web search grounding | No | — |

When set, AI replies use web search to reduce hallucinations for factual questions.

## Server

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `PORT` | HTTP server port | No | `8000` |
| `TRUSTED_PROXIES` | Comma-separated IPs or CIDRs of reverse proxies that supply client IP headers | No | `127.0.0.1` in the combined Docker image |

## Frontend

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `VITE_API_BASE_URL` | Optional same-site HTTPS API origin, set at build time or in `frontend/.env.local` | No | Same origin |

Vite proxies local requests to the backend `PORT` from the project `.env`. A separate API origin needs credentialed CORS with `CORS_ORIGINS` set to the frontend's exact origin.

## Initial Admin Account

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `ADMIN_USERNAME` | Admin username | No | — |
| `ADMIN_EMAIL` | Admin email | No | — |
| `ADMIN_PASSWORD` | Admin password | No | — |

Set all three to auto-create an admin on first startup. Skipped if the username or email already exists.

## Runtime Configuration

The following can also be changed for the running process in the admin panel (`/admin` → System Config). Update the deployment environment for settings that should survive a restart.

- `OPENAI_API_KEY`, `OPENAI_BASE_URL`, `OPENAI_MODEL`
- `JWT_ACCESS_TOKEN_EXPIRE_MINUTES`, `JWT_REFRESH_TOKEN_EXPIRE_DAYS`

Read-only in admin panel (requires restart): `DATABASE_URL`, `JWT_ALGORITHM`, `PORT`.

---

[Back to Documentation Index](README.md) | [Back to main README](../README.md)
