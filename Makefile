.PHONY: up down logs test

up:
	docker compose --profile build build
	docker compose up -d

down:
	docker compose down

logs:
	docker compose logs -f

test:
	go test ./...