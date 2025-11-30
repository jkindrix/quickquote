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

- **Backend**: Go 1.23 with Chi router
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

### Makefile Commands

The project includes a comprehensive Makefile. Run `make help` for all available commands:

```bash
make help           # Show all commands
make dev            # Run with hot reload
make test           # Run all tests
make prod-deploy    # Full production deployment
make prod-backup    # Backup production database
make prod-logs      # View production logs
```

### Building

The application uses a multi-stage Docker build. Go 1.23 compiles the binary in the first stage, then it runs in a minimal Alpine container.

To rebuild after code changes:
```bash
docker-compose build app
docker-compose up -d
```

### Running Tests

Since Go is not installed on the host, run tests via Docker:

```bash
docker run --rm -v /path/to/quickquote:/app -w /app golang:1.23-alpine go test ./...
```

For verbose output:
```bash
docker run --rm -v /path/to/quickquote:/app -w /app golang:1.23-alpine go test ./... -v
```

### Database Migrations

Migrations run **automatically on application startup**. The app tracks applied migrations in a `schema_migrations` table and only runs pending ones.

Migration files are in `/migrations/` with naming convention: `NNN_description.up.sql` and `NNN_description.down.sql`.

```bash
# Manually rollback a migration (if needed)
docker exec -i quickquote-db psql -U quickquote -d quickquote < migrations/001_initial_schema.down.sql

# Check migration status
docker exec quickquote-db psql -U quickquote -d quickquote -c "SELECT * FROM schema_migrations ORDER BY version;"
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

Migrations run automatically on startup - no manual steps required.

### Zero-Config Deployment

QuickQuote supports fully automated deployment. On a fresh start:

1. **Database schema** - Auto-created via migrations
2. **Default settings** - Seeded via migrations (call settings, pricing)
3. **Admin user** - Created from `ADMIN_EMAIL`/`ADMIN_PASSWORD` environment variables

To enable zero-config deployment, set these environment variables:
```bash
ADMIN_EMAIL=admin@yourcompany.com
ADMIN_PASSWORD=your-secure-password
```

The admin user is only created if no users exist in the database.

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

## Operations

### Database Backup & Restore

```bash
# Create backup
docker exec quickquote-db pg_dump -U quickquote quickquote > backup_$(date +%Y%m%d_%H%M%S).sql

# Restore from backup
docker exec -i quickquote-db psql -U quickquote -d quickquote < backup_20250101_120000.sql

# Automated daily backup (add to cron)
0 2 * * * docker exec quickquote-db pg_dump -U quickquote quickquote | gzip > /backups/quickquote_$(date +\%Y\%m\%d).sql.gz
```

### Health Monitoring

The `/health` endpoint returns detailed status:
```json
{
  "status": "ok",
  "checks": {
    "database": {"status": "healthy"},
    "ai_service": {"status": "healthy"},
    "voice_providers": {"status": "healthy"}
  }
}
```

Monitor this endpoint with your preferred monitoring tool (Uptime Robot, Healthchecks.io, etc.).

### Log Management

Logs are written to stdout in JSON format (production) or console format (development).

```bash
# Stream logs
docker-compose -f docker-compose.prod.yml logs -f app

# Filter for errors
docker-compose -f docker-compose.prod.yml logs app 2>&1 | grep '"level":"error"'

# Export logs for analysis
docker-compose -f docker-compose.prod.yml logs --no-color app > app.log
```

### Credential Rotation

1. **Database password**: Update in `.env` and restart both containers
2. **API keys**: Update in `.env` and restart app container
3. **Session secret**: Update in `.env`; existing sessions will be invalidated

### Disaster Recovery

To fully rebuild from scratch:
```bash
# Destroy everything (WARNING: deletes all data)
docker-compose -f docker-compose.prod.yml down -v

# Start fresh (migrations and admin user auto-created)
docker-compose -f docker-compose.prod.yml up -d

# Optionally restore data from backup
docker exec -i quickquote-db psql -U quickquote -d quickquote < backup.sql
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

> **Security note:** All `/api/v1/*` routes now require an authenticated dashboard session (or future API token) and enforce CSRF protection. Browser-based tools automatically send the `csrf_token` cookie/header combination; API clients must do the same when making state-changing requests.

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
| `WEBHOOK_BASE_URL` | Base URL for voice provider webhooks |
| `ADMIN_EMAIL` | Initial admin email (zero-config deployment) |
| `ADMIN_PASSWORD` | Initial admin password (zero-config deployment) |

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
