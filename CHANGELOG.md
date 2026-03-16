# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.3.0] - 2026-03-16

### Added

- **Task priorities (T0/T1/T2)**: Tasks now carry a `priority` field across template and AI tasks; the UI sorts and highlights urgent/milestone tasks.
- **Proof photo completion**: Dad can optionally upload a proof photo when completing a task; Mom review UI shows thumbnails.
- **Shell Gifts (blind-box shells)**: Completing a task generates a shell gift for Mom (AI-generated title/content + optional proof photo); Mom can list and open shell gifts.
- **Dad dashboard**: Skill radar (six dimensions), achievements (seeded defaults + auto-unlock on verification), and perk cards (Mom issues, Dad redeems).

### Fixed

- **Vite 8 dependency resolution**: Updated `@vitejs/plugin-react` to match Vite 8 peer dependencies so `npm ci` succeeds.
- **MediaPipe Hands build**: Switched to side-effect import + global `Hands` usage to avoid missing ESM named exports under Vite 8/Rolldown.
- **ESLint**: Removed `any` usage in MediaPipe global access.

---

## [1.2.0] - 2026-03-15

### Added

#### Mobile Responsive & PWA

- **Mobile-first responsive design**: Full mobile adaptation for the beach scene, overlay panels, and all feature pages
- **Responsive sprite positioning**: `SpriteData` interface extended with `mobile` and `landscape` override fields; `spriteStyle()` and `labelStyle()` functions select the correct tier (landscape > mobile > desktop)
- **Landscape orientation support**: Separate sprite positioning config for landscape mode via `useIsMobile` composable returning `isMobile`, `isSmall`, and `isLandscape` reactive refs
- **Portrait scroll range**: Increased horizontal parallax scroll distance on portrait mobile (`maxOffset` multiplier 1.5x vs 1.2x desktop)
- **PWA service worker**: Service worker registration for offline caching, gated behind `import.meta.env.PROD` to prevent stale-cache issues during development
- **Touch gesture support**: `touch-action: pan-x` on `.scene` element enables horizontal swipe navigation on mobile
- **Dynamic viewport units**: `dvh`/`vh` fallback pattern across all mobile and landscape media queries for correct viewport height on mobile browsers with dynamic toolbars

#### Navigation & Scene Interaction

- **Collapsible NavBar**: Glassmorphism floating navigation menu in the top-right corner; collapsed by default, expands on click with frosted glass styling and monochrome outline icons
- **Scroll-based navigation**: NavBar items scroll the parallax beach scene to center each feature sprite instead of directly opening panels
- **Sprite highlight animation**: Golden pulse glow effect (drop-shadow) on feature sprites after navigating to them, with automatic cleanup on animation end
- **Horizontal scroll support**: Trackpad two-finger horizontal swipe and mouse horizontal wheel scrolling now navigate the parallax beach scene; shift+scroll fallback for mice without horizontal scroll; overlay panels excluded from scroll capture

#### AI Chat Improvements

- **Memory filtering rules**: Added filtering to prevent persisting negative or harmful content in AI memory extraction

#### Community Board

- **Improved post layout**: Increased row spacing, reduced position jitter, and index-based z-ordering so newer posts appear on top of older ones

### Fixed

- **dvh/vh fallback order**: Corrected dynamic viewport height fallback across `mobile.css`, `OverlayPanel.vue`, and `RoleSelectPanel.vue` — `vh` first (fallback), `dvh` second (override)
- **touch-action scope**: Moved `touch-action: pan-x` from `body` to `.scene` only, so overlay panel content can scroll vertically
- **Event listener leaks**: Extracted anonymous `resize` and `orientationchange` handlers to named functions with proper cleanup in `onUnmounted` (`useParallax.ts`)
- **useIsMobile layout flash**: Compute initial `isMobile`/`isSmall`/`isLandscape` values synchronously in `setup()` to prevent false → true flicker on first render
- **useIsMobile MediaQueryList reuse**: `update()` reads `.matches` from stored MQL objects instead of recreating them on each call
- **Go static analysis**: Replaced `fmt.Errorf` with `errors.New` for constant error strings across `photo.go`, `task.go`, and `whisper.go` (12 occurrences, SA1006)

### Security

- **Dockerfile hardening**: Replaced `COPY . .` with explicit file/directory copies in both `frontend/Dockerfile` and `backend/Dockerfile` to prevent leaking secrets, `.env` files, or unintended files into container images
- **Go version alignment**: Updated `backend/Dockerfile` from `golang:1.23-alpine` to `golang:1.25-alpine` to match `go.mod` requirement

### Performance

- **Gesture control optimization**: Reduced GPU contention on Mac by switching to MediaPipe lite model, lowering camera resolution to 320×240, throttling inference to ~10fps with manual rAF loop, and adding GSAP camera follow throttle with `overwrite: true`

---

## [1.1.0] - 2026-03-14

### Added

#### Onboarding Tutorial

- **Interactive guided tour**: 7-step onboarding tutorial using driver.js, walking new users through beach scene sprites with auto-scrolling parallax viewport
- **Tutorial completion persistence**: `TutorialCompleted` field on User model with `PATCH /api/v1/auth/me/tutorial` endpoint; logged-in users see the tutorial only once, guests see it every visit
- **Admin tutorial status**: Tutorial completion badge displayed in admin user table for at-a-glance onboarding tracking
- **Skip & cancel support**: Users can skip via close button or ESC with confirmation; tutorial auto-cancels if a feature panel opens mid-tour

#### Partner Data Sharing

- **Shared photo library**: Bound partners (linked via `PartnerID`) see a merged photo library; family photo limit raised to 50
- **Shared photo wall**: Both partners share the same wall (max 10); either can add/remove/reorder
- **Shared AI memory facts**: Memory facts visible across bound partners with source attribution (`OwnerUserID`); AI prompt labels facts with `[她]/[他]/[家庭]` for family context
- **Owner badges in memory panel**: When multiple family members exist, memory facts show owner nickname badge
- **Memory update toast**: "记住了" toast now fires reliably when new facts are saved or corrections applied (previously only triggered on legacy profile updates)

