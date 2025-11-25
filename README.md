# QuickQuote

An AI-powered voice quoting system built with Go, demonstrating Voice AI integration with Bland AI and intelligent quote generation using Claude AI.

## Overview

QuickQuote is a production-ready web application that:
- Receives inbound phone calls via Bland AI
- Conducts conversational interviews to gather project requirements
- Automatically generates professional quotes using Claude AI
- Displays all calls, transcripts, and quotes in a dashboard

## Features

- **Voice AI Integration**: Bland AI handles inbound calls with natural conversation
- **Intelligent Data Extraction**: Automatically captures caller name, project type, requirements, timeline, budget, and contact preferences
- **AI Quote Generation**: Claude generates professional, detailed quotes from call transcripts
- **Dashboard**: View all calls, transcripts, and generated quotes
- **Authentication**: Session-based auth with secure password hashing

## Tech Stack

- **Backend**: Go 1.22 with Chi router
- **Database**: PostgreSQL 16
- **Voice AI**: Bland AI
- **Quote Generation**: Anthropic Claude
- **Deployment**: Docker + Traefik
- **Frontend**: Server-side rendered HTML with htmx

## Quick Start

### Prerequisites

- Docker and Docker Compose
- Bland AI account with inbound phone number
- Anthropic API key

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

### Production Deployment

1. Configure environment variables in `.env`
2. Deploy with production compose:
   ```bash
   docker-compose -f docker-compose.prod.yml up -d
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

## Environment Variables

| Variable | Description |
|----------|-------------|
| `DATABASE_HOST` | PostgreSQL host |
| `DATABASE_PORT` | PostgreSQL port |
| `DATABASE_USER` | Database username |
| `DATABASE_PASSWORD` | Database password |
| `DATABASE_NAME` | Database name |
| `BLAND_API_KEY` | Bland AI API key |
| `BLAND_INBOUND_NUMBER` | Inbound phone number |
| `ANTHROPIC_API_KEY` | Claude API key |
| `SESSION_SECRET` | Session encryption key |

## Demo Credentials

- **URL**: https://quickquote.jdok.dev
- **Phone**: +1 (415) 483-4051
- **Demo Login**: verifier@quickquote.demo / VerifyMe2025!

## Architecture

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│   Phone Call    │────▶│    Bland AI      │────▶│   QuickQuote    │
│   (Inbound)     │     │   (Voice Agent)  │     │   (Webhook)     │
└─────────────────┘     └──────────────────┘     └────────┬────────┘
                                                          │
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

## License

MIT License - see LICENSE file for details.
