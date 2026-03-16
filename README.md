# MomShell

[![CI](https://img.shields.io/github/actions/workflow/status/koishi510/MomShell/ci.yml?branch=main&style=flat&label=CI)](https://github.com/koishi510/MomShell/actions/workflows/ci.yml)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=koishi510_MomShell&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=koishi510_MomShell)
[![Security Rating](https://sonarcloud.io/api/project_badges/measure?project=koishi510_MomShell&metric=security_rating)](https://sonarcloud.io/summary/new_code?id=koishi510_MomShell)
[![Reliability Rating](https://sonarcloud.io/api/project_badges/measure?project=koishi510_MomShell&metric=reliability_rating)](https://sonarcloud.io/summary/new_code?id=koishi510_MomShell)
[![Maintainability Rating](https://sonarcloud.io/api/project_badges/measure?project=koishi510_MomShell&metric=sqale_rating)](https://sonarcloud.io/summary/new_code?id=koishi510_MomShell)

[![Release](https://img.shields.io/github/v/release/koishi510/MomShell?style=flat&label=Release)](https://github.com/koishi510/MomShell/releases)
[![License: AGPL-3.0](https://img.shields.io/badge/License-AGPL%203.0-blue?style=flat)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat&logo=go&logoColor=white)](https://go.dev/)
[![Vue](https://img.shields.io/badge/Vue-3-4FC08D?style=flat&logo=vue.js&logoColor=white)](https://vuejs.org/)
[![TypeScript](https://img.shields.io/badge/TypeScript-3178C6?style=flat&logo=typescript&logoColor=white)](https://www.typescriptlang.org/)
[![Docker](https://img.shields.io/badge/Docker-2496ED?style=flat&logo=docker&logoColor=white)](https://www.docker.com/)

AI-powered wellness platform combining emotional companionship, community support, and self-reflection tools in a calming beach-themed experience.

## Features

| Module              | Description                                                                        |
| ------------------- | ---------------------------------------------------------------------------------- |
| **Soul Companion**  | AI chat companion with conversation memory, history viewing, and emotional support |
| **Sisterhood Bond** | Community Q&A with verified healthcare professionals and content moderation        |
| **Echo / Memoir**   | Self-reflection space with AI-generated memoir stickers and partner connection     |
| **Photo Gallery**   | Photo wall with AI-generated images, lifecycle management, and drag/zoom UI        |
| **Whisper**         | Audio-to-text conversation using speech recognition                                |
| **Tasks / Dad Console** | Daily task board with T0/T1/T2 priority, proof photos, partner review, XP/level, skill radar + achievements, perk cards, and blind-box shell gifts |
| **Mobile & PWA**    | Responsive mobile layout with portrait/landscape sprite configs, touch gestures, dynamic viewport units, and offline-capable service worker |
| **Admin Panel**     | Embedded single-page admin at `/admin` — dashboard, user CRUD, config management   |

## Quick Start

```bash
git clone https://github.com/koishi510/MomShell.git
cd MomShell
./scripts/dev-setup.sh

# Start (in separate terminals)
make dev-backend    # http://localhost:8000
make dev-frontend   # http://localhost:5173
```

[Full getting started guide](docs/getting-started.md)

## Project Structure

```
MomShell/
├── backend/            # Go (Gin + GORM + PostgreSQL)
│   ├── cmd/server/     # Entry point
│   ├── internal/       # App code (handler/service/repository/model/dto)
│   │   ├── admin/      # Embedded admin panel (go:embed)
│   │   ├── fileutil/   # Shared file helpers
│   │   └── scheduler/  # Background job scheduling
│   └── pkg/            # Shared utilities (JWT, password, OpenAI, Firecrawl)
├── frontend/           # Vue 3 (Vite + TypeScript + Pinia)
│   └── src/
│       ├── components/ # Overlay panels + beach scene + React 3D shell
│       ├── composables/# Animation, parallax, waves, music, mobile detection
│       ├── lib/api/    # API client modules (chat, community, echo, photo, etc.)
│       ├── stores/     # Pinia stores (auth, UI)
│       ├── types/      # TypeScript type definitions
│       └── utils/      # Shared utility functions
├── deploy/             # Docker Compose + Nginx
├── docs/               # Documentation
├── scripts/            # Development setup scripts
└── Makefile            # Development commands
```

## Docker Deployment

```bash
cp .env.example .env    # Edit with your config
make docker-build-backend
```

[Full deployment guide](docs/deployment.md)

## Documentation

| Document                                   | Description             |
| ------------------------------------------ | ----------------------- |
| [Getting Started](docs/getting-started.md) | Installation and setup  |
| [Development](docs/development.md)         | Development workflow    |
| [Configuration](docs/configuration.md)     | Environment variables   |
| [Architecture](docs/architecture.md)       | Technical overview      |
| [Deployment](docs/deployment.md)           | Docker deployment       |
| [Features](docs/features.md)               | Feature descriptions    |
| [Contributing](CONTRIBUTING.md)            | Contribution guidelines |
| [Changelog](CHANGELOG.md)                  | Version history         |

## License

[AGPL-3.0](LICENSE)