#### Beach Scene Decorations

- **Stars**: Interactive starfish decorations on beach scene (CarPage)
- **Pearls**: Pearl collection/decoration elements on beach scene (CarPage)
- **Bloom effect**: Bloom post-processing effect in pearl shell 3D scene
- **Photo wall filter**: Changed photo wall stickers from grayscale to brightness-based filter
- **AI image stickers**: Improved AI-generated image sticker rendering and positioning

#### AI Memory Improvements

- **Memory extraction hardening**: Improved AI memory fact extraction with category-based classification, soft-delete aware deduplication, correction handling, and last-referenced tracking
- **Memory toast relocation**: Repositioned memory update toast for better visibility

### Changed

- **Photo detail modal style**: Unified photo detail modal with glassmorphism variables to match profile modal style

### Fixed

- **Per-user fact dedup**: Memory fact deduplication is now per-user instead of per-family, so both partners can independently record the same fact (e.g., both saying "我喜欢吃苹果")
- **AI reference fix**: Fixed AI reference handling in chat prompts
- **Suitcase position**: Adjusted suitcase position to align with background ground level
- **Pearl shell trigger zone**: Decoupled pearl shell trigger zone from photo wall to prevent interaction conflicts
- **Pearl shell position**: Fixed pearl shell positioning issues

### Added

#### AI Memory & Conversation History

- **Conversation summary (Phase 2)**: Auto-summarize older conversation turns via AI when history exceeds 20 turns; keeps recent 15 turns + compressed summary for long-term context
- **Structured memory facts (Phase 3)**: `ChatMemoryFact` model with category-based classification (family, interest, concern, personal_info, preference, other); auto-extract and deduplicate facts from AI `memory_extract`
- **Memory management panel**: `AiMemoryPanel.vue` with list/delete UI for structured facts, category-colored badges, and memory button in chat header (auth-only)
- **Conversation history viewing**: Users can view full conversation history (turns + summary) via a new "对话历史" tab in memory panel
- **Conversation history clearing**: Clear conversation history with confirm-to-clear button for privacy control
- **Memory panel tabs**: Tab switcher between "记忆" (structured facts) and "对话历史" (conversation history)
- **Backend API**: `GET/DELETE /api/v1/companion/memories`, `GET/DELETE /api/v1/companion/history` endpoints
- **Role-based AI prompts**: Different system prompts for mom, dad, and professional roles with role-appropriate tone and pronouns

#### Community Board Redesign

- **Community bar UI**: Full board-style community redesign with layered layout, avatars, and scroll behavior
- **Board scroll**: Board scrolls as whole unit; comments area scrolls independently within its flex-allocated space
- **Comment editing**: Users and admins can edit comments in the community
- **My-questions / my-answers tabs**: Added tabs to bag panel for viewing own questions and answers
- **Avatar integration**: User avatars displayed in community board posts
- **AI auto-reply on all posts**: AI replies to all new posts and comment replies with source footnotes

#### ModelScope Docker Deployment

- **Embedded PostgreSQL**: Single-container Docker deployment with embedded PostgreSQL for ModelScope platform; nginx listens on port 7860; entrypoint auto-initializes PostgreSQL
- **Companion renamed**: Chat companion renamed to 小石光

#### Security Hardening

- **httpOnly cookie authentication**: Migrated auth tokens from localStorage to httpOnly cookies, eliminating XSS token theft vectors
- **Fixed-window rate limiting**: Added per-endpoint rate limiting to all API routes to prevent abuse
- **Input validation hardening**: Strengthened validation across all user-facing endpoints
- **SQL injection prevention**: Added allowlist validation for ORDER BY clauses in repository layer (defense-in-depth for `question.go` and `answer.go`)
- **Insecure randomness fix**: Replaced `Math.random()` fallback with `crypto.getRandomValues()` for cryptographically secure session ID generation in `ChatPanel.vue`
- **Auth token extraction hardening**: Improved multi-source token extraction security (header, cookie)
- **OpenAI error sanitization**: Removed response body from OpenAI error messages to prevent data exposure
- **CodeQL compliance**: Cookie `Secure=true` always set; taint flow in ORDER BY broken for CodeQL analysis; `X-Forwarded-Proto` guarded behind `TrustProxy` config; hardened cookie attributes and permissions policy

#### Photo Gallery & Lifecycle Management

- **Photo lifecycle management**: Auto-cleanup of expired photos and admin-level photo controls
- **Pic wall**: Interactive photo wall with drag, zoom, and close-window UI
- **AI photo generation**: AI-generated photos in memoir using image model integration
- **Photo wall expansion**: Photo wall expanded from 3x3 (9 photos) to 2x5 grid (10 photos)

#### Memoir & Echo Enhancements

- **Memoir text editing**: Users can edit AI-generated memoir text inline
- **Image regeneration**: Regenerate memoir cover images on demand

#### Whisper & Tasks

- **Whisper (Conque)**: Audio-to-text conversation feature using speech recognition
- **Tasks (Star)**: Goal-setting and task-tracking system with partner support
- **Partner task UI**: Improved task interface with reject and polling support

#### Admin Panel

- **Frontend admin exposure**: Admin panel accessible through the Vue frontend with navigation integration

#### Community Enhancements

- **AI auto-reply trigger**: AI reply triggered on answers mentioning @小石光
- **Cascade delete**: Deleting a question now cascades to its answers and comments
- **Edit/delete UI**: Added edit and delete controls for questions, answers, and comments
- **AI avatar**: Dedicated AI user avatar with simplified mention pattern
- **Source citation**: AI replies cite sources only for factual claims, not emotional support; sources renumbered sequentially

#### Vue 3 Frontend

- **Background music loop**: Global alternating playback for `The Shore and You` and `Travelogue`
- **Profile music control**: Background music volume slider in the profile settings panel
- **Beach shell sprites**: Repositioned shell sprites to the right side of the scene
- **Hand detection camera**: MediaPipe hand detection integration (hidden camera preview)
- **Crab hints update**: Updated crab hints; replaced profile button with avatar click
- **Profile tabs cleanup**: Removed profile tabs from CarPage

### Changed

