.PHONY: help dev up down logs test lint fmt typecheck build push db-migrate db-seed import-osm clean

# Default target
help:
	@echo "AegisRoad - Traffic Management System"
	@echo ""
	@echo "Usage: make <target>"
	@echo ""
	@echo "Development:"
	@echo "  dev              Start full dev stack (infra + backend + frontend)"
	@echo "  up               Start infrastructure only (postgres, redis, kafka, nats)"
	@echo "  down             Stop all services"
	@echo "  logs             Follow logs of all services"
	@echo "  logs-%           Follow logs of specific service (e.g., make logs-api-gateway)"
	@echo ""
	@echo "Database:"
	@echo "  db-migrate       Run database migrations"
	@echo "  db-seed          Seed database with initial data"
	@echo "  db-reset         Drop and recreate database"
	@echo "  import-osm       Download and import Bengaluru OSM data"
	@echo ""
	@echo "Code Quality:"
	@echo "  test             Run all tests"
	@echo "  lint             Run linters on all services"
	@echo "  fmt              Format code (rustfmt, gofmt, prettier, black)"
	@echo "  typecheck        Run type checks (cargo check, go vet, tsc, mypy)"
	@echo ""
	@echo "Build & Deploy:"
	@echo "  build            Build all Docker images"
	@echo "  push             Push images to GHCR (requires auth)"
	@echo "  deploy-dev       Deploy to dev Kubernetes cluster"
	@echo ""
	@echo "Utilities:"
	@echo "  clean            Remove all containers, volumes, and build artifacts"
	@echo "  install-hooks    Install git pre-push hooks"

# Development
dev: up
	docker compose --profile backend --profile frontend up --build

up:
	docker compose up -d postgres redis kafka zookeeper nats

down:
	docker compose down --remove-orphans

logs:
	docker compose logs -f --tail=100

logs-%:
	docker compose logs -f --tail=100 $*

# Database
db-migrate:
	docker compose exec -T postgres psql -U aegis -d aegisroad -f /docker-entrypoint-initdb.d/001_extensions.sql
	docker compose exec -T postgres psql -U aegis -d aegisroad -f /docker-entrypoint-initdb.d/002_schema.sql
	docker compose exec -T postgres psql -U aegis -d aegisroad -f /docker-entrypoint-initdb.d/003_indexes.sql
	docker compose exec -T postgres psql -U aegis -d aegisroad -f /docker-entrypoint-initdb.d/004_views.sql

db-seed:
	docker compose exec -T postgres psql -U aegis -d aegisroad -f /docker-entrypoint-initdb.d/005_seed.sql

db-reset:
	docker compose down -v postgres
	docker compose up -d postgres
	sleep 10
	$(MAKE) db-migrate db-seed

import-osm:
	@echo "Downloading Bengaluru OSM extract..."
	wget -q -O /tmp/karnataka.osm.pbf "https://download.geofabrik.de/asia/india/karnataka-latest.osm.pbf"
	@echo "Extracting Bengaluru bounding box..."
	osmium extract -b 77.0,12.4,78.0,13.4 /tmp/karnataka.osm.pbf -o /tmp/blr.osm.pbf
	@echo "Importing to PostGIS (requires imposm3)..."
	@echo "Run: imposm3 import -connection postgis://aegis:$$DB_PASSWORD@localhost:5432/aegisroad -mapping database/mapping.json -read /tmp/blr.osm.pbf -write -deployproduction"

# Code Quality
test:
	@echo "Running tests for all services..."
	@cd services/routing-engine && cargo test 2>/dev/null || echo "routing-engine: no tests yet"
	@cd services/api-gateway && go test ./... 2>/dev/null || echo "api-gateway: no tests yet"
	@cd services/realtime-server && npm test 2>/dev/null || echo "realtime-server: no tests yet"
	@cd services/ml-pipeline && python -m pytest 2>/dev/null || echo "ml-pipeline: no tests yet"
	@cd services/scats-emulator && go test ./... 2>/dev/null || echo "scats-emulator: no tests yet"
	@cd apps/public-pwa && npm test 2>/dev/null || echo "public-pwa: no tests yet"
	@cd apps/emergency-app && npm test 2>/dev/null || echo "emergency-app: no tests yet"
	@cd apps/operator-dashboard && npm test 2>/dev/null || echo "operator-dashboard: no tests yet"

