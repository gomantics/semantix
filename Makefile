.PHONY: dev build clean cfgx sqlc deps docker-up docker-down help

# Default target
help:
	@echo "Available targets:"
	@echo "  dev         - Run the API server in development mode"
	@echo "  clean       - Remove build artifacts"
	@echo "  deps        - Download and tidy dependencies"
	@echo "  docker-up   - Start all Docker services"
	@echo "  docker-down - Stop all Docker services"
	@echo "  gen         - Run all code generation"

# Run the API server in development mode
dev:
	go tool air

# Remove build artifacts
clean:
	rm -rf bin/
	rm -rf tmp/

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
gen: 
	go generate ./...

