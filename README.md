# Parsa - Personal Finance Management API

A production-ready RESTful API for personal finance management built with Go and PostgreSQL. To go with Parsa-Go in Flutter.

## Features

- **Authentication**: OAuth 2.0 (Google) with JWT token-based sessions
- **User Management**: Profile management with OAuth integration
- **Accounts**: Manage multiple financial accounts (checking, savings, credit cards, etc.)
- **Transactions**: Track income and expenses with categorization
- **RESTful API**: Clean API design following REST principles
- **Standard Library**: Built with Go standard library (minimal dependencies)
- **Clean Architecture**: Separation of concerns with repository pattern

## Architecture

```
parsa-go/
├── cmd/api/              # Application entrypoint
├── internal/
│   ├── auth/            # OAuth & JWT logic
│   ├── config/          # Configuration management
│   ├── database/        # Database connection & repositories
│   ├── handlers/        # HTTP handlers
│   ├── middleware/      # HTTP middleware (auth, CORS, logging)
│   └── models/          # Domain models
└── migrations/          # Database migrations
```

## Tech Stack

- **Language**: Go (standard library)
- **Database**: PostgreSQL
- **Authentication**: OAuth 2.0 + JWT
- **Migration**: golang-migrate

## Setup

### Prerequisites

- Go 1.21+
- PostgreSQL 14+
- Google OAuth credentials

### 1. Clone and Install Dependencies

```bash
cd parsa-go
go mod download
```

### 2. Database Setup

Create a PostgreSQL database:

```sql
CREATE DATABASE parsa;
```

### 3. Environment Configuration

Create a `.env` file (see `.env.example`):

```bash
# Server
PORT=8080
HOST=0.0.0.0

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=yourpassword
DB_NAME=parsa
DB_SSLMODE=disable

# JWT
JWT_SECRET=your-super-secret-jwt-key-change-this

# Google OAuth
GOOGLE_CLIENT_ID=your-google-client-id
GOOGLE_CLIENT_SECRET=your-google-client-secret
GOOGLE_REDIRECT_URL=http://localhost:8080/auth/google/callback
```

### 4. Run Migrations

Install golang-migrate:

```bash
# macOS
brew install golang-migrate

# Linux
curl -L https://github.com/golang-migrate/migrate/releases/download/v4.16.2/migrate.linux-amd64.tar.gz | tar xvz
sudo mv migrate /usr/local/bin/
```

Run migrations:

```bash
migrate -path migrations -database "postgres://$DB_USER:$DB_PASSWORD$@localhost:5432/parsa?sslmode=disable" up
```

### 5. Run the Server

```bash
go run cmd/api/main.go
```

The server will start on `http://localhost:8080`

## API Endpoints

### Authentication

- `GET /auth/google/url` - Get Google OAuth authorization URL
- `POST /auth/google/callback` - Handle OAuth callback and issue JWT

### Users

- `GET /users/me` - Get current user profile (protected)

### Accounts

- `GET /accounts` - List all accounts (protected)
- `POST /accounts` - Create new account (protected)
- `GET /accounts/:id` - Get specific account (protected)
- `DELETE /accounts/:id` - Delete account (protected)

### Transactions

- `GET /transactions?account_id=xxx` - List transactions for account (protected)
- `POST /transactions` - Create new transaction (protected)
- `GET /transactions/:id` - Get specific transaction (protected)
- `DELETE /transactions/:id` - Delete transaction (protected)

### Health

- `GET /health` - Health check endpoint

## Authentication Flow

1. Client calls `GET /auth/google/url` to get authorization URL
2. User completes OAuth flow in browser
3. Client receives authorization code
4. Client calls `POST /auth/google/callback` with code
5. Server validates code, creates/finds user, returns JWT
6. Client includes JWT in `Authorization: Bearer <token>` header for protected routes

## Example Requests

### Login

```bash
# Get auth URL
curl http://localhost:8080/auth/google/url

# After OAuth flow, exchange code for JWT
curl -X POST http://localhost:8080/auth/google/callback \
  -H "Content-Type: application/json" \
  -d '{"code":"your-oauth-code","state":"state-from-url"}'
```

### Create Account

```bash
curl -X POST http://localhost:8080/accounts \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Chase Checking",
    "account_type": "checking",
    "currency": "USD",
    "balance": 1000.00
  }'
```

### Create Transaction

```bash
curl -X POST http://localhost:8080/transactions \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "account_id": "account-uuid",
    "amount": -50.00,
    "description": "Grocery shopping",
    "category": "Food",
    "transaction_date": "2025-01-15"
  }'
```

## Development

### Building

```bash
go build -o bin/parsa cmd/api/main.go
```

### Testing

```bash
go test ./...
```

## Production Deployment

### Build for Production

```bash
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o parsa cmd/api/main.go
```

### Systemd Service

Create `/etc/systemd/system/parsa.service`:

```ini
[Unit]
Description=Parsa Finance API
After=network.target postgresql.service

[Service]
Type=simple
User=www-data
WorkingDirectory=/opt/parsa
EnvironmentFile=/opt/parsa/.env
ExecStart=/opt/parsa/parsa
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

### Nginx Reverse Proxy

```nginx
server {
    listen 80;
    server_name api.yourdomain.com;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## License

MIT
