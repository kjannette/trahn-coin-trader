.PHONY: build run test lint format clean help db-setup db-migrate

BINARY := server
CMD    := ./cmd/server

# Default target
help:
	@echo "Trahn Grid Trader - Available Commands"
	@echo "======================================="
	@echo "  make build       Build the server binary"
	@echo "  make run         Build and run the trading bot"
	@echo "  make dev         Run directly without building (go run)"
	@echo "  make test        Run all tests"
	@echo "  make test-v      Run all tests (verbose)"
	@echo "  make lint        Check code formatting (go vet + staticcheck)"
	@echo "  make format      Format all Go source files"
	@echo "  make clean       Remove build artifacts"
	@echo "  make db-setup    Create database and run schema"
	@echo "  make db-migrate  Run pending migrations"
	@echo ""

build:
	go build -o $(BINARY) $(CMD)

run: build
	./$(BINARY)

dev:
	go run $(CMD)

test:
	go test ./...

test-v:
	go test -v ./...

lint:
	go vet ./...
	@which staticcheck > /dev/null 2>&1 && staticcheck ./... || echo "staticcheck not installed, skipping (go install honnef.co/go/tools/cmd/staticcheck@latest)"

format:
	gofmt -w .

clean:
	rm -f $(BINARY)
	@echo "Cleaned!"

db-setup:
	psql -U postgres -f db/setup.sql
	psql -U postgres -d trahn_grid_trader -f db/schema.sql
	@echo "Database setup complete"

db-migrate:
	@for f in db/migrations/*.sql; do \
		echo "Running $$f ..."; \
		psql -U postgres -d trahn_grid_trader -f "$$f"; \
	done
	@echo "Migrations complete"