- **Frontend assets**: Reorganized static assets into `frontend/src/assets/images/` and `frontend/src/assets/audio/`
- **Crab speech bubble anchoring**: Reworked hint bubble positioning to use rendered sprite bounds instead of static offsets
- **Pearl shell layout**: Adjusted pearl shell positioning; uses all photos in pearl shell
- **Pearl removal**: Removed pearl mesh, point light, and all related animations from PearlShell 3D scene
- **Backend refactor**: Extracted shared file deletion helper and removed redundant admin checks

### Fixed

- Missing `autocomplete` attribute on nickname input
- Stray character in `App.vue` template
- Pearl shell layout positioning issues
- Camera preview visibility in pic wall
- Auth token sync between Axios interceptor and Pinia store on refresh
- Crypto fallback guarded; logout auth check restored
- Chinese error messages for frontend validation; `crypto.randomUUID` fallback documented
- Memory button overlapping with close button in chat header
- Photo zoom behavior and drag-too-fast issue
- Rate limiting description corrected from sliding-window to fixed-window in docs
- `TRUST_PROXY` configuration reference added to docs

---

## [1.0.0] - 2026-03-05

### Major Rewrite

Complete rewrite of the backend (Python → Go) and frontend (Next.js → Vue 3). The old codebase has been archived.

### Added

#### Go Backend

- **Tech stack**: Go 1.23, Gin, GORM, PostgreSQL, JWT (golang-jwt), OpenAI SDK
- **Architecture**: Handler → Service → Repository layered design
- **Authentication**: JWT access tokens (30 min) + refresh tokens (7 days), multi-source token extraction (header, cookie)
- **Content moderation**: Keyword-based detection with categories (pseudoscience, violence, self-harm, spam), crisis keyword auto-rejection
- **Embedded admin panel** (`/admin`): Single-file HTML (Tailwind CSS + Alpine.js) via `go:embed`
  - Dashboard with user stats and role distribution
  - User management: search, filter, paginate, create, edit (role/status), delete
  - Runtime config management: view and edit API keys, token expiration (mutex-protected)
  - Self-protection: prevents admin from demoting/banning/deleting themselves
- **Community (Sisterhood Bond)**: Q&A with dual channels, verified professionals, likes, collections, comments
- **Soul Companion**: AI chat with session persistence and conversation memory
- **Echo / Memoir module**: Identity tags (music, sound, literature, memory) and AI-generated memoir stickers using Qwen3, SVG gradient covers, full CRUD with auth

#### Vue 3 Frontend

- **Tech stack**: Vue 3, Vite, TypeScript, Pinia, Axios
- **Beach scene UI**: Parallax layers (sky, ocean, sand), wave animations, interactive elements
- **Overlay system**: Auth, chat, community, profile panels
- **Community panel**: Full backend API integration — question detail fetch, question/answer likes, bookmarks, nested comments, pagination, tag selection, hot posts tab
- **Chat panel**: Session-based conversation continuity, visual effects from `visual_metadata` (5 animations + 7 color tones), memory update toast, personalized greeting from companion profile
- **User center**: Inline nickname editing, 4-stat grid, tabbed my-questions/my-answers with pagination and status badges, change password form
- **ESLint + vue-tsc**: Full lint and type checking configured

#### Docker Deployment

- **Root Dockerfile**: Multi-stage build (Node → Go → Nginx Alpine), frontend + backend in single image
- **docker-compose**: App container (Nginx + Go) + PostgreSQL 16 with health checks and data volume
- **Nginx config**: SPA fallback, gzip, static asset caching, `/api/` and `/admin` reverse proxy to backend
- **Entrypoint script**: Starts Go backend in background + Nginx in foreground

#### Developer Experience

- **dev-setup.sh**: Interactive setup script
  - Auto-detects package manager (pacman/apt/dnf/brew) and offers to install missing dependencies
  - PostgreSQL setup: start service, create user and database, verify connection
  - Interactive environment variable configuration with sensible defaults
  - Auto-generates JWT secret, auto-fills DATABASE_URL from PostgreSQL step
  - Installs Go/npm dependencies, pre-commit hooks, verifies build
- **Pre-commit hooks**: gofmt, go vet, go build, ESLint, vue-tsc, trailing whitespace, YAML validation
- **GitHub Actions CI**: Go (gofmt, vet, golangci-lint, build, test) + Vue (ESLint, typecheck, build)
- **Makefile**: Unified commands for dev, lint, typecheck, build, docker, postgres, deps, clean
- **Frontend Makefile**: Standalone dev/build/lint/typecheck/docker commands

#### Configuration

- **`.env.example`**: Full environment template with English comments
  - Database, JWT, OpenAI, server, frontend, Docker Postgres, initial admin settings
  - Separate local/Docker DATABASE_URL examples
- **`.env` auto-generation**: dev-setup.sh creates `.env` interactively with all variables

#### Documentation

- **README.md**: Project overview with Go/Vue/TypeScript badges
- **docs/architecture.md**: Tech stack, project structure, architecture layers, design decisions
- **docs/getting-started.md**: Prerequisites, setup, first run
- **docs/development.md**: Dev workflow, make commands
- **docs/configuration.md**: All environment variables with descriptions
- **docs/deployment.md**: Docker quick start, architecture diagram, compose setup
- **docs/features.md**: Soul Companion, Sisterhood Bond, Admin Panel
- **CONTRIBUTING.md**: Go/Vue code standards, hooks, commit conventions, branch workflow

### Changed

- **Backend language**: Python (FastAPI/SQLAlchemy) → Go (Gin/GORM)
- **Frontend framework**: Next.js (React) → Vue 3 (Vite)
- **Package name**: `beach-scene` → `momshell-frontend` v1.0.0
- **Frontend port**: 3000 → 5173 (Vite default)
- **API base URL**: Hardcoded → `import.meta.env.VITE_API_BASE_URL` with fallback
- **Env loading**: `godotenv.Load()` → `godotenv.Overload()` so `.env` overrides shell environment
- **OpenAI client**: Replaced go-openai library with raw HTTP client supporting Qwen3 `enable_thinking` field

