# QuickQuote

An AI-powered voice quoting system built with Go, demonstrating Voice AI integration with multiple providers (Bland AI, Vapi, Retell) and intelligent quote generation using Claude AI.

## Overview

QuickQuote is a production-ready web application that:
- Receives inbound phone calls via configurable Voice AI providers
- Conducts conversational interviews to gather project requirements
- Automatically generates professional quotes using Claude AI
- Displays all calls, transcripts, and quotes in a dashboard

## Features

- **Multi-Provider Voice AI**: Supports Bland AI, Vapi, and Retell with a unified abstraction layer
- **Provider Agnostic Design**: Easily switch between providers or add new ones
- **Intelligent Data Extraction**: Automatically captures caller name, project type, requirements, timeline, budget, and contact preferences
- **AI Quote Generation**: Claude generates professional, detailed quotes from call transcripts
- **Dashboard**: View all calls, transcripts, and generated quotes
- **Authentication**: Session-based auth with secure password hashing

## Tech Stack

- **Backend**: Go 1.22 with Chi router
- **Database**: PostgreSQL 16
- **Voice AI**: Bland AI, Vapi, Retell (pluggable architecture)
- **Quote Generation**: Anthropic Claude
- **Deployment**: Docker + Traefik
- **Frontend**: Server-side rendered HTML with htmx

## Quick Start

### Prerequisites

- Docker and Docker Compose
- Bland AI account with inbound phone number
- Anthropic API key

**Note:** Go is not required on the host machine. All building and testing happens inside Docker containers.

### Local Development

1. Clone the repository:
   ```bash
   git clone https://github.com/jkindrix/quickquote.git
   cd quickquote
   ```

2. Copy environment template:
   ```bash
   cp .env.example .env
   # Edit .env with your credentials
   ```

3. Start services:
   ```bash
   docker-compose up -d
   ```

4. Access the dashboard at http://localhost:8080

**Note:** The default `docker-compose.yml` exposes PostgreSQL on port 5432. If you have another PostgreSQL instance running, use `docker-compose.dev.yml` or modify the port mapping.

## Development

### Building

The application uses a multi-stage Docker build. Go 1.22 compiles the binary in the first stage, then it runs in a minimal Alpine container.

To rebuild after code changes:
```bash
docker-compose build app
docker-compose up -d
```

### Running Tests

Since Go is not installed on the host, run tests via Docker:

```bash
docker run --rm -v /path/to/quickquote:/app -w /app golang:1.22-alpine go test ./...
```

For verbose output:
```bash
docker run --rm -v /path/to/quickquote:/app -w /app golang:1.22-alpine go test ./... -v
```

### Database Migrations

Migrations are in `/migrations/` and must be applied manually:

```bash
# Apply a migration
docker exec -i quickquote-db psql -U quickquote -d quickquote < migrations/001_initial_schema.up.sql

# Rollback a migration
docker exec -i quickquote-db psql -U quickquote -d quickquote < migrations/001_initial_schema.down.sql
```

### Docker Compose Files

| File | Purpose |
|------|---------|
| `docker-compose.yml` | Local development (exposes ports, simple setup) |
| `docker-compose.dev.yml` | Development variant |
| `docker-compose.prod.yml` | Production with Traefik integration |

## Deployment

### Production (Traefik)

The production setup uses Traefik for reverse proxy and TLS termination.

1. Ensure the external `web` network exists (created by Traefik)
2. Configure `.env` with production credentials
3. Deploy:
   ```bash
   docker-compose -f docker-compose.prod.yml up -d --build
   ```

4. Apply any pending migrations:
   ```bash
   docker exec -i quickquote-db psql -U quickquote -d quickquote < migrations/002_session_security.up.sql
   ```

### Container Management

```bash
# View logs
docker-compose -f docker-compose.prod.yml logs -f app

# Restart app after code changes
docker-compose -f docker-compose.prod.yml up -d --build app

# Check health
docker exec quickquote-app wget -qO- http://127.0.0.1:8080/health

# Remove orphaned containers (when switching compose files)
docker-compose -f docker-compose.prod.yml down --remove-orphans
```

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check |
| `/login` | GET/POST | Authentication |
| `/dashboard` | GET | Main dashboard |
| `/calls` | GET | List all calls |
| `/calls/{id}` | GET | Call details |
| `/webhook/bland` | POST | Bland AI webhook |
| `/webhook/vapi` | POST | Vapi webhook |
| `/webhook/retell` | POST | Retell webhook |

## Environment Variables

### Core Configuration
| Variable | Description |
|----------|-------------|
| `DATABASE_HOST` | PostgreSQL host |
| `DATABASE_PORT` | PostgreSQL port |
| `DATABASE_USER` | Database username |
| `DATABASE_PASSWORD` | Database password |
| `DATABASE_NAME` | Database name |
| `ANTHROPIC_API_KEY` | Claude API key |
| `SESSION_SECRET` | Session encryption key |
| `APP_PUBLIC_URL` | Public URL of the application |

### Voice Provider Configuration

At least one voice provider must be configured.

