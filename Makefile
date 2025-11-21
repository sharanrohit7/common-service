.PHONY: build test lint clean run-example docker-build

# Build the example service
build:
	@echo "Building example service..."
	@go build -o bin/example-service ./cmd/example-service

# Run all tests
test:
	@echo "Running tests..."
	@go test -v -race -coverprofile=coverage.out ./...

# Run tests with coverage report
test-coverage:
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run linter
lint:
	@echo "Running linter..."
	@golangci-lint run

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html

# Run the example service locally
run-example:
	@echo "Running example service..."
	@go run ./cmd/example-service

# Build Docker image for example service
docker-build:
	@echo "Building Docker image..."
	@docker build -t go-service-kit-example:latest -f cmd/example-service/Dockerfile .

# Install dependencies
deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Vet code
vet:
	@echo "Running go vet..."
	@go vet ./...

