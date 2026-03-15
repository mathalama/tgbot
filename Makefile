.PHONY: build run test clean help tidy

help:
	@echo "Available commands:"
	@echo "  make build       - Build the bot executable"
	@echo "  make run         - Run the bot"
	@echo "  make test        - Run tests"
	@echo "  make clean       - Clean build artifacts"
	@echo "  make tidy        - Tidy go modules"
	@echo "  make docker      - Build Docker image"
	@echo "  make docker-up   - Start Docker container"
	@echo "  make docker-down - Stop Docker container"
	@echo "  make fmt         - Format code"
	@echo "  make lint        - Run linter"
	@echo "  make deps        - Download dependencies"

build:
	go build -o bot.exe ./cmd/bot/main.go

run:
	go run ./cmd/bot/main.go

test:
	go test -v ./...

clean:
	rm -f bot.exe
	go clean

tidy:
	go mod tidy

docker:
	docker build -t telegram-vacancies-bot:latest .

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

fmt:
	go fmt ./...

lint:
	golangci-lint run

deps:
	go mod download
