.PHONY: build run dev install clean test test-e2e test-server test-quick help

# Default target
.DEFAULT_GOAL := help

# Build the Go backend
build:
	go build -o sentry-alt .

# Install Go dependencies
install-go:
	go mod download

# Install frontend dependencies
install-frontend:
	cd frontend && npm install

# Install all dependencies
install: install-go install-frontend

# Build frontend
build-frontend:
	cd frontend && npm run build

# Run the backend server
run: build
	./sentry-alt

# Run in development mode (requires frontend dev server running separately)
dev:
	go run .

# Clean build artifacts
clean:
	rm -f sentry-alt
	rm -rf frontend/node_modules
	rm -rf frontend/public
	rm -f *.db *.sqlite *.sqlite3

# Full production build
build-all: build-frontend build
	@echo "Build complete! Run ./sentry-alt to start the server"

# Check if server is running
check-server:
	@echo "🔍 Checking if server is running..."
	@curl -s -f http://localhost:8080/api/health > /dev/null 2>&1 || \
		(echo "❌ Server is not running. Start it with: make run" && exit 1)
	@echo "✅ Server is running"

# Run end-to-end tests (requires server to be running)
test-e2e: check-server
	@echo "🧪 Running end-to-end test suite..."
	@chmod +x test.sh
	@./test.sh http://localhost:8080

# Run end-to-end tests with custom URL
test-e2e-url: check-server
	@if [ -z "$(URL)" ]; then \
		echo "❌ Please provide URL: make test-e2e-url URL=http://localhost:8080"; \
		exit 1; \
	fi
	@echo "🧪 Running end-to-end test suite against $(URL)..."
	@chmod +x test.sh
	@./test.sh $(URL)

# Start server in background, run tests, then stop server
test: build
	@echo "🚀 Starting server for testing..."
	@if [ -f .env ]; then \
		set -a; \
		. .env; \
		set +a; \
	fi
	@ADMIN_EMAIL=$${ADMIN_EMAIL:-admin@example.com} \
	 ADMIN_PASSWORD=$${ADMIN_PASSWORD:-admin123} \
	 JWT_SECRET=$${JWT_SECRET:-test-secret} \
	 PORT=$${PORT:-8080} \
	 ./sentry-alt > /tmp/sentry-alt-test.log 2>&1 & \
		echo $$! > /tmp/sentry-alt-test.pid
	@echo "⏳ Waiting for server to start..."
	@for i in 1 2 3 4 5 6 7 8 9 10; do \
		curl -s -f http://localhost:8080/api/health > /dev/null 2>&1 && break || sleep 1; \
	done
	@curl -s -f http://localhost:8080/api/health > /dev/null 2>&1 || \
		(echo "❌ Server failed to start. Check logs: tail -f /tmp/sentry-alt-test.log" && exit 1)
	@echo "✅ Server started. Running tests..."
	@chmod +x test.sh
	@if [ -f .env ]; then \
		set -a; \
		. .env; \
		set +a; \
	fi
	@ADMIN_EMAIL=$${ADMIN_EMAIL:-admin@example.com} \
	 ADMIN_PASSWORD=$${ADMIN_PASSWORD:-admin123} \
	 ./test.sh http://localhost:8080 || (echo "❌ Tests failed" && make test-stop && exit 1)
	@echo "✅ All tests passed!"
	@make test-stop

# Stop test server
test-stop:
	@if [ -f /tmp/sentry-alt-test.pid ]; then \
		kill $$(cat /tmp/sentry-alt-test.pid) 2>/dev/null || true; \
		rm -f /tmp/sentry-alt-test.pid; \
		echo "🛑 Test server stopped"; \
	fi

# Quick test (just health check and basic API)
test-quick: check-server
	@echo "🧪 Running quick tests..."
	@echo "📡 Testing health endpoint..."
	@curl -s http://localhost:8080/api/health | grep -q "ok" && echo "✅ Health check passed" || (echo "❌ Health check failed" && exit 1)
	@echo "✅ Quick tests passed"

# Send test data (tracing, coverage, errors)
test-data:
	@echo "🧪 Sending test data to Pulse..."
	@if [ -z "$(PROJECT_ID)" ] || [ -z "$(API_KEY)" ]; then \
		echo "❌ Please provide PROJECT_ID and API_KEY:"; \
		echo "   make test-data PROJECT_ID=xxx API_KEY=yyy"; \
		echo ""; \
		echo "   Or get them from the API:"; \
		echo "   TOKEN=\$$(curl -s -X POST http://localhost:8080/api/auth/login \\"; \
		echo "     -H 'Content-Type: application/json' \\"; \
		echo "     -d '{\"email\":\"admin@example.com\",\"password\":\"admin\"}' | grep -o '\"token\":\"[^\"]*' | cut -d'\"' -f4)"; \
		echo "   PROJECT=\$$(curl -s -X GET http://localhost:8080/api/projects \\"; \
		echo "     -H \"Authorization: Bearer \$$TOKEN\" | grep -o '\"id\":\"[^\"]*' | head -1 | cut -d'\"' -f4)"; \
		echo "   KEY=\$$(curl -s -X GET http://localhost:8080/api/projects/\$$PROJECT \\"; \
		echo "     -H \"Authorization: Bearer \$$TOKEN\" | grep -o '\"api_key\":\"[^\"]*' | cut -d'\"' -f4)"; \
		echo "   make test-data PROJECT_ID=\$$PROJECT API_KEY=\$$KEY"; \
		exit 1; \
	fi
	@chmod +x test-data-send.sh
	@./test-data-send.sh http://localhost:8080 $(PROJECT_ID) $(API_KEY)

# Help target
help:
	@echo "Pulse OSS - Makefile Commands"
	@echo ""
	@echo "Build & Run:"
	@echo "  make build          - Build Go backend"
	@echo "  make build-frontend - Build frontend"
	@echo "  make build-all      - Build both frontend and backend"
	@echo "  make run            - Build and run server"
	@echo "  make dev            - Run server in dev mode (go run)"
	@echo ""
	@echo "Testing:"
	@echo "  make test           - Start server, run full E2E tests, stop server"
	@echo "  make test-e2e       - Run E2E tests (server must be running)"
	@echo "  make test-e2e-url   - Run E2E tests against custom URL"
	@echo "                       Usage: make test-e2e-url URL=http://localhost:8080"
	@echo "  make test-quick     - Run quick health check tests"
	@echo "  make test-data      - Send test data (errors, traces, coverage)"
	@echo "                       Usage: make test-data PROJECT_ID=xxx API_KEY=yyy"
	@echo "  make test-stop      - Stop test server if running"
	@echo ""
	@echo "Setup:"
	@echo "  make install        - Install all dependencies"
	@echo "  make install-go     - Install Go dependencies"
	@echo "  make install-frontend - Install frontend dependencies"
	@echo ""
	@echo "Cleanup:"
	@echo "  make clean          - Remove build artifacts and databases"
	@echo ""
	@echo "Examples:"
	@echo "  make test           # Full automated test (starts/stops server)"
	@echo "  make run &          # Start server in background"
	@echo "  make test-e2e       # Run tests against running server"
	@echo "  make test-data PROJECT_ID=xxx API_KEY=yyy  # Send test data"