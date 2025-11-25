# QuickQuote: Building a Voice AI System in an Unfamiliar Technology Stack

## Executive Summary

This project demonstrates building a production-ready Voice AI quoting system from scratch in Go - a language I had limited prior experience with. The goal was to prove that with AI assistance (Claude), a developer can build high-quality software in unfamiliar technology stacks rapidly and effectively.

**Final Deliverables:**
- **Live Application**: https://quickquote.jdok.dev
- **Phone Number**: +1 (415) 483-4051
- **GitHub Repository**: https://github.com/jkindrix/quickquote
- **Demo Credentials**: verifier@quickquote.demo / VerifyMe2025!

---

## The Challenge

Build a production-quality Voice AI demo that:
1. Uses Go (unfamiliar technology)
2. Integrates with Bland AI for voice calls
3. Uses Claude AI for intelligent quote generation
4. Includes authentication, database persistence, and a dashboard
5. Deploys publicly with proper infrastructure

The challenge was to demonstrate that AI-assisted development enables rapid learning and production-quality output in new technology domains.

---

## Timeline & Approach

### Phase 1: Requirements & Architecture

**Approach**: Started by creating comprehensive requirements documentation (REQUIREMENTS.md) with:
- Functional requirements for each component
- Integration requirements for Bland AI and Claude
- Non-functional requirements (security, performance, reliability)
- Clear acceptance criteria

This upfront planning prevented scope creep and provided clear success metrics.

### Phase 2: Core Implementation

**Go Learning**: Relied on Claude to:
- Explain Go idioms and patterns (error handling, interfaces, packages)
- Suggest appropriate libraries (chi for routing, pgx for PostgreSQL, viper for config)
- Review code for Go-specific best practices

**Architecture Decisions**:
- Clean architecture with separate layers (domain, repository, service, handler)
- Interface-based design for testability
- Configuration via environment variables with validation

### Phase 3: Integration & Deployment

**Bland AI Integration**:
- Configured inbound phone number via API
- Designed conversational prompt for gathering project requirements
- Set up webhook endpoint to receive call completion data
- Configured analysis schema for structured data extraction

**Claude AI Integration**:
- Implemented quote generation from transcripts
- Designed prompt to produce professional, actionable quotes
- Handled async processing for better webhook response times

**Deployment**:
- Docker multi-stage builds for minimal image size
- Traefik integration for TLS termination and routing
- PostgreSQL with health checks and connection pooling

---

## Technical Decisions & Rationale

### Why Go?

Go was chosen as the "unfamiliar" technology because:
1. Different paradigm from my primary languages
2. Strong typing and explicit error handling
3. Excellent for backend services
4. Good ecosystem for web services

### Architecture

```
cmd/
  server/main.go       # Application entry point
internal/
  ai/                  # Claude integration
  config/              # Configuration management
  database/            # PostgreSQL connection
  domain/              # Business entities
  handler/             # HTTP handlers
  middleware/          # Logging, recovery
  repository/          # Data access layer
  service/             # Business logic
  webhook/             # Bland AI webhook handling
migrations/            # SQL migrations
```

This structure follows Go conventions and maintains clear separation of concerns.

### Database Design

```sql
-- Core tables
users (id, email, password_hash, name, created_at, updated_at)
sessions (id, user_id, token, expires_at, created_at)
calls (id, bland_call_id, phone_number, caller_name, status,
       transcript, quote_summary, extracted_data, ...)
```

Using JSONB for extracted_data provides flexibility for varying call data structures.

---

## Challenges & Solutions

### Challenge 1: Go Error Handling

**Issue**: Coming from languages with exceptions, Go's explicit error handling was initially verbose.

**Solution**: Embraced the pattern of checking errors immediately and wrapping with context:
```go
if err != nil {
    return fmt.Errorf("failed to create call: %w", err)
}
```

### Challenge 2: Bland AI Webhook Format

**Issue**: Documentation didn't fully specify all webhook payload fields.

**Solution**: Built flexible parsing that handles multiple field names (e.g., `from_number` vs `from`) and gracefully degrades with missing data.

### Challenge 3: Async Quote Generation

**Issue**: Quote generation with Claude takes several seconds, too long for webhook response.

**Solution**: Return immediately from webhook, spawn goroutine for quote generation:
```go
if call.Status == domain.CallStatusCompleted && call.Transcript != nil {
    go s.generateQuoteAsync(call.ID)
}
```

### Challenge 4: Docker Build in CI

**Issue**: No local Go installation, needed to verify code compiles.

**Solution**: Used Docker for compilation verification:
```bash
docker run --rm -v "$PWD":/app -w /app golang:1.22-alpine \
    sh -c "go mod tidy && go build -v ./..."
```

---

## What AI Assistance Enabled

### Learning Acceleration

Claude explained Go concepts in context as I coded:
- Package organization and visibility rules
- Interface satisfaction and composition
- Context usage for cancellation and timeouts
- Proper resource cleanup with defer

### Best Practice Guidance

Received immediate feedback on:
- Go naming conventions (CamelCase for exported, camelCase for internal)
- Error handling patterns
- Struct design and method receivers
- HTTP handler patterns with chi

### Debugging Support

When issues arose (e.g., config field name mismatch), Claude:
1. Identified the root cause from error messages
2. Suggested the minimal fix
3. Explained why the issue occurred

---

## Key Learnings

### 1. Requirements First

Investing time in comprehensive requirements documentation (REQUIREMENTS.md) paid dividends:
- Clear scope prevented feature creep
- Acceptance criteria defined "done"
- Edge cases were considered upfront

### 2. Unfamiliar ≠ Impossible

With AI assistance, building in an unfamiliar language was:
- Faster than expected (production-ready in hours, not weeks)
- Higher quality than typical first projects
- Educational - I now understand Go patterns

### 3. Integration Complexity

External API integrations (Bland AI) required:
- Flexible parsing for undocumented variations
- Robust error handling for network issues
- Clear logging for debugging production issues

### 4. Infrastructure Matters

Proper deployment setup (Docker, Traefik, TLS) was essential for:
- Webhook reliability (HTTPS required)
- Production debugging (structured logging)
- Professional presentation

---

## Final Architecture

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

---

## Metrics & Results

- **Lines of Code**: ~4,800 (Go backend)
- **Time to Production**: Single session
- **External Integrations**: 2 (Bland AI, Anthropic Claude)
- **Database Tables**: 3 (users, sessions, calls)
- **API Endpoints**: 8

---

## Conclusion

This project demonstrates that AI-assisted development fundamentally changes what's possible for learning and building in new technology stacks. The combination of:

1. **Clear requirements** - knowing what to build
2. **AI guidance** - understanding how to build it
3. **Iterative verification** - confirming it works

Enables production-quality software in unfamiliar domains with unprecedented speed.

The resulting application is not a prototype - it's a fully functional, production-ready Voice AI system with proper error handling, security, and infrastructure. This is the future of software development.

---

*Built with Claude AI assistance on November 25, 2025*
