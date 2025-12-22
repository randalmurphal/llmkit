.PHONY: all build test lint clean coverage

# Default target
all: lint test

# Build all packages
build:
	go build ./...

# Run tests
test:
	go test -race ./...

# Run tests with coverage
coverage:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run linter
lint:
	golangci-lint run

# Format code
fmt:
	gofmt -s -w .
	goimports -w .

# Tidy dependencies
tidy:
	go mod tidy

# Clean build artifacts
clean:
	rm -f coverage.out coverage.html
	go clean

# Verify (build + test + lint)
verify: build test lint
	@echo "All checks passed!"
