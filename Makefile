.PHONY: all build test clean run docker-build docker-push deploy-aws generate-mocks version-check

# Variables
DOCKER_USERNAME ?= yourdockerhubusername
IMAGE_TAG ?= latest
AWS_REGION ?= us-east-1
EC2_HOST ?= your-ec2-instance.amazonaws.com
GO_VERSION = 1.24.5

# Service names
SERVICES = gateway analyzer link-checker

all: clean build test

version-check:
	@echo "Checking Go version..."
	@./scripts/check-go-version.sh

build: version-check
	@echo "Building services..."
	@for service in $(SERVICES); do \
		echo "Building $service..."; \
		cd services/$service && go build -o ../../bin/$service ./... && cd ../..; \
	done

test: test-unit test-integration

test-unit:
	@echo "Running unit tests..."
	@go test -v -short ./...

test-integration:
	@echo "Running integration tests..."
	@go test -v -run Integration ./...

test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@go clean -testcache

clean-modules:
	@echo "Cleaning and regenerating go.sum..."
	@rm -f go.sum
	@go mod download
	@go mod tidy
	@echo "✓ go.sum regenerated"

run: build
	@echo "Starting services..."
	@trap 'kill %1 %2 %3' EXIT; \
	./bin/gateway & \
	./bin/analyzer & \
	./bin/link-checker & \
	wait

docker-build:
	@echo "Building Docker images..."
	@for service in $(SERVICES); do \
		echo "Building $$service image..."; \
		docker build -t $(DOCKER_USERNAME)/webpage-analyzer-$$service:$(IMAGE_TAG) \
			-f services/$$service/Dockerfile .; \
	done

docker-push: docker-build
	@echo "Pushing Docker images..."
	@for service in $(SERVICES); do \
		echo "Pushing $$service image..."; \
		docker push $(DOCKER_USERNAME)/webpage-analyzer-$$service:$(IMAGE_TAG); \
	done

docker-compose-up:
	@echo "Starting services with Docker Compose..."
	@docker-compose up --build

docker-compose-down:
	@echo "Stopping services..."
	@docker-compose down

generate-mocks:
	@echo "Generating mocks..."
	@go generate ./...

lint:
	@echo "Running linters..."
	@golangci-lint run ./...

fmt:
	@echo "Formatting code..."
	@go fmt ./...

deploy-aws:
	@echo "Deploying to AWS EC2..."
	@ssh ec2-user@$(EC2_HOST) 'cd /home/ec2-user/webpage-analyzer && git pull && docker-compose down && docker-compose up -d --build'

# Development helpers
dev-setup:
	@echo "Setting up development environment..."
	@go mod download
	@go install github.com/golang/mock/mockgen@v1.6.0
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Monitoring
monitor-cpu:
	@echo "Starting CPU profiling..."
	@go tool pprof http://localhost:8080/debug/pprof/profile

monitor-heap:
	@echo "Starting heap profiling..."
	@go tool pprof http://localhost:8080/debug/pprof/heap