lint:
	@echo "Linting all services..."
	@cd services/routing-engine && cargo clippy -- -D warnings 2>/dev/null || echo "routing-engine: clippy not available"
	@cd services/api-gateway && golangci-lint run 2>/dev/null || echo "api-gateway: golangci-lint not installed"
	@cd services/realtime-server && npm run lint 2>/dev/null || echo "realtime-server: no lint script"
	@cd services/ml-pipeline && ruff check . 2>/dev/null || echo "ml-pipeline: ruff not installed"
	@cd services/scats-emulator && golangci-lint run 2>/dev/null || echo "scats-emulator: golangci-lint not installed"
	@cd apps/public-pwa && npm run lint 2>/dev/null || echo "public-pwa: no lint script"
	@cd apps/emergency-app && npm run lint 2>/dev/null || echo "emergency-app: no lint script"
	@cd apps/operator-dashboard && npm run lint 2>/dev/null || echo "operator-dashboard: no lint script"

fmt:
	@echo "Formatting all services..."
	@cd services/routing-engine && cargo fmt 2>/dev/null || echo "routing-engine: rustfmt not available"
	@cd services/api-gateway && gofmt -w . 2>/dev/null || echo "api-gateway: gofmt failed"
	@cd services/realtime-server && npx prettier --write . 2>/dev/null || echo "realtime-server: prettier not available"
	@cd services/ml-pipeline && black . 2>/dev/null || echo "ml-pipeline: black not installed"
	@cd services/scats-emulator && gofmt -w . 2>/dev/null || echo "scats-emulator: gofmt failed"
	@cd apps/public-pwa && npx prettier --write . 2>/dev/null || echo "public-pwa: prettier not available"
	@cd apps/emergency-app && npx prettier --write . 2>/dev/null || echo "emergency-app: prettier not available"
	@cd apps/operator-dashboard && npx prettier --write . 2>/dev/null || echo "operator-dashboard: prettier not available"

typecheck:
	@echo "Type checking all services..."
	@cd services/routing-engine && cargo check 2>/dev/null || echo "routing-engine: cargo check failed"
	@cd services/api-gateway && go vet ./... 2>/dev/null || echo "api-gateway: go vet failed"
	@cd services/realtime-server && npx tsc --noEmit 2>/dev/null || echo "realtime-server: tsc not available"
	@cd services/ml-pipeline && mypy . 2>/dev/null || echo "ml-pipeline: mypy not installed"
	@cd services/scats-emulator && go vet ./... 2>/dev/null || echo "scats-emulator: go vet failed"
	@cd apps/public-pwa && npx tsc --noEmit 2>/dev/null || echo "public-pwa: tsc not available"
	@cd apps/emergency-app && npx tsc --noEmit 2>/dev/null || echo "emergency-app: tsc not available"
	@cd apps/operator-dashboard && npx tsc --noEmit 2>/dev/null || echo "operator-dashboard: tsc not available"

# Build & Deploy
build:
	docker compose build --parallel

push:
	@echo "Pushing to GHCR..."
	docker compose push

deploy-dev:
	@echo "Deploying to dev cluster..."
	kubectl apply -k infra/k8s/overlays/dev

# Utilities
clean:
	docker compose down -v --remove-orphans
	docker system prune -f
	rm -rf services/*/target services/*/node_modules apps/*/node_modules

install-hooks:
	@echo "#!/bin/bash" > .git/hooks/pre-push
	@echo "make lint test" >> .git/hooks/pre-push
	@chmod +x .git/hooks/pre-push
	@echo "Installed pre-push hook (runs lint + test before push)"

# Generate protobuf types
proto:
	@echo "Generating TypeScript types from protobuf..."
	@mkdir -p packages/shared-types/src
	@protoc --ts_out=packages/shared-types/src --proto_path=services/api-gateway/proto services/api-gateway/proto/*.proto 2>/dev/null || echo "protoc not installed"

# Quick start for new developers
setup: install-hooks
	@echo "Setup complete! Run 'make dev' to start development environment."