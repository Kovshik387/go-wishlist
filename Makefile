.PHONY: dev down logs build test frontend-test backend-test lint webhook backup

dev:
	docker compose up --build

down:
	docker compose down

logs:
	docker compose logs -f app worker

build:
	docker build --target runtime -t wishtrack:local .

test:
	docker build --target test .

frontend-test:
	cd web && npm test

backend-test:
	go test ./...

lint:
	go vet ./...
	cd web && npm run lint

webhook:
	powershell -ExecutionPolicy Bypass -File scripts/setup-webhook.ps1

backup:
	powershell -ExecutionPolicy Bypass -File scripts/backup.ps1
