# Parsa

A personal finance management API built with Go for Brazil's Open Finance Data Exchange Ecosystem. 

## Overview

Parsa is a RESTful API for managing personal finances — accounts, transactions, and bank synchronization via OpenFinance. The project prioritizes:

- **Layered Architecture** — Hexagonal design with strict dependency inversion
- **Minimal Dependencies** — Standard library for HTTP, JSON, and database access
- **No ORM** — Raw SQL for full control over queries
- **Production Patterns** — Context propagation, graceful shutdown, background job scheduling

## Features

- User authentication (JWT + OAuth 2.0 with Google)
- Multi-account support (checking, savings, credit cards, investments)
- Transaction tracking with categorization
- OpenFinance integration for automatic bank sync
- Background job scheduler with worker pool
- AES-256 encryption for sensitive data
- Argon2id password hashing

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     INTERFACES LAYER                        │
│  HTTP handlers, Scheduler                                   │
│  internal/interfaces/                                       │
└─────────────────────────┬───────────────────────────────────┘
                          │
┌─────────────────────────▼───────────────────────────────────┐
│                      DOMAIN LAYER                           │
│  Services, Models, Repository interfaces                    │
│  internal/domain/                                           │
└─────────────────────────┬───────────────────────────────────┘
                          │
┌─────────────────────────▼───────────────────────────────────┐
│                  INFRASTRUCTURE LAYER                       │
│  PostgreSQL repositories, OpenFinance client                │
│  internal/infrastructure/                                   │
└─────────────────────────────────────────────────────────────┘
```

Dependencies point inward. The domain layer has no knowledge of HTTP or databases.

## Project Structure

```
parsa/
├── cmd/api/main.go              # Entry point
├── internal/
│   ├── domain/                  # Business logic
│   │   ├── account/
│   │   ├── transaction/
│   │   ├── user/
│   │   └── openfinance/
│   ├── infrastructure/          # External adapters
│   │   ├── postgres/
│   │   ├── openfinance/
│   │   └── crypto/
│   ├── interfaces/              # Entry points
│   │   ├── http/
│   │   └── scheduler/
│   └── shared/                  # Cross-cutting concerns
│       ├── auth/
│       ├── config/
│       └── middleware/
├── migrations/
└── go.mod
```

## Tech Stack

| Component | Choice |
|-----------|--------|
| Language | Go (standard library) |
| Database | PostgreSQL 17+ |
| Driver | lib/pq |
| Encryption | golang.org/x/crypto |
| Migrations | golang-migrate |

Two external dependencies beyond the standard library.

## Getting Started

### Prerequisites

- Go 1.21+
- PostgreSQL 14+
- Google OAuth credentials 
- OpenFinance API credentials 

### Setup

1. Clone and install dependencies:

```bash
git clone https://github.com/lazaroborges/parsa-go.git
cd parsa-go
go mod download
```

2. Create database:

```sql
CREATE DATABASE parsa;
CREATE USER parsa_user WITH PASSWORD 'your_password';
GRANT ALL PRIVILEGES ON DATABASE parsa TO parsa_user;
```

3. Configure environment (copy `.env.example` to `.env`):

```bash
PORT=8080
DB_HOST=localhost
DB_PORT=5432
DB_USER=parsa_user
DB_PASSWORD=your_password
DB_NAME=parsa

JWT_SECRET=your-256-bit-secret
GOOGLE_CLIENT_ID=your-client-id
GOOGLE_CLIENT_SECRET=your-client-secret
```

4. Run migrations:

```bash
migrate -path migrations -database "postgresql://parsa_user:your_password@localhost:5432/parsa?sslmode=disable" up
```

5. Start the server:

```bash
go run cmd/api/main.go
```

## API Endpoints

### Authentication

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/auth/register` | Register with email/password |
| POST | `/api/auth/login` | Login |
| GET | `/api/auth/oauth/url` | Get OAuth URL |
| GET | `/api/auth/oauth/callback` | OAuth callback |

### Protected Routes

**Accounts**
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/accounts` | List accounts |
| GET | `/api/accounts/{id}` | Get account |
| POST | `/api/accounts` | Create account |
| DELETE | `/api/accounts/{id}` | Delete account |

**Transactions**
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/transactions` | List transactions |
| GET | `/api/transactions/{id}` | Get transaction |
| POST | `/api/transactions` | Create transaction |
| DELETE | `/api/transactions/{id}` | Delete transaction |

### Example

```bash
# Register
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "password", "name": "User"}'

# Create account
curl -X POST http://localhost:8080/api/accounts \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "Checking", "account_type": "checking", "currency": "BRL"}'
```

## Development

```bash
# Build
go build -o bin/parsa cmd/api/main.go

# Test
go test ./...

# Format
go fmt ./...

# Hot reload (requires Air)
air
```

## Deployment

Build for production:

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o parsa-api cmd/api/main.go
```

See the `/deployment` directory for systemd and nginx configuration examples.

## Background Jobs

The scheduler runs OpenFinance sync jobs at configured intervals:

```bash
SYNC_SCHEDULE_TIMES=02:00,14:00
```

Jobs execute concurrently via a worker pool with graceful shutdown support.

## Security

- JWT authentication (HS256)
- Argon2id password hashing
- AES-256-GCM for sensitive data encryption
- OAuth 2.0 with CSRF state validation
- CORS and security headers middleware

## Roadmap

- [ ] Improve Test coverage
- [ ] Additional OAuth provider (Apple)
- [ ] Spending analytics
- [ ] OpenAPI documentation
- [ ] Integration with Parsa's Proprietary Analytics Engine 

## License

MIT