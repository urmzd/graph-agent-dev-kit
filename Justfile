# graph-agent-dev-kit development commands

# List available recipes
default:
    @just --list

# Run all tests
test:
    go test ./...

# Run tests with coverage report
test-cover:
    go test -coverprofile=coverage.out -covermode=atomic ./...
    go tool cover -func=coverage.out

# Run golangci-lint
lint:
    golangci-lint run ./...

# Run go vet
vet:
    go vet ./...

# Format code
fmt:
    gofmt -w .

# Build all packages
build:
    go build ./...

# Run govulncheck
vuln:
    govulncheck ./...

# Tidy modules
tidy:
    go mod tidy

# Run full CI locally (lint + vet + test)
check: lint vet test

# Run benchmarks
bench:
    go test -bench=. -benchmem ./...

# Run fuzz tests for a specific package and function
fuzz PACKAGE FUNC DURATION="30s":
    go test -fuzz={{FUNC}} -fuzztime={{DURATION}} {{PACKAGE}}

# Build docker image
docker-build:
    docker build -t graph-agent-dev-kit .

# Clean build artifacts
clean:
    rm -f coverage.out
    go clean -cache -testcache
