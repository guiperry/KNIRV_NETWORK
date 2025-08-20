BINARY_NAME=crypto-wallet-backend
DOCKER_IMAGE=crypto-wallet-backend
VERSION=latest

.PHONY: build run test clean docker-build docker-run

# Build the application
build:
	go build -o bin/$(BINARY_NAME) cmd/server/main.go

# Run the application
run:
	go run cmd/server/main.go

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	go clean
	rm -f bin/$(BINARY_NAME)

# Install dependencies
deps:
	go mod download
	go mod tidy

# Run with hot reload (requires air)
dev:
	air

# Database migrations
migrate-up:
	migrate -path migrations -database "$(DATABASE_URL)\" up

migrate-down:
	migrate -path migrations -database "$(DATABASE_URL)\" down

# Docker commands
docker-build:
	docker build -t $(DOCKER_IMAGE):$(VERSION) .

docker-run:
	docker run -p 8080:8080 --env-file .env $(DOCKER_IMAGE):$(VERSION)

# Android APK build (requires gomobile)
android-build:
	gomobile bind -target=android -o android/app.aar ./pkg/mobile

# Generate API documentation
docs:
	swag init -g cmd/server/main.go

# Security scan
security-scan:
	gosec ./...

# Lint code
lint:
	golangci-lint run

# Format code
fmt:
	go fmt ./...

# Generate mocks
mocks:
	mockgen -source=internal/services/interfaces.go -destination=mocks/services.go

# Load test
load-test:
	k6 run tests/load/script.js

# Deploy to production
deploy:
	docker build -t $(DOCKER_IMAGE):$(VERSION) .
	docker push $(DOCKER_IMAGE):$(VERSION)
	kubectl apply -f k8s/

.DEFAULT_GOAL := build