.PHONY: up down logs test build

up:
	docker compose up --build -d

down:
	docker compose down

logs:
	docker compose logs -f

test:
	cd backend && go test ./...

build:
	cd frontend && npm ci && npm run build
	cd backend && CGO_ENABLED=0 go build -o bin/server ./cmd/server
