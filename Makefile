.PHONY: openapi-lint openapi-gen-go openapi-gen-ts openapi-gen openapi-docs \
        infra-up infra-down \
        backend-dev dev \
        test test-cover test-race test-e2e \
        build build-frontend build-backend \
        docker-build up down \
        logs-up logs-down \
        docs

API_DIR      := api
API_SPEC     := $(API_DIR)/openapi.yaml
BACKEND_GEN  := ex-files-backend/oapi
FRONTEND_GEN := ex-files-frontend/src/lib/api
BACKEND_DIR  := ex-files-backend
FRONTEND_DIR := ex-files-frontend
E2E_DIR      := integration-testing

# --- OpenAPI ---

openapi-lint:
	bunx --bun @redocly/cli@latest lint $(API_SPEC)

openapi-gen-go:
	ogen --target $(BACKEND_GEN) --package oapi --clean $(API_SPEC)
	@echo "Go server skeleton -> $(BACKEND_GEN)"

openapi-gen-ts:
	cd $(FRONTEND_DIR) && bun run gen:api
	@echo "TS client -> $(FRONTEND_GEN)"

openapi-gen: openapi-gen-go openapi-gen-ts

openapi-docs:
	mkdir -p docs/api
	bunx --bun @redocly/cli@latest build-docs $(API_SPEC) -o docs/api/index.html

# --- Documentation ---

docs: openapi-docs
	@echo "API docs -> docs/api/index.html"

# --- Local development ---

infra-up:
	docker compose up -d ex-files-pg redis ex-files-minio ex-files-minio-init

infra-down:
	docker compose down

backend-dev:
	cd $(BACKEND_DIR) && air

dev: infra-up backend-dev

# --- Testing ---

test:
	cd $(BACKEND_DIR) && go test ./... -v -count=1

test-cover:
	cd $(BACKEND_DIR) && go test ./... -coverprofile=coverage.out -covermode=atomic
	cd $(BACKEND_DIR) && go tool cover -func=coverage.out

test-race:
	cd $(BACKEND_DIR) && go test ./... -race -count=1

test-e2e:
	cd $(E2E_DIR) && bun run test

# --- Production build ---

build-frontend:
	cd $(FRONTEND_DIR) && bun install --frozen-lockfile && bun run build

build-backend:
	cd $(BACKEND_DIR) && CGO_ENABLED=0 go build -o bin/server .

build: openapi-gen build-frontend build-backend

# --- Docker ---

docker-build:
	docker compose build ex-files-backend ex-files-frontend

up: docker-build
	docker compose up -d

down:
	docker compose down

# --- Observability ---

logs-up:
	docker compose up -d loki promtail grafana

logs-down:
	docker compose stop loki promtail grafana
