.PHONY: dev build clean cfgx sqlc deps docker-up docker-down help

# Default target
help:
	@echo "Available targets:"
	@echo "  dev         - Run the API server in development mode"
	@echo "  build       - Build the API binary"
	@echo "  clean       - Remove build artifacts"
	@echo "  cfgx        - Generate config code from config.toml"
	@echo "  sqlc        - Generate database code from SQL"
	@echo "  deps        - Download and tidy dependencies"
	@echo "  docker-up   - Start all Docker services"
	@echo "  docker-down - Stop all Docker services"
	@echo "  gen         - Run all code generation (cfgx + sqlc)"

# Run the API server in development mode
dev:
	go run ./cmd/api

# Build the API binary
build:
	go build -o bin/api ./cmd/api

# Remove build artifacts
clean:
	rm -rf bin/
	rm -rf tmp/

# Generate config code from config.toml
cfgx:
	go tool cfgx generate --in config/config.toml --out config/config.gen.go --pkg config --mode getter

# Generate database code from SQL
sqlc:
	go tool sqlc generate

# Download and tidy dependencies
deps:
	go mod download
	go mod tidy

# Start all Docker services
docker-up:
	docker compose up -d

# Stop all Docker services
docker-down:
	docker compose down

# Run all code generation
gen: cfgx sqlc

