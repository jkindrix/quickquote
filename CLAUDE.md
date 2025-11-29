# QuickQuote - AI Agent Instructions

## Project Purpose

QuickQuote is a **SOFTWARE PROJECT quoting system**. It is NOT for insurance, NOT for any other industry.

**What this app does:**
1. Receives inbound phone calls via Voice AI (Bland AI, Vapi, Retell)
2. AI agent conducts a conversation to gather **software project requirements**
3. Automatically generates professional quotes for software development work
4. Displays calls, transcripts, and quotes on a dashboard

## Critical Context - READ THIS

**This is for SOFTWARE PROJECTS only:**
- Web applications
- Mobile apps
- APIs and integrations
- Custom software development
- SaaS products
- Consulting/advisory services

**This is NOT for:**
- Insurance quotes (auto, home, life, health, business)
- Any other industry vertical
- Generic lead capture

## Information to Collect from Callers

Per REQUIREMENTS.md Section 2.1 (FR-VOICE-003):
- Caller name
- **Project type** (web app, mobile app, API, etc.)
- **Requirements/specifications** (features, integrations, constraints)
- **Timeline expectations** (when they need it)
- **Budget range** (if discussed)
- Contact preference for follow-up

## Tech Stack

- **Backend:** Go with Chi router
- **Database:** PostgreSQL 16
- **Voice AI:** Bland AI (primary), with Vapi and Retell as alternatives
- **Quote Generation:** Anthropic Claude
- **Frontend:** Server-side HTML with htmx
- **Deployment:** Docker + Traefik

## Architecture

```
cmd/server/main.go          # Entry point
internal/
  ai/                       # Claude integration for quote generation
  bland/                    # Bland AI client and configuration
  config/                   # Environment configuration
  database/                 # PostgreSQL connection
  domain/                   # Business entities and interfaces
  handler/                  # HTTP handlers
  middleware/               # Auth, logging, CSRF
  repository/               # Data access layer
  service/                  # Business logic
  voiceprovider/            # Multi-provider abstraction
web/
  static/                   # CSS, JS assets
  templates/                # HTML templates
migrations/                 # SQL migrations
```

## What NOT to Do

1. **Never assume this is for insurance** - It's for software project quotes
2. **Never configure containers live** - Docker containers are immutable; rebuild and redeploy
3. **Never hardcode business-specific values** - Use configuration/environment variables
4. **Never leave TODOs in shipped code** - Implement or don't ship
5. **Never show success for non-functional features** - If it doesn't persist, don't accept the form

## Configuration Approach

All business-specific values should be configurable via:
1. Environment variables (`.env`)
2. Database settings (for runtime changes)

Hardcoding defaults is acceptable for technical defaults (timeouts, buffer sizes) but NOT for:
- Company/business names
- Greeting messages
- Project types
- AI prompts
- Voice settings

## Database Schema

Key tables:
- `users` - Authentication
- `sessions` - Session management
- `calls` - Call records with transcripts and quotes
- `prompts` - Configurable AI presets
- `personas` - Voice agent configurations
- `knowledge_bases` - Reference content for AI
- `pathways` - Conversation flow definitions

## Testing

Run tests via Docker (no local Go installation required):
```bash
docker run --rm -v "$PWD":/app -w /app golang:1.23-alpine go test ./...
```

## Deployment

```bash
# Rebuild and deploy
make prod-restart

# View logs
make prod-logs

# Check health
curl https://quickquote.jdok.dev/health
```

## Key Files for AI Behavior

- `internal/bland/quote_pathway.go` - Conversation flow for gathering project info
- `internal/bland/quickquote_config.go` - Default AI agent configuration
- `internal/domain/persona.go` - Voice agent personality definitions
- `internal/ai/quote_generator.go` - Quote generation prompts

When modifying AI behavior, update these files to reflect **software project** context.
