# Load environment variables from .env
include .env
export

# Run the application
run:
	go run ./cmd/api/

dev:
	air

test:
	go test ./...

# Build the application
build:
	go build -o bin/parsa ./cmd/api/

# Run migrations up
migrate-up:
	migrate -path migrations -database "postgresql://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)" up

# Run migrations down
migrate-down:
	migrate -path migrations -database "postgresql://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)" down

# Create database
db-create:
	createdb -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) -O $(DB_USER) $(DB_NAME)

# Drop database (terminates active connections first)
db-drop:
	psql -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) -d postgres -c "SELECT pg_terminate_backend(pg_stat_activity.pid) FROM pg_stat_activity WHERE pg_stat_activity.datname = '$(DB_NAME)' AND pid <> pg_backend_pid();" || true
	dropdb -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) $(DB_NAME) || true

# Reset database (drop, create, migrate)
db-reset: db-drop db-create migrate-up

.PHONY: run build migrate-up migrate-down db-create db-drop db-reset