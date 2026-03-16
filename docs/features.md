# Features

MomShell provides integrated modules designed to support new mothers through postpartum recovery.

## Soul Companion

AI-powered emotional support companion.

- **Empathetic conversations** designed for postpartum emotional support
- **Conversation memory** for personalized experience across sessions
- **Structured memory facts** — AI extracts and stores key information (family, interests, concerns) with category labels
- **Memory management** — view, browse, and delete individual memory facts in the memory panel
- **Conversation history** — view full conversation history (recent turns + compressed summary) in a dedicated tab
- **History clearing** — clear conversation history with confirmation for privacy control
- **Auto-summarization** — older conversation turns are automatically compressed into a summary to keep context efficient
- **Content moderation** with crisis keyword detection
- **Visual effects** — color tones and animations on AI responses
- **Web search grounding** — AI responses reference web search results for factual questions

## Sisterhood Bond

Community Q&A connecting mothers with verified healthcare professionals.

- **Dual channels**: Professional advice and peer experience sharing
- **Verified professionals**: Doctors, therapists, and nurses with credential verification
- **Community board UI**: Board-style layout with avatars, layered scroll, and elastic bounce effects
- **Comment editing**: Users and admins can edit their comments
- **My content tabs**: View own questions and answers in bag panel tabs
- **Engagement**: Q&A, likes, collections, comments
- **AI auto-reply**: AI community assistant replies to all new posts and comment replies with source citations
- **Content moderation**: Keyword-based filtering with manual review queue

## Echo / Memoir

Self-reflection space for mothers to reconnect with their pre-motherhood identity.

- **Identity tags**: Capture personal preferences (music, sounds, literature, memories)
- **AI-generated memoirs**: Nostalgic stories based on personal tags with editable text
- **Memoir covers**: AI-generated SVG gradient covers with image regeneration
- **Partner connection**: Partners can observe and support through a glass window metaphor

## Photo Gallery

Photo management with AI-powered image generation.

- **Photo wall**: Interactive drag-and-zoom photo browsing (pic wall UI)
- **AI photo generation**: Generate photos in memoir using image model
- **Photo lifecycle**: Auto-cleanup of expired photos with admin controls
- **Pearl shell**: Photo display integrated into the beach scene

## Whisper

Audio-to-text conversation feature.

- **Speech recognition**: Convert spoken words to text for the AI companion
- **Conque shell metaphor**: Listen and speak through the beach shell

## Tasks

Daily task execution and partner verification loop (Dad Console).

- **Daily task board**: Dad sees today's tasks with a `pending → completed → verified` lifecycle and XP/level progression
- **AI task generation**: Tasks can be generated based on the baby's age stage, with categories and difficulty
- **Priority levels**: `T0` (urgent / emotional intervention), `T1` (milestone), `T2` (daily routine)
- **Proof photo completion**: Dad can optionally upload a proof photo when completing a task
- **Partner review (Mom)**: Mom reviews Dad's completed tasks with score (1-5) or reject back to pending
- **Progress polling**: UI polls for updates to keep task state in sync between partners
- **Skill radar + achievements**: Verified task scores aggregate into a six-dimension radar; achievements auto-unlock from seeded rules
- **Perk cards**: Mom can issue perk cards; Dad can redeem them (active/used/expired)
- **Shell gifts**: Completing a task can generate a blind-box shell gift for Mom (AI title/content + optional proof photo)

## Beach Scene Navigation

Interactive navigation and exploration of the beach scene.

- **Collapsible NavBar**: Glassmorphism floating menu in the top-right corner with monochrome outline icons; expands on click to reveal navigation items
- **Scroll-to-sprite**: NavBar items scroll the parallax view to center each feature sprite on screen
- **Sprite highlight**: Golden pulse glow animation on feature sprites after scrolling to them
- **Horizontal scrolling**: Trackpad horizontal swipe and mouse wheel horizontal scroll to pan the beach scene; shift+scroll fallback for standard mice
- **Keyboard navigation**: Arrow keys for horizontal panning
- **Drag navigation**: Click-and-drag to pan the scene
- **Interactive tutorial**: 7-step onboarding tour using driver.js, walking new users through each beach scene sprite with auto-scrolling parallax viewport

## Admin Panel

Embedded management interface at `/admin`.

- **Dashboard**: User statistics, content counts, role distribution
- **User management**: Search, filter, paginate, create, edit (role/status), delete
- **Config management**: View and edit runtime configuration (API keys, token expiration)
- **Photo management**: Admin-level photo lifecycle controls
- **Single-file UI**: Tailwind CSS + Alpine.js, embedded via `go:embed`
- **Frontend access**: Admin panel also accessible through the Vue frontend navigation

---

[Back to Documentation Index](README.md) | [Back to main README](../README.md)