#### Primary Provider Selection
| Variable | Description |
|----------|-------------|
| `VOICE_PROVIDER_PRIMARY` | Primary provider: `bland`, `vapi`, or `retell` |

#### Bland AI (Default)
| Variable | Description |
|----------|-------------|
| `VOICE_PROVIDER_BLAND_ENABLED` | Enable Bland AI (`true`/`false`) |
| `VOICE_PROVIDER_BLAND_API_KEY` | Bland AI API key |
| `VOICE_PROVIDER_BLAND_WEBHOOK_SECRET` | Webhook signature secret (optional) |
| `BLAND_API_KEY` | Legacy: Bland AI API key (backward compatible) |
| `BLAND_INBOUND_NUMBER` | Legacy: Inbound phone number |

#### Vapi
| Variable | Description |
|----------|-------------|
| `VOICE_PROVIDER_VAPI_ENABLED` | Enable Vapi (`true`/`false`) |
| `VOICE_PROVIDER_VAPI_API_KEY` | Vapi API key |
| `VOICE_PROVIDER_VAPI_WEBHOOK_SECRET` | Webhook signature secret (optional) |

#### Retell
| Variable | Description |
|----------|-------------|
| `VOICE_PROVIDER_RETELL_ENABLED` | Enable Retell (`true`/`false`) |
| `VOICE_PROVIDER_RETELL_API_KEY` | Retell API key |
| `VOICE_PROVIDER_RETELL_WEBHOOK_SECRET` | Webhook signature secret (optional) |

## Demo Credentials

- **URL**: https://quickquote.jdok.dev
- **Phone**: +1 (415) 483-4051
- **Demo Login**: verifier@quickquote.demo / VerifyMe2025!

## Architecture

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│   Phone Call    │────▶│  Voice Provider  │────▶│   QuickQuote    │
│   (Inbound)     │     │ (Bland/Vapi/     │     │   (Webhook)     │
└─────────────────┘     │  Retell)         │     └────────┬────────┘
                        └──────────────────┘              │
                        ┌──────────────────┐              │
                        │   Claude AI      │◀─────────────┤
                        │ (Quote Gen)      │              │
                        └──────────────────┘              │
                                                          ▼
                        ┌──────────────────┐     ┌─────────────────┐
                        │   PostgreSQL     │◀───▶│   Dashboard     │
                        │   (Storage)      │     │   (Web UI)      │
                        └──────────────────┘     └─────────────────┘
```

## Voice Provider Architecture

QuickQuote uses a pluggable voice provider architecture that allows switching between or adding new voice AI providers without changing business logic.

### Provider Interface

All voice providers implement the `Provider` interface:

```go
type Provider interface {
    GetName() ProviderType
    ParseWebhook(r *http.Request) (*CallEvent, error)
    ValidateWebhook(r *http.Request) bool
    GetWebhookPath() string
}
```

### Key Components

```
internal/voiceprovider/
├── provider.go       # Core interface and normalized types
├── factory.go        # Registry for provider management
├── bland/
│   └── adapter.go    # Bland AI implementation
├── vapi/
│   └── adapter.go    # Vapi implementation
└── retell/
    └── adapter.go    # Retell implementation
```

### Normalized Call Event

All providers convert their webhook payloads to a normalized `CallEvent`:

```go
type CallEvent struct {
    Provider       ProviderType
    ProviderCallID string
    ToNumber       string
    FromNumber     string
    Status         CallStatus
    Transcript     string
    ExtractedData  *ExtractedData
    // ... more fields
}
```

### Adding a New Provider

1. Create a new package under `internal/voiceprovider/`:
   ```
   internal/voiceprovider/newprovider/
   └── adapter.go
   ```

2. Implement the `Provider` interface:
   ```go
   type Provider struct {
       config *Config
       logger *zap.Logger
   }

   func (p *Provider) GetName() voiceprovider.ProviderType {
       return voiceprovider.ProviderType("newprovider")
   }

   func (p *Provider) ParseWebhook(r *http.Request) (*voiceprovider.CallEvent, error) {
       // Parse provider-specific webhook payload
       // Convert to normalized CallEvent
   }

   func (p *Provider) ValidateWebhook(r *http.Request) bool {
       // Verify webhook signature
   }

   func (p *Provider) GetWebhookPath() string {
       return "/webhook/newprovider"
   }
   ```

3. Add configuration in `internal/config/config.go`

4. Register the provider in `cmd/server/main.go`:
   ```go
   if cfg.VoiceProvider.NewProvider.Enabled {
       registry.Register(newprovider.New(cfg, logger))
   }
   ```

### Provider Comparison

| Feature | Bland AI | Vapi | Retell |
|---------|----------|------|--------|
| Inbound Calls | ✅ | ✅ | ✅ |
| Outbound Calls | ✅ | ✅ | ✅ |
| Conversational Pathways | ✅ | ✅ | ✅ |
| Custom Voices | ✅ | ✅ | ✅ |
| Webhook Signatures | ✅ | ✅ | ✅ |
| Variable Extraction | ✅ | ✅ | ✅ |

## License

MIT License - see LICENSE file for details.