### Removed

- Python backend (FastAPI, SQLAlchemy, LangGraph, MediaPipe)
- Next.js frontend (React, Framer Motion)
- Recovery Coach module (pose estimation, exercise library)
- Guardian Partner module
- Web search / Firecrawl integration
- Chain-of-Verification (CoVe) service
- Slider CAPTCHA

---

## [0.6.0] - 2026-02-09

### Added

#### Echo Domain - Self-Reflection & Partner Connection

A meditative space for mothers to reconnect with their pre-motherhood identity, and a window for partners to observe and support.

- **Mom Mode (The Origin)**: Self-reflection and meditation features
  - **Identity Tags**: Capture personal preferences across 4 categories (music, natural sounds, literature, youth memories)
  - **Meditation Timer**: Guided breathing sessions with 4-4-6 second rhythm (inhale-hold-exhale)
  - **Scene & Audio Matching**: Personalized backgrounds and ambient audio based on identity tags
  - **Youth Memoir Generation**: AI-generated nostalgic stories based on personal tags
  - **Memoir Rating**: Rate generated memoirs to improve future suggestions

- **Partner Mode (The Guardian)**: Support through observation
  - **Window Clarity System**: Glass window gradually clears as mom meditates (0-100% clarity)
  - **Blurred Mom View**: Partners see a frosted glass view of mom's meditation status
  - **Memory Injection**: Partners can prepare heartwarming memories that unlock at clarity thresholds
  - **Revealed Memories**: Memories automatically unlock and display as clarity increases

- **Dual-Color Nebula UI**: Split-screen entrance with warm amber (mom) and cool indigo (partner) themes
  - Particle animations and breathing nebula effects
  - Role-locked navigation (once chosen, cannot switch roles)
  - Automatic role detection and redirection

- **Backend**:
  - New `echo` service module (`backend/app/services/echo/`)
  - Models: `IdentityTag`, `Scene`, `Audio`, `MeditationSession`, `WindowClarity`, `PartnerMemory`, `YouthMemoir`
  - 14 API endpoints for identity tags, meditation, scenes, audio, memories, and memoirs
  - Seed data with predefined scenes and audio resources

- **Frontend**:
  - Echo entrance page with dual-side selection (`/echo`)
  - Mom dashboard with identity editor and meditation timer (`/echo/mom`)
  - Meditation session page with breathing guidance (`/echo/mom/meditation`)
  - Partner view with clarity meter and memory injector (`/echo/partner`)
  - New components: `GlassWindow`, `IdentityTagEditor`, `MeditationTimer`, `MemoirCard`, `BlurredMomView`, `ClarityMeter`, `MemoryInjector`
  - Echo-specific design tokens with mom/partner color themes

---

## [0.5.3] - 2026-02-09

### Added

#### Security Improvements

- **JWT Secret Auto-Generation**: Automatic random JWT secret key generation for Docker deployments
  - If `JWT_SECRET_KEY` is not set, a random 64-char hex key is generated at startup
  - Startup warning in production mode when using auto-generated (non-persisted) key
  - `dev-setup.sh` now auto-generates a persistent JWT secret in `.env`

- **Slider CAPTCHA**: Human verification for login and registration
  - New `SliderCaptcha` component with drag-to-verify interaction
  - Validates position accuracy, timing, and drag trail
  - Required before form submission on both login and register pages

- **Content Moderation Expansion**: Extended sensitive content filtering
  - Post titles now moderated (previously only content was checked)
  - Comments rejection on moderation failure (previously only marked for review)
  - Nickname moderation on registration and profile update
  - Post title moderation on edit (previously only content was checked)

#### User Experience Improvements

- **Profile Page Enhancements**:
  - Username display in account security section (read-only)
  - Email display with edit capability
  - Backend support for email updates with duplicate checking

- **Registration Improvements**:
  - Role selection during registration (Mom/Dad/Family)
  - Backend accepts `role` field in registration request

- **Password Visibility Toggle**: Show/hide password button on login and register pages
  - Eye icon toggle for all password fields
  - Both password fields share the same visibility state

#### Developer Experience

- **Local Development Setup**:
  - New `frontend/.env.example` template with `NEXT_PUBLIC_API_URL`
  - `dev-setup.sh` now creates `frontend/.env.local` automatically
  - Updated `docs/development.md` with frontend environment setup instructions

### Changed

- **Environment Variables**: Unified `.env.example` style
  - Required keys use placeholder values (e.g., `MODELSCOPE_KEY=your_key_here`)
  - Optional keys are commented out with placeholder values

### Fixed

- Backend lint errors in `config.py` (unnecessary f-string) and `main.py` (unused import)
- **Moderation error messages not displayed**: Frontend now correctly extracts and shows detailed error messages from backend moderation responses
  - Updated `getErrorMessage()` to handle `{message: ...}` response format
  - Applied fix to post creation, comments, profile updates, and all error handlers

---

## [0.5.2] - 2026-02-06

### Added

#### Source Citation Links in AI Replies

- **Source Link Extraction**: AI replies now show which web sources were used
  - New `format_source_links()` helper to format source links for display
  - New `extract_used_sources()` helper to parse `[1]`, `[2]` references from AI response
  - New `strip_citation_markers()` helper to remove inline markers from final output
  - System prompt instructs AI to cite sources with `[1]`, `[2]` notation internally
  - Markers are stripped before display; only clean source links section is shown

- **Integrated into Community AI Replies**:
  - All 6 reply methods in `ai_reply.py` now append source links when web search was used
  - Maximum 3 sources displayed per reply

#### Chain-of-Verification (CoVe) for Hallucination Reduction

- **CoVe Service**: New verification service to reduce AI hallucinations
  - New `verification.py` with `ChainOfVerification` class
  - Implements the CoVe approach from https://arxiv.org/abs/2309.11495
  - Three-step verification: extract claims, verify against search context, correct if needed

- **Integrated into All AI Response Paths**:
  - Community: All 6 reply methods now apply CoVe after generating responses
  - Chat: Both `chat_authenticated` and `chat` (guest) apply CoVe
  - Only triggers when web search context is available
  - Logs when corrections are made for observability

