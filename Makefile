# Load environment variables from .env
include .env
export

# Run the application
run:
	go run cmd/api/main.go

dev:
	air

# Build the application
build:
	go build -o bin/parsa cmd/api/main.go

# Run migrations up
migrate-up:
	migrate -path migrations -database "postgresql://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)" up

# Run migrations down
migrate-down:
	migrate -path migrations -database "postgresql://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)" down

# Create database
db-create:
	createdb -O $(DB_USER) $(DB_NAME)

# Drop database
db-drop:
	dropdb $(DB_NAME)

# Reset database (drop, create, migrate)
db-reset: db-drop db-create migrate-up

.PHONY: run build migrate-up migrate-down db-create db-drop db-reset