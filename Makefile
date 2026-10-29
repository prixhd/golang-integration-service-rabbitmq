.PHONY: up down logs test
.PHONY: test-integration

test-integration:
	docker compose --profile build build
	docker compose up -d --wait
	python -m pytest tests/tests_hw3.py -v

up:
	docker compose --profile build build
	docker compose up -d

down:
	docker compose down

logs:
	docker compose logs -f

test:
	go test ./...