### Changed

- **AI Response Flow**: Now two-stage hallucination reduction
  1. Stage 1 (RAG): Web search provides grounding context
  2. Stage 2 (CoVe): Response verification and correction

---

## [0.5.1] - 2026-02-05

### Added

#### Developer Setup Script

- **One-command setup for new contributors**: `./scripts/dev-setup.sh`
  - Checks system prerequisites (uv, nvm, git, git-lfs) with install hints
  - Creates `.env` from `.env.example`
  - Initializes Git LFS
  - Installs backend Python dependencies via `uv sync`
  - Installs pre-commit hooks
  - Installs frontend Node.js (via nvm) and npm dependencies
  - Verifies setup and prints next-step instructions

#### Web Search Integration (Reduce AI Hallucinations)

- **Firecrawl API Integration**: New web search service for grounding AI responses
  - New `web_search.py` service with `WebSearchService` class
  - Auto-detects factual/medical questions using keyword analysis
  - Searches are triggered for questions containing medical terms + question patterns
  - Emotional support queries are skipped (no search needed)

- **AI Reply Improvements**:
  - All 8 LLM call sites now use web search for factual questions:
    - Community: `reply_to_question`, `reply_to_answer`, `reply_to_comment`, `reply_to_comment_on_ai_answer`, `reply_to_reply_on_ai_comment`, `reply_as_comment_to_answer`
    - Chat: `chat_authenticated`, `chat` (guest mode)
  - Search results formatted as context for LLM to ground responses
  - System prompts updated to instruct AI to use search results when available
  - Search returns up to 10 results with 500-char content preview each

- **New Environment Variable**: `FIRECRAWL_API_KEY` for web search configuration

### Changed

- **License**: Changed from GPL-3.0 to AGPL-3.0 to prevent closed-source SaaS forks
  - AGPL-3.0 requires network service operators to share modified source code
  - Better suited for a web application that could be deployed as a service

- **Documentation Standards**: CLAUDE.md now specifies English-only rule for documentation
  - Exception: Chinese text allowed for code identifiers and UI references

### Fixed

- **Chat Route Authentication**: Direct URL access to `/chat` now requires login
  - Added `AuthGuard` component that redirects unauthenticated users
  - Previously users could bypass homepage auth check by navigating directly

---

## [0.5.0] - 2026-02-05

### Added

#### AI Auto-Reply Feature

- **Community AI Assistant**: New AI role "Shell Sister" that automatically replies to posts and comments
  - Auto-replies to new questions with warm, empathetic responses
  - Responds when users reply to AI's answers
  - `@贝壳姐姐` mention in comments triggers AI reply
  - Uses ModelScope Qwen LLM for generating contextual responses

- **Backend**:
  - New `ai_reply.py` service with `AIReplyService` class
  - Background task execution for non-blocking AI responses
  - Auto-creates AI user account on first use

#### User Data Persistence

- **Chat Memory Persistence**: Chat history and profile saved to database for logged-in users
  - New `ChatMemory` model stores conversation turns and user profile
  - Guests use in-memory storage; authenticated users persist to database
  - Seamless experience across sessions for logged-in users

- **Coach Progress Persistence**: Exercise progress saved to database for authenticated users
  - New `CoachProgress` model stores all progress data as JSON
  - Progress automatically saved after each training session
  - Uses authenticated user ID when logged in, falls back to local ID for guests

#### Account Security

- **Password Change**: Users can change password in profile page
  - New `/auth/change-password` endpoint with old password verification
  - Profile page account security section with password change form
  - Frontend validation for password requirements

#### Admin Features

- **Certification Revocation**: Admins can revoke approved professional certifications
  - New `REVOKED` status in `CertificationStatus` enum
  - New `/certifications/{id}/revoke` endpoint
  - Revoke button and confirmation modal in admin certification review page
  - Revoked users' role reset to default (mom)

### Changed

- **Login Requirements**: Chat, Community, and Coach features now require login from homepage
  - FloatingCard component supports `requiresAuth` prop
  - Redirects to login page if not authenticated

- **Page Title Consistency**: Homepage button titles now match page headers
  - Chat: "Soul Harbor" (was "Soul Companion")
  - Community: "Experience Connect" (was "Mutual Help Community")
  - Coach: "Body Rebuild" (was "AI Recovery Coach")

### Fixed

- **Post Content Display**: Question detail modal now always fetches full content
  - Removed `viewingInProgress` guard that prevented content reload
  - Content now properly displays when reopening the same question

- **Admin Role Preservation**: Admin users no longer lose admin role when approving their own certification
  - Added check `user.role != UserRole.ADMIN` before changing role on certification approval/revocation

---

## [0.4.0] - 2026-02-02

### Added

#### Guardian Partner Module

- **New Feature**: Guardian Partner - A gamified system to help partners participate in postpartum recovery
  - Invite & Bind: Mom generates invite code, partner scans/enters to bind
  - Daily Status Recording: Mom records mood, energy, health conditions, feeding count, sleep hours
  - Smart Notifications: System generates personalized suggestions for partner based on mom's status
  - Task System: Daily tasks with 3 difficulty levels (Easy 10pts, Medium 30pts, Hard 50pts)
  - Partner Levels: Intern → Trainee → Regular → Gold based on points
  - Time Recorder: Photo memories with milestone tracking

- **Backend**:
  - New `guardian` service module (`backend/app/services/guardian/`)
  - Models: `PartnerBinding`, `MomDailyStatus`, `TaskTemplate`, `PartnerDailyTask`, `PartnerProgress`, `PartnerBadge`, `Memory`
  - API endpoints: `/api/v1/guardian/*` (14 endpoints for binding, status, tasks, progress, memories)
  - Task templates seed data with 14 predefined tasks across 4 categories

- **Frontend**:
  - New Guardian dashboard component with role-based UI (Mom/Partner views)
  - Mom view: Partner stats, daily status form, task confirmation
  - Partner view: Progress card, mom's status alert with suggestions, daily tasks
  - Types and API client for Guardian feature

### Changed

