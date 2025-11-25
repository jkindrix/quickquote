# QuickQuote Requirements Specification

**Version:** 1.0
**Purpose:** Comprehensive requirements for the QuickQuote Voice AI Demo
**Challenge Context:** Prove that with AI assistance, any technology is accessible at a professional level

---

## Table of Contents

1. [Project Overview](#1-project-overview)
2. [Core Functional Requirements](#2-core-functional-requirements)
3. [Technical Requirements](#3-technical-requirements)
4. [Integration Requirements](#4-integration-requirements)
5. [Security Requirements](#5-security-requirements)
6. [User Experience Requirements](#6-user-experience-requirements)
7. [Deployment Requirements](#7-deployment-requirements)
8. [Quality Standards](#8-quality-standards)
9. [Acceptance Criteria](#9-acceptance-criteria)
10. [Out of Scope](#10-out-of-scope)
11. [Verification Checklist](#11-verification-checklist)
12. [Final Deliverables](#12-final-deliverables)

---

## 1. Project Overview

### 1.1 Challenge Statement

Build "QuickQuote" - a Voice AI demo that proves capability in an unfamiliar technology stack (Go) while delivering production-quality software.

### 1.2 Business Purpose

A voice-enabled system where potential clients can call a real phone number, speak with an AI assistant that gathers project details, and automatically generates a professional quote summary viewable on a dashboard.

### 1.3 Success Criteria

The project succeeds when:
- A real person can call a real phone number
- An AI conducts a natural conversation gathering project requirements
- The system automatically generates a professional quote from the conversation
- An authenticated user can view calls, transcripts, and quotes on a dashboard
- The entire system is deployed and publicly accessible

### 1.4 Technology Constraints

| Component | Technology | Rationale |
|-----------|------------|-----------|
| Backend | Go | Challenge requirement (unfamiliar language) |
| Database | PostgreSQL | Reliable, full-featured RDBMS |
| Voice AI | Bland AI | Simplest integration, built-in phone numbers |
| Quote Generation | Claude (Anthropic) | High-quality text generation |
| Frontend | htmx + Server-side HTML | Modern, simple, no JS framework |
| Deployment | Docker + Traefik | Existing infrastructure |

---

## 2. Core Functional Requirements

### 2.1 Voice AI System

#### FR-VOICE-001: Inbound Call Reception
- **Description:** System must receive inbound phone calls on a real, dialable phone number
- **Priority:** Critical
- **Acceptance:** A call placed from any phone connects to the AI assistant
- **Verification:** Manual call test from a mobile phone

#### FR-VOICE-002: AI Conversation
- **Description:** AI assistant conducts a natural conversation to gather project details
- **Priority:** Critical
- **Acceptance:** AI asks relevant questions, responds appropriately, handles conversation flow
- **Verification:** Complete a full call and review transcript for coherence

#### FR-VOICE-003: Information Extraction
- **Description:** System extracts structured data from conversations
- **Priority:** High
- **Required Fields:**
  - [ ] Caller name (if provided)
  - [ ] Project type/description
  - [ ] Requirements/specifications
  - [ ] Timeline expectations
  - [ ] Budget range (if discussed)
  - [ ] Contact preference for follow-up
- **Acceptance:** Extracted data is stored and displayed correctly
- **Verification:** Review extracted data against transcript

#### FR-VOICE-004: Call Recording/Transcript
- **Description:** System captures and stores call transcripts
- **Priority:** Critical
- **Acceptance:** Full transcript available after call completion
- **Verification:** Transcript matches actual conversation content

#### FR-VOICE-005: Call Status Tracking
- **Description:** System tracks call lifecycle status
- **Priority:** High
- **Required Statuses:**
  - [ ] `pending` - Call initiated but not yet processed
  - [ ] `in_progress` - Call currently active
  - [ ] `completed` - Call finished successfully
  - [ ] `failed` - Call encountered an error
  - [ ] `no_answer` - Call was not answered
- **Acceptance:** Status accurately reflects call state
- **Verification:** Status transitions correctly during call lifecycle

### 2.2 Quote Generation System

#### FR-QUOTE-001: Automatic Quote Generation
- **Description:** System automatically generates a quote summary after call completion
- **Priority:** Critical
- **Acceptance:** Quote is generated within 60 seconds of call completion
- **Verification:** Quote appears on dashboard without manual intervention

#### FR-QUOTE-002: Quote Content Quality
- **Description:** Generated quotes must be professional and comprehensive
- **Priority:** Critical
- **Required Elements:**
  - [ ] Project summary (what the client needs)
  - [ ] Scope of work (deliverables)
  - [ ] Key requirements mentioned
  - [ ] Timeline considerations
  - [ ] Estimated complexity/effort indication
  - [ ] Recommended next steps
- **Acceptance:** Quote reads as professional business communication
- **Verification:** Review generated quote for completeness and professionalism

#### FR-QUOTE-003: Quote Regeneration
- **Description:** Users can manually regenerate quotes
- **Priority:** Medium
- **Acceptance:** Button click triggers new quote generation
- **Verification:** New quote differs from previous (not cached)

### 2.3 Dashboard System

#### FR-DASH-001: Call List View
- **Description:** Dashboard displays list of all calls
- **Priority:** Critical
- **Required Columns:**
  - [ ] Caller name/identifier
  - [ ] Phone number
  - [ ] Call status
  - [ ] Duration
  - [ ] Quote generated (yes/no)
  - [ ] Date/time
  - [ ] Action buttons
- **Acceptance:** All calls visible with correct data
- **Verification:** Cross-reference with database records

#### FR-DASH-002: Call Detail View
- **Description:** Detailed view for individual calls
- **Priority:** Critical
- **Required Sections:**
  - [ ] Call metadata (phone, duration, status, timestamps)
  - [ ] Full transcript
  - [ ] Extracted information
  - [ ] Generated quote
  - [ ] Regenerate quote button
- **Acceptance:** All sections display correctly with real data
- **Verification:** Manual inspection of detail page

#### FR-DASH-003: Pagination
- **Description:** Call list supports pagination for large datasets
- **Priority:** Medium
- **Acceptance:** Navigate between pages, correct count displayed
- **Verification:** Create >20 test records, verify pagination

#### FR-DASH-004: Real-time Updates (Optional Enhancement)
- **Description:** Dashboard updates without full page refresh
- **Priority:** Low
- **Acceptance:** New calls appear, status updates reflect automatically
- **Verification:** Place call while dashboard open, observe updates

### 2.4 Authentication System

#### FR-AUTH-001: User Login
- **Description:** Secure login with email and password
- **Priority:** Critical
- **Acceptance:** Valid credentials grant access, invalid credentials rejected
- **Verification:** Test both valid and invalid login attempts

#### FR-AUTH-002: Session Management
- **Description:** Authenticated sessions with secure cookies
- **Priority:** Critical
- **Requirements:**
  - [ ] Sessions expire after configured duration (default 24h)
  - [ ] HttpOnly cookies (no JavaScript access)
  - [ ] Secure flag when using HTTPS
  - [ ] SameSite protection
- **Acceptance:** Sessions work correctly, expire appropriately
- **Verification:** Check cookie attributes, test expiration

#### FR-AUTH-003: Protected Routes
- **Description:** Dashboard and data routes require authentication
- **Priority:** Critical
- **Acceptance:** Unauthenticated requests redirect to login
- **Verification:** Access protected routes without login

#### FR-AUTH-004: Logout
- **Description:** Users can log out and terminate session
- **Priority:** High
- **Acceptance:** Logout clears session, subsequent requests require login
- **Verification:** Logout then attempt protected route access

---

## 3. Technical Requirements

### 3.1 Architecture

#### TR-ARCH-001: Clean Architecture
- **Description:** Codebase follows clean architecture principles
- **Priority:** High
- **Requirements:**
  - [ ] Domain layer (entities, business rules) has no external dependencies
  - [ ] Repository interfaces defined in domain, implemented separately
  - [ ] Service layer contains business logic
  - [ ] Handler layer handles HTTP concerns only
  - [ ] Clear dependency direction (outer depends on inner)
- **Verification:** Code review of import statements and layer boundaries

#### TR-ARCH-002: Separation of Concerns
- **Description:** Each package has a single, clear responsibility
- **Priority:** High
- **Package Responsibilities:**
  - [ ] `config` - Configuration loading only
  - [ ] `database` - Connection management only
  - [ ] `domain` - Entities and interfaces only
  - [ ] `repository` - Data access only
  - [ ] `service` - Business logic only
  - [ ] `handler` - HTTP handling only
  - [ ] `middleware` - Cross-cutting HTTP concerns only
  - [ ] `webhook` - External webhook payloads only
  - [ ] `ai` - AI provider integration only
- **Verification:** Review each package for scope creep

#### TR-ARCH-003: Dependency Injection
- **Description:** Dependencies injected rather than created internally
- **Priority:** High
- **Acceptance:** All dependencies passed via constructors
- **Verification:** Review constructor signatures

### 3.2 Code Quality

#### TR-CODE-001: Error Handling
- **Description:** Comprehensive error handling throughout
- **Priority:** Critical
- **Requirements:**
  - [ ] All errors checked (no ignored errors)
  - [ ] Errors wrapped with context where appropriate
  - [ ] User-facing errors are friendly, logs contain details
  - [ ] No panics in normal code paths
- **Verification:** Code review, test error scenarios

#### TR-CODE-002: Logging
- **Description:** Structured logging with appropriate levels
- **Priority:** High
- **Requirements:**
  - [ ] Structured logging (zap)
  - [ ] Request logging with method, path, status, duration
  - [ ] Error logging with stack traces where appropriate
  - [ ] Debug logging for development
  - [ ] No sensitive data in logs (passwords, API keys)
- **Verification:** Review log output in various scenarios

#### TR-CODE-003: Configuration
- **Description:** All configuration via environment variables
- **Priority:** High
- **Requirements:**
  - [ ] All configurable values in .env.example
  - [ ] Validation of required configuration on startup
  - [ ] Sensible defaults where appropriate
  - [ ] No hardcoded secrets or environment-specific values
- **Verification:** Review .env.example completeness

#### TR-CODE-004: Type Safety
- **Description:** Strong typing throughout
- **Priority:** Medium
- **Requirements:**
  - [ ] Custom types for domain concepts (CallStatus, etc.)
  - [ ] No interface{} without good reason
  - [ ] JSON marshaling with proper struct tags
- **Verification:** Code review for type usage

### 3.3 Database

#### TR-DB-001: Schema Design
- **Description:** Proper relational database schema
- **Priority:** High
- **Requirements:**
  - [ ] Primary keys on all tables
  - [ ] Foreign key constraints where appropriate
  - [ ] Indexes on frequently queried columns
  - [ ] Timestamps (created_at, updated_at) on all tables
  - [ ] UUID for primary keys (better for distributed systems)
- **Verification:** Review migration files

#### TR-DB-002: Migrations
- **Description:** Database changes via versioned migrations
- **Priority:** High
- **Requirements:**
  - [ ] Up and down migrations for all changes
  - [ ] Migrations are idempotent where possible
  - [ ] Clear naming convention
- **Verification:** Run migrations up and down

#### TR-DB-003: Connection Management
- **Description:** Proper database connection handling
- **Priority:** High
- **Requirements:**
  - [ ] Connection pooling configured
  - [ ] Health checks available
  - [ ] Graceful shutdown closes connections
- **Verification:** Load test, shutdown test

### 3.4 HTTP Server

#### TR-HTTP-001: Router Configuration
- **Description:** Chi router with proper middleware stack
- **Priority:** High
- **Middleware Stack:**
  - [ ] Request ID generation
  - [ ] Real IP extraction
  - [ ] Request logging
  - [ ] Panic recovery
  - [ ] Compression
- **Verification:** Review main.go middleware setup

#### TR-HTTP-002: Request/Response Handling
- **Description:** Proper HTTP semantics
- **Priority:** High
- **Requirements:**
  - [ ] Correct status codes (200, 201, 400, 401, 404, 500)
  - [ ] Content-Type headers set correctly
  - [ ] POST/Redirect/GET pattern for forms
- **Verification:** Test various scenarios

#### TR-HTTP-003: Timeouts
- **Description:** Appropriate timeouts configured
- **Priority:** Medium
- **Requirements:**
  - [ ] Read timeout
  - [ ] Write timeout
  - [ ] Idle timeout
- **Verification:** Review server configuration

---

## 4. Integration Requirements

### 4.1 Bland AI Integration

#### IR-BLAND-000: Phone Number Provisioning
- **Description:** Inbound phone number for receiving calls
- **Priority:** Critical
- **Approach:** Bland-managed phone number (simplest option)
- **Clarification:** Using a Bland AI-provisioned phone number directly, NOT porting an external number or routing through Twilio. Bland provides the number as part of their service.
- **Requirements:**
  - [ ] Phone number provisioned through Bland AI dashboard
  - [ ] Number is US-based and dialable from any phone
  - [ ] Number configured to use QuickQuote AI agent
  - [ ] Webhook URL configured for call completion events
- **Acceptance:** Calling the number connects to Bland AI agent
- **Verification:** Place test call from mobile phone

#### IR-BLAND-001: Webhook Reception
- **Description:** Receive and process Bland AI webhooks
- **Priority:** Critical
- **Requirements:**
  - [ ] Accept POST requests at /webhook/bland
  - [ ] Parse Bland webhook payload correctly
  - [ ] Handle all expected payload fields
  - [ ] Return appropriate HTTP status codes
- **Verification:** Test with actual Bland webhook

#### IR-BLAND-002: Payload Handling
- **Description:** Correctly handle all Bland payload fields
- **Priority:** High
- **Required Fields:**
  - [ ] call_id
  - [ ] status
  - [ ] concatenated_transcript
  - [ ] transcripts (array with speaker labels)
  - [ ] variables (custom data)
  - [ ] call_length
  - [ ] to/from numbers
  - [ ] recording_url (if available)
- **Verification:** Log and review actual payloads

#### IR-BLAND-003: Error Resilience
- **Description:** Handle malformed or unexpected webhooks gracefully
- **Priority:** Medium
- **Requirements:**
  - [ ] Log malformed payloads for debugging
  - [ ] Don't crash on unexpected fields
  - [ ] Return 400 for clearly invalid requests
- **Verification:** Send malformed test payloads

### 4.2 Claude AI Integration

#### IR-CLAUDE-001: API Communication
- **Description:** Communicate with Claude API for quote generation
- **Priority:** Critical
- **Requirements:**
  - [ ] Correct API endpoint usage
  - [ ] Proper authentication header
  - [ ] Appropriate model selection (claude-3-sonnet or similar)
  - [ ] Handle rate limits gracefully
- **Verification:** Generate quotes, check API usage

#### IR-CLAUDE-002: Prompt Engineering
- **Description:** Effective prompts for quote generation
- **Priority:** High
- **Requirements:**
  - [ ] Clear instructions for quote format
  - [ ] Include full transcript in context
  - [ ] Include extracted data for reference
  - [ ] Professional tone guidance
- **Verification:** Review prompt quality and output

#### IR-CLAUDE-003: Response Handling
- **Description:** Parse and store Claude responses
- **Priority:** High
- **Requirements:**
  - [ ] Extract generated text correctly
  - [ ] Handle API errors gracefully
  - [ ] Timeout for long-running requests
- **Verification:** Test various response scenarios

---

## 5. Security Requirements

### 5.1 Authentication Security

#### SR-AUTH-001: Password Storage
- **Description:** Secure password hashing
- **Priority:** Critical
- **Requirements:**
  - [ ] bcrypt with appropriate cost factor
  - [ ] No plaintext passwords stored
  - [ ] No passwords in logs
- **Verification:** Inspect database, review logs

#### SR-AUTH-002: Session Security
- **Description:** Secure session token handling
- **Priority:** Critical
- **Requirements:**
  - [ ] Cryptographically random session tokens
  - [ ] Tokens stored hashed (if persistent)
  - [ ] Session expiration enforced
  - [ ] Session invalidation on logout
- **Verification:** Review session implementation

### 5.2 Data Security

#### SR-DATA-001: Sensitive Data Handling
- **Description:** Protect sensitive information
- **Priority:** Critical
- **Requirements:**
  - [ ] API keys not logged or exposed
  - [ ] Phone numbers treated as PII
  - [ ] Transcripts protected behind auth
- **Verification:** Security review of logs and responses

#### SR-DATA-002: Input Validation
- **Description:** Validate all external input
- **Priority:** High
- **Requirements:**
  - [ ] Validate webhook payloads
  - [ ] Validate form inputs
  - [ ] Sanitize data before display (XSS prevention)
- **Verification:** Test with malicious inputs

### 5.3 Infrastructure Security

#### SR-INFRA-001: HTTPS
- **Description:** All traffic over HTTPS in production
- **Priority:** Critical
- **Requirements:**
  - [ ] Valid TLS certificate
  - [ ] HTTP redirects to HTTPS
  - [ ] Secure cookie flag enabled
- **Verification:** Check production deployment

#### SR-INFRA-002: Environment Isolation
- **Description:** Secrets not in code
- **Priority:** Critical
- **Requirements:**
  - [ ] All secrets via environment variables
  - [ ] .env not committed to git
  - [ ] .env.example has no real values
- **Verification:** Review .gitignore and .env.example

---

## 6. User Experience Requirements

### 6.1 Visual Design

#### UX-VIS-001: Clean, Professional Interface
- **Description:** Dashboard looks professional and is easy to use
- **Priority:** Medium
- **Requirements:**
  - [ ] Consistent color scheme
  - [ ] Clear typography hierarchy
  - [ ] Adequate whitespace
  - [ ] Responsive on mobile devices
- **Verification:** Visual inspection, mobile testing

#### UX-VIS-002: Status Indicators
- **Description:** Clear visual indicators for call status
- **Priority:** Medium
- **Requirements:**
  - [ ] Color-coded status badges
  - [ ] Consistent iconography
  - [ ] Loading states visible
- **Verification:** Visual inspection

### 6.2 Usability

#### UX-USE-001: Navigation
- **Description:** Easy navigation between views
- **Priority:** Medium
- **Requirements:**
  - [ ] Clear navigation bar
  - [ ] Breadcrumbs or back links
  - [ ] Current location indicated
- **Verification:** Navigate through all views

#### UX-USE-002: Feedback
- **Description:** User actions provide feedback
- **Priority:** Medium
- **Requirements:**
  - [ ] Form submission feedback
  - [ ] Error messages are helpful
  - [ ] Success messages confirm actions
  - [ ] Loading indicators for async operations
- **Verification:** Test all user actions

### 6.3 Performance

#### UX-PERF-001: Page Load Time
- **Description:** Pages load quickly
- **Priority:** Medium
- **Requirements:**
  - [ ] Initial page load < 2 seconds
  - [ ] Subsequent navigation < 500ms
  - [ ] No blocking resources
- **Verification:** Browser dev tools, lighthouse

---

## 7. Deployment Requirements

### 7.1 Containerization

#### DR-CONT-001: Docker Image
- **Description:** Application packaged as Docker image
- **Priority:** High
- **Requirements:**
  - [ ] Multi-stage build for small image
  - [ ] Non-root user in container
  - [ ] Health check endpoint configured
  - [ ] All dependencies included
- **Verification:** Build and run container

#### DR-CONT-002: Docker Compose
- **Description:** Full stack deployment via compose
- **Priority:** High
- **Requirements:**
  - [ ] Application service
  - [ ] Database service
  - [ ] Migration service
  - [ ] Proper networking
  - [ ] Volume persistence
  - [ ] Health check dependencies
- **Verification:** docker-compose up succeeds

### 7.2 Infrastructure

#### DR-INFRA-001: Reverse Proxy
- **Description:** Traefik routing configured
- **Priority:** High
- **Requirements:**
  - [ ] Domain routing (quickquote.jdok.dev or similar)
  - [ ] TLS termination
  - [ ] Proper headers forwarded
- **Verification:** Access via domain

#### DR-INFRA-002: Database Persistence
- **Description:** Database data survives restarts
- **Priority:** Critical
- **Requirements:**
  - [ ] Named volume for PostgreSQL data
  - [ ] Backup strategy documented
- **Verification:** Restart containers, verify data

### 7.3 Operations

#### DR-OPS-001: Logging
- **Description:** Logs accessible for debugging
- **Priority:** High
- **Requirements:**
  - [ ] Structured JSON logs in production
  - [ ] Log level configurable
  - [ ] Logs accessible via docker logs
- **Verification:** Review production logs

#### DR-OPS-002: Health Checks
- **Description:** Health endpoint for monitoring
- **Priority:** High
- **Requirements:**
  - [ ] /health endpoint returns 200 when healthy
  - [ ] Database connectivity checked
  - [ ] Response includes status information
- **Verification:** Curl health endpoint

#### DR-OPS-003: Graceful Shutdown
- **Description:** Clean shutdown on SIGTERM
- **Priority:** Medium
- **Requirements:**
  - [ ] Finish in-flight requests
  - [ ] Close database connections
  - [ ] Timeout for forced shutdown
- **Verification:** Send SIGTERM, observe logs

### 7.4 Code Repository

#### DR-REPO-001: Public Code Repository
- **Description:** Code must be hosted in a public Git repository for verification
- **Priority:** Critical
- **Requirements:**
  - [ ] Code hosted on GitHub or GitLab
  - [ ] Repository is public (accessible without authentication)
  - [ ] All source code committed (no binary blobs)
  - [ ] .gitignore properly configured (no secrets committed)
  - [ ] README with setup instructions
- **Acceptance:** Repository URL provided and accessible
- **Verification:** Clone repository, review code, build from source
- **Deliverable:** Repository URL provided upon completion

---

## 8. Quality Standards

### 8.1 Code Standards

#### QS-CODE-001: Go Idioms
- **Description:** Code follows Go best practices
- **Priority:** High
- **Requirements:**
  - [ ] Effective Go guidelines followed
  - [ ] Standard project layout
  - [ ] Idiomatic error handling
  - [ ] Context propagation for cancellation
- **Verification:** Code review

#### QS-CODE-002: Documentation
- **Description:** Code is documented appropriately
- **Priority:** Medium
- **Requirements:**
  - [ ] Package documentation
  - [ ] Exported function documentation
  - [ ] Complex logic explained
  - [ ] README is comprehensive
- **Verification:** godoc review

### 8.2 Testing Standards

#### QS-TEST-001: Unit Tests
- **Description:** Critical business logic has unit tests
- **Priority:** Medium
- **Requirements:**
  - [ ] Service layer tested
  - [ ] Repository layer tested (with mocks)
  - [ ] Error paths tested
- **Verification:** go test ./... passes

#### QS-TEST-002: Integration Tests
- **Description:** Key integrations tested end-to-end
- **Priority:** Low
- **Requirements:**
  - [ ] Database operations tested
  - [ ] HTTP handlers tested
- **Verification:** Integration tests pass

### 8.3 Maintainability

#### QS-MAINT-001: Modularity
- **Description:** Code is modular and extensible
- **Priority:** High
- **Requirements:**
  - [ ] New providers can be added without major changes
  - [ ] Configuration can be extended
  - [ ] Features can be toggled
- **Verification:** Architecture review

#### QS-MAINT-002: Dependency Management
- **Description:** Dependencies are managed properly
- **Priority:** Medium
- **Requirements:**
  - [ ] go.mod and go.sum committed
  - [ ] No unnecessary dependencies
  - [ ] Versions pinned
- **Verification:** Review go.mod

---

## 9. Acceptance Criteria

### 9.1 Minimum Viable Demo

The following must be true for the project to be considered complete:

#### End-to-End Flow
- [ ] **AC-E2E-001:** A real person can call the phone number and talk to AI
- [ ] **AC-E2E-002:** The AI successfully gathers project information
- [ ] **AC-E2E-003:** Call transcript is stored and viewable
- [ ] **AC-E2E-004:** Quote is automatically generated from transcript
- [ ] **AC-E2E-005:** Authenticated user can view everything on dashboard

#### Technical Proof
- [ ] **AC-TECH-001:** Backend is written in Go (unfamiliar technology)
- [ ] **AC-TECH-002:** System is deployed and publicly accessible
- [ ] **AC-TECH-003:** Code demonstrates production-quality practices
- [ ] **AC-TECH-004:** Architecture is clean and maintainable

#### Verification Access
- [ ] **AC-ACCESS-001:** Verifier Credentials Provided
  - **Description:** Demo credentials must be provided for dashboard verification
  - **Requirements:**
    - [ ] Username (email) for verifier login
    - [ ] Password for verifier login
    - [ ] Credentials grant full dashboard access
  - **Acceptance:** Verifier can log in and inspect all calls, transcripts, and quotes
  - **Deliverable:** Username and password provided to verifier upon completion

### 9.2 Quality Gates

#### Before Deployment
- [ ] All critical and high priority requirements implemented
- [ ] No known security vulnerabilities
- [ ] Application starts without errors
- [ ] Health check passes

#### Before Demo
- [ ] Full end-to-end test call completed
- [ ] All dashboard features working
- [ ] Logs show no errors
- [ ] Performance is acceptable

---

## 10. Out of Scope

The following are explicitly NOT part of this project:

### 10.1 Features Not Included

- **User Registration:** Admin creates users manually
- **Multi-tenancy:** Single-tenant system
- **Outbound Calls:** Inbound only
- **Call Recording Playback:** Transcript only (unless Bland provides)
- **Payment Processing:** Quotes are informational only
- **Email Notifications:** No automated emails
- **API for External Consumers:** Dashboard only
- **Mobile App:** Web responsive only
- **Internationalization:** English only
- **Advanced Analytics:** Basic call list only

### 10.2 Technical Exclusions

- **Horizontal Scaling:** Single instance sufficient
- **Message Queues:** Direct processing
- **Caching Layer:** Database queries acceptable
- **Full-text Search:** Basic filtering only
- **Audit Logging:** Request logging sufficient
- **A/B Testing:** Not required

---

## 11. Verification Checklist

### 11.1 Pre-Deployment Checklist

```
[ ] Code compiles without errors
[ ] All migrations run successfully
[ ] Configuration validated
[ ] Docker image builds
[ ] Health check passes
[ ] Bland webhook URL configured
[ ] Anthropic API key working
[ ] Admin user created
```

### 11.2 Deployment Checklist

```
[ ] Docker containers running
[ ] Database accessible
[ ] Domain resolves correctly
[ ] HTTPS certificate valid
[ ] Application responds at domain
[ ] Health endpoint returns 200
```

### 11.3 Functional Test Checklist

```
[ ] Login works with valid credentials
[ ] Login fails with invalid credentials
[ ] Dashboard loads after login
[ ] Logout works
[ ] Protected routes redirect to login
[ ] Call test: dial number, complete conversation
[ ] Webhook received and processed
[ ] Transcript stored correctly
[ ] Quote generated automatically
[ ] Quote displayed on dashboard
[ ] Call detail page shows all data
[ ] Regenerate quote button works
[ ] Pagination works (if enough data)
```

### 11.4 Quality Checklist

```
[ ] No hardcoded secrets
[ ] Logs don't contain sensitive data
[ ] Error messages are user-friendly
[ ] Status codes are correct
[ ] Mobile responsive
[ ] Page loads < 2s
```

---

## 12. Final Deliverables

Upon project completion, the following deliverables must be provided to the verifier:

### 12.1 Deliverables Checklist

```
[ ] Public repository URL (GitHub/GitLab)
[ ] Live dashboard URL (HTTPS, publicly accessible)
[ ] Working phone number (Bland AI-provisioned, US-dialable)
[ ] Demo login credentials (email and password for verifier)
[ ] Project write-up (timeline, challenges, learnings)
```

### 12.2 Deliverable Details

#### DEL-001: Public Repository
- **Format:** URL to GitHub or GitLab repository
- **Contents:** All source code, configurations, documentation
- **Access:** Public (no authentication required to view)
- **Example:** `https://github.com/jkindrix/quickquote`

#### DEL-002: Live Dashboard URL
- **Format:** HTTPS URL to deployed application
- **Access:** Login page publicly accessible
- **Requirements:** Valid TLS certificate, responsive
- **Example:** `https://quickquote.jdok.dev`

#### DEL-003: Working Phone Number
- **Format:** US phone number in E.164 or standard format
- **Requirements:** Dialable from any phone, connects to AI agent
- **Example:** `+1 (555) 123-4567`

#### DEL-004: Demo Credentials
- **Format:** Email address and password
- **Access Level:** Full dashboard access (view all calls, transcripts, quotes)
- **Security Note:** Credentials created specifically for verification, can be rotated after review

#### DEL-005: Project Write-up
- **Format:** Markdown document or section in README
- **Required Sections:**
  - **Timeline:** How long the project took (start to finish)
  - **Challenges:** Technical obstacles encountered and how they were resolved
  - **Learnings:** Key insights from building in an unfamiliar technology (Go)
  - **AI Assistance:** How AI assistance was leveraged throughout the project
- **Purpose:** Provides context for evaluating the "AI-assisted development" claim

### 12.3 Delivery Format

All deliverables will be provided in a single summary message containing:

```markdown
## QuickQuote - Final Deliverables

**Repository:** https://github.com/jkindrix/quickquote
**Dashboard:** https://quickquote.jdok.dev
**Phone Number:** +1 (XXX) XXX-XXXX

**Demo Credentials:**
- Email: verifier@quickquote.demo
- Password: [provided securely]

**Write-up:** See WRITEUP.md in repository or below...
```

---

## Document History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2025-11-25 | System | Initial requirements specification |
| 1.1 | 2025-11-25 | System | Added: DR-REPO-001, AC-ACCESS-001, IR-BLAND-000, Section 12 |

---

## Notes

This document serves as the authoritative source for QuickQuote requirements. All implementation decisions should reference this document. Requirements marked as Critical must be satisfied for project success. High priority requirements should be satisfied. Medium and Low priority requirements are desirable but can be deferred if time-constrained.

**Remember:** The purpose of this project is to prove that with AI assistance, production-quality software can be built in unfamiliar technologies. The code quality and architecture matter as much as the features.