- **Homepage**: Updated feature cards with new Chinese names and added Guardian Partner
  - Soul Companion
  - Sisterhood Bond
  - Recovery Coach
  - Guardian Partner
- Grid layout changed from 3 columns to 2x2 / 4 columns responsive

---

## [0.3.4] - 2026-02-01

### Added

#### Community Module - User Role Selection & Professional Certification

- **User Role Selection**: Users can freely switch between family roles (Mom, Dad, Family) in profile page
  - Backend `UserProfileUpdate` schema accepts `role` field (restricted to `mom`, `dad`, `family`)
  - Backend `update_user_profile` validates role change permissions (certified professionals and admins blocked)
  - Frontend profile edit mode includes role selector button group

- **Professional Certification Application** (`/community/certification`)
  - Users can submit certification applications (Doctor, Therapist, Nurse)
  - Form fields: real name, ID card number (optional), license number, hospital/institution, department (optional), title (optional)
  - Page displays current certification status (none / pending / approved / rejected)
  - Frontend validation matches backend constraints (min lengths, ID card format)

- **Admin Certification Review** (`/community/admin/certifications`)
  - Admin-only page for reviewing certification applications
  - Filter by status (pending / approved / rejected / all)
  - Review modal with applicant details, optional comment, approve/reject actions
  - Approval automatically updates user role to corresponding professional identity

- Frontend certification API client functions (`getMyCertification`, `createCertification`)
- Profile quick links: "Professional Certification" for all users; "Certification Review" for admins

### Changed

- Updated `roleNames` mapping to cover all roles (mom, dad, family, certified_doctor, certified_therapist, certified_nurse, admin)
- `UserProfileUpdateParams` now includes optional `role` field

---

## [0.3.3] - 2026-01-28

### Added

#### Community Module - Comment System

- **Answer Comments**: Users can now reply to answers with nested comments
  - Backend: `get_comments`, `create_comment`, `delete_comment` service methods
  - API endpoints: `GET/POST /answers/{id}/comments`, `DELETE /comments/{id}`
  - Max 1 level nesting - deeper replies shown flat with @mention
  - Like/delete functionality for comments
  - Content moderation for new comments

- **PostCard Improvements**
  - Added view count display (eye icon) to all post cards
  - Unified PostCard template across Community Feed, Collections, and My Posts pages
  - Consistent animation pattern using `AnimatePresence mode="popLayout"`

- **QuestionDetailModal Improvements**
  - Exit animation when closing (slide out to right)
  - Delete button moved from card to detail modal (shown for author only)

### Changed

- **Project Structure**: Reorganized backend code into `backend/` directory
  - Moved `app/`, `models/`, `tests/`, `data/`, `tts_cache/` to `backend/`
  - Moved Python config files (`pyproject.toml`, `uv.lock`, `requirements.txt`) to `backend/`
  - Config now loads `.env` from project root
  - Added standalone `backend/Dockerfile`

- **Deployment Structure**: Created `deploy/` directory for deployment configs
  - Moved `docker-compose.yml` to `deploy/`
  - Combined Dockerfile at project root for platform compatibility
  - Moved `nginx.conf` to `deploy/`
  - Updated Makefile with new docker commands (`docker-build-backend`, `docker-build-frontend`)

- **API Versioning**: All API routes now use `/api/v1/` prefix
  - Renamed `app/api/routes/` to `app/api/v1/`
  - Updated all router prefixes in main.py
  - Updated frontend API calls to use `/api/v1/` paths

- **UserMenu**: Replaced framer-motion entry animations with CSS transitions to prevent hydration flicker
- **Comment Structure**: Flattened to max 1 level depth - all deeper replies use @mention format

#### Bug Fixes

- Fixed SQLAlchemy async lazy loading errors by eagerly loading `author.certification` relationships
- Fixed hydration flickering in UserMenu by using CSS transitions triggered after mount
- Fixed comment persistence - nested replies now properly load when reopening detail modal
- Fixed like/collect state not showing correctly in My Posts page

---

## [0.3.2] - 2026-01-26

### Added

#### Developer Experience

- **Makefile**: Added unified command interface for common development tasks
  - `make install` - Install all dependencies (backend + frontend)
  - `make dev-backend` / `make dev-frontend` - Start development servers
  - `make dev-tmux` - Start both servers in tmux split panes
  - `make lint` / `make format` / `make typecheck` - Code quality tools
  - `make docker-up` / `make docker-down` - Docker management
  - `make clean` - Clean caches and temporary files
  - Run `make help` for full command list

- **GitHub Issue Templates**: Added structured issue templates
  - Bug Report - General bug reporting with component selection
  - Feature Request - Feature proposals with user perspective
  - Recovery Coach Issue - Pose detection and exercise-specific issues
  - AI Provider Issue - LLM and API-related issues
  - Documentation Issue - Docs improvements and corrections

- **Security Policy**: Added `SECURITY.md` with vulnerability reporting guidelines

---

## [0.3.1] - 2026-01-26

### Changed

#### AI Provider Migration

- **Soulful Companion**: Migrated from Zhipu GLM-4 to ModelScope Qwen (Qwen2.5-72B-Instruct)
  - Uses OpenAI-compatible API via ModelScope
  - Environment variable changed from `ZHIPUAI_API_KEY` to `MODELSCOPE_KEY`

- **Pose Detection Model**: Default changed from FULL to LITE
  - LITE model provides better performance on low-end servers
  - Configurable via `MEDIAPIPE_MODEL` environment variable (`lite` or `full`)

#### Bug Fixes

- Fixed race condition in camera initialization where video element might not be mounted in time
  - Replaced fixed 100ms delay with `requestAnimationFrame` polling (max 2s timeout)

- Fixed port configuration inconsistency
  - Local development now correctly defaults to port 8000
  - Docker single-container deployment uses 7860 (set via Dockerfile ENV)

- Fixed Docker Compose configuration
  - Removed port conflict (both nginx and frontend were binding to host port 7860)
  - Added `PORT=8000` to backend environment for correct port binding
  - Frontend now uses `expose` instead of `ports` (nginx handles external access)

- Fixed frontend Dockerfile for static export mode
  - Previous Dockerfile expected `output: "standalone"` but next.config.ts uses `output: "export"`
  - Now uses `serve` package to serve static files from `out/` directory

---

## [0.3.0] - 2026-01-25

### Added

#### Docker Deployment Support

- **Docker Compose (Multi-container)**
  - nginx reverse proxy for unified access on port 7860
  - Separate containers for backend, frontend, and nginx
  - Health checks for backend service
  - Persistent data volume for database

- **Single Container Deployment (ModelScope)**
  - Combined frontend/backend in one container
  - Multi-stage build for optimized image size
  - Static frontend served by FastAPI backend
  - Suitable for platforms requiring single container

- **nginx Configuration**
  - Reverse proxy with WebSocket support
  - API routing to backend (`/api/`, `/ws/`, `/health`)
  - Frontend routing for all other requests
  - 50MB max upload size for file uploads

#### Community Module - Mutual Support Community (`frontend/app/community/`, `frontend/components/community/`)

- **Dual-Channel System**
  - Professional Channel: Expert advice from verified healthcare professionals
  - Experience Channel: Real stories and tips from fellow mothers
  - Smooth channel switching with animated transitions

- **User Role System**
  - Regular users: Mom, Dad, Family Member
  - Certified professionals: Doctor, Therapist, Nurse
  - Visual badges and verification indicators

- **Q&A Features**
  - Question posting with rich text and images
  - Like and collection functionality
  - Professional vs experience answer distinction
  - Accepted answer marking

- **Content Moderation**
  - Automated sensitive content filtering
  - Multi-category keyword detection (violence, fraud, self-harm, etc.)
  - Real-time moderation status feedback with animations

- **Community Components**
  - `CommunityFeed`: Main feed with infinite scroll layout
  - `ChannelSwitcher`: Animated dual-channel toggle
  - `PostCard`: Glass-morphism post cards with hover effects
  - `QuestionModal`: Full-screen question composer
  - `QuestionDetailModal`: Detailed question view with answers
  - `DailyResonance`: Trending topics with staggered card layout
  - `ShellPicks`: Collections entry point
  - `CommunityBackground`: Animated gradient background

### Changed

#### UI/UX Improvements

- **Design System Enhancement**
  - Consistent "breathing" visual style across all pages
  - Glass-morphism cards with backdrop blur effects
  - Warm, healing color palette (rose, amber, stone tones)
  - Smooth Framer Motion animations throughout

- **Home Page Polish**
  - Refined navigation and layout consistency
  - Improved visual hierarchy and spacing

- **Component Architecture**
  - Added `lib/design-tokens.ts` for centralized design constants
  - Unified animation spring configurations
  - Reusable motion variants

---

## [0.2.2] - 2026-01-25

### Performance Optimization - Non-blocking LLM Feedback

#### Fixed

- **Critical: Eliminated 1-2 second stuttering** caused by blocking LLM API calls
  - Root cause: `FeedbackNode` awaited LLM API response synchronously, blocking frame processing
  - Solution: LLM feedback generation now runs in background using `asyncio.create_task()`

#### Changed

- **Feedback Node** (`app/services/coach/workflow/nodes/feedback.py`)
  - Background task pattern: feedback generated asynchronously, results delivered on next frame
  - Added `_pending_generation` task tracking to prevent duplicate generations
  - No blocking of frame processing pipeline

- **Pose Detection** (`app/services/coach/pose/detector.py`)
  - Switched back to VIDEO mode from LIVE_STREAM for better responsiveness
  - Default model changed to LITE for performance (configurable via `MEDIAPIPE_MODEL` env var)
  - VIDEO mode provides synchronous results without the 1-frame delay of LIVE_STREAM

- **Frontend** (`frontend/app/coach/page.tsx`)
  - Increased frame rate to 20 FPS
  - Added keypoint smoothing (EMA) for smoother skeleton rendering
  - Using `requestAnimationFrame` for optimal frame timing
  - Increased capture resolution to 480x360 for better detection
  - `PHASE_NAMES` moved to module level constant

#### Technical Decisions

| Decision                    | Rationale                                                                     |
| --------------------------- | ----------------------------------------------------------------------------- |
| Background LLM generation   | LLM API calls take 1-2 seconds; running in background prevents frame blocking |
| VIDEO mode over LIVE_STREAM | LIVE_STREAM has 1-frame delay; VIDEO mode is synchronous and more responsive  |
| LITE model by default       | Better performance on low-end servers; FULL available via env var             |
| Keypoint smoothing (EMA)    | Reduces jitter in skeleton rendering, factor of 0.25 for responsiveness       |

#### Performance Impact

- **Eliminated periodic 1-2 second freezes** during exercise sessions
- Smooth 20+ FPS skeleton rendering
- LLM feedback still generated every 6 seconds but no longer blocks UI

---

## [0.2.1] - 2026-01-24

### Performance Optimization - Async Processing & Data Transfer

#### Changed

- **Critical: Eliminated Full Image Return** (`app/api/routes/websocket.py`)
  - Server now returns only keypoint coordinates (~1KB) instead of full annotated images (~15KB)
  - **90%+ reduction in WebSocket data transfer**
  - Removed server-side skeleton drawing and image encoding for each frame

- **Frontend Skeleton Rendering** (`frontend/app/rehab/page.tsx`)
  - Added client-side skeleton drawing using Canvas API
  - `drawSkeleton()` function renders pose with color-coded feedback
  - Increased frame rate from 8 to 10 FPS (feasible due to reduced latency)
  - Added `POSE_CONNECTIONS` constant for MediaPipe 33-landmark skeleton

- **Pose Detection** (`app/services/coach/pose/detector.py`)
  - Added `detect_async()` method using thread pool executor
  - Added `draw_landmarks_async()` method for non-blocking drawing
  - Added `detection_scale` parameter (default 0.5x) for faster detection
  - Frame downscaling before MediaPipe processing reduces computation time
  - Shared `ThreadPoolExecutor` with 2 workers for CPU-bound operations

- **Detection Node** (`app/services/coach/workflow/nodes/detect.py`)
  - Updated to use async pose detection
  - Added `get_annotated_frame_async()` method
  - Configurable `detection_scale` parameter passed through factory function

- **WebSocket Handler** (`app/api/routes/websocket.py`)
  - Added async `decode_frame_async()` for base64 decoding
  - Dedicated `ThreadPoolExecutor` with 2 workers for image I/O
  - Returns `keypoints` and `skeleton_color` instead of `annotated_frame`

#### Technical Decisions

| Decision                       | Rationale                                                                   |
| ------------------------------ | --------------------------------------------------------------------------- |
| Client-side skeleton rendering | Eliminates ~15KB/frame return transfer, removes encoding latency            |
| Thread pool for pose detection | MediaPipe is CPU-bound, thread pool prevents blocking async event loop      |
| 0.5x detection scale default   | MediaPipe landmarks are normalized, half resolution sufficient for accuracy |
| Return keypoints as JSON       | ~1KB vs ~15KB per frame, 90%+ bandwidth reduction                           |

#### Performance Impact

- **Network transfer reduced by 90%+** (keypoints ~1KB vs full image ~15KB)
- Event loop no longer blocked by CPU-intensive operations
- Server no longer encodes images for each frame
- Smoother real-time feedback with lower latency

---

## [0.2.0] - 2026-01-24

### Changed

#### Architecture Refactoring - Frontend/Backend Separation

- **Backend (FastAPI)**
  - Migrated to pure API server architecture
  - Added CORS middleware for cross-origin requests
  - API routes served under `/api` prefix
  - Health check endpoint at `/health`

- **Frontend (Next.js)**
  - New Next.js 16 + React 19 frontend in `frontend/` directory
  - TailwindCSS 4 for styling
  - Framer Motion for animations
  - TypeScript support

### Added

#### Soulful Companion Chat Module (`app/services/chat/`)

- **AI Chat Service**
  - Zhipu GLM-4 powered emotional companion
  - Warm, validating, non-judgmental conversation style
  - Designed specifically for postpartum women support
  - User profile and conversation memory management
  - Visual metadata generation for UI effects

- **Chat API**
  - `POST /api/v1/companion/chat` - Send message and receive AI response
  - Visual response with color tones and effects

- **Chat Frontend**
  - `CompanionInterface` component for chat UI
  - `AmbientCanvas` for visual ambient effects
  - `InputArea` and `ResponseText` components

### Dependencies

- zhipuai >= 2.1.5 (GLM-4 chat integration)
- next 16.1.4 (frontend framework)
- react 19.2.3 (UI library)
- tailwindcss 4 (styling)
- framer-motion 12.29.0 (animations)

---

## [0.1.0] - 2026-01-23

### Added

#### Core Recovery Coach Module (`app/recovery_coach/`)

- **Pose Estimation** (`pose/`)
  - MediaPipe-based pose detection with 33 keypoints
  - Real-time landmark drawing and feedback overlay
  - Angle calculation utilities for joint analysis
  - Privacy-first local processing

- **Exercise Library** (`exercises/`)
  - 9 postpartum-specific exercises across 5 categories:
    - Breathing: Diaphragmatic Breathing, Core Activation Breath
    - Pelvic Floor: Kegel Exercise, Pelvic Tilt
    - Diastasis Recti: Dead Bug Modified, Abdominal Bracing
    - Posture: Cat-Cow Stretch
    - Strength: Glute Bridge, Side-Lying Leg Lift
  - Structured phase requirements with angle targets
  - 3 pre-built training sessions

- **Motion Analysis** (`analysis/`)
  - Posture analyzer with joint angle evaluation
  - Safety monitor with fatigue detection
  - Consecutive poor form tracking
  - Automatic rest prompts

- **Feedback System** (`feedback/`)
  - AI-powered feedback generation using Claude/LangChain
  - Warm, encouraging feedback tone (non-judgmental)
  - edge-tts integration for Chinese TTS
  - Template fallback for offline operation

- **Progress Tracking** (`progress/`)
  - Session recording and statistics
  - Strength recovery metrics (Core, Pelvic Floor, Posture, Flexibility)
  - Streak tracking
  - Achievement/badge system with 9 achievements

- **LangGraph Workflow** (`workflow/`)
  - State machine for coaching flow: Detect → Analyze → Feedback → Track
  - Real-time session state management
  - Phase timing and rep/set progression

#### API Layer (`app/api/`)

- **WebSocket endpoint** (`/api/ws/coach/{session_id}`)
  - Real-time video frame processing
  - Bidirectional feedback communication
  - Session control (start, pause, resume, rest, end)

- **REST API endpoints**
  - `GET /api/exercises/` - List all exercises
  - `GET /api/exercises/{id}` - Get exercise details
  - `GET /api/exercises/sessions/` - List training sessions
  - `GET /api/progress/{user_id}` - Get user progress
  - `GET /api/progress/{user_id}/summary` - Get progress summary
  - `GET /api/progress/{user_id}/achievements` - Get achievements

#### Frontend (`app/static/`, `app/templates/`)

- Responsive web interface
- Exercise selection with category filtering
- Real-time coaching view with:
  - Camera feed with pose overlay
  - Progress indicators (sets, reps, phase)
  - Live score display
  - Feedback messages
- Progress dashboard with:
  - Session statistics
  - Strength recovery metrics visualization
  - Achievement badges
- Session completion modal with summary

#### Configuration

- Environment-based settings via pydantic-settings
- Configurable MediaPipe parameters
- TTS voice and rate customization
- Safety thresholds configuration

### Dependencies

- mediapipe >= 0.10.0 (pose estimation)
- langgraph >= 0.2.0 (workflow orchestration)
- langchain-anthropic >= 0.3.0 (LLM integration)
- edge-tts >= 6.1.0 (text-to-speech)
- fastapi >= 0.115.0 (web framework)
- websockets >= 12.0 (real-time communication)
- opencv-python >= 4.8.0 (image processing)
- sqlalchemy >= 2.0.0 (database ORM)

### Technical Notes

- All pose processing runs locally (privacy-first design)
- LangGraph manages the coaching state machine
- TTS audio is cached to reduce latency
- WebSocket protocol supports 15 FPS frame processing
