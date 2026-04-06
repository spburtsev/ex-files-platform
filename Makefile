.PHONY: proto proto-lint proto-breaking \
        infra-up infra-down \
        backend-dev dev \
        test test-cover test-race test-e2e \
        build build-frontend build-backend \
        docker-build up down \
        logs-up logs-down

PROTO_DIR    := protocol
BACKEND_GEN  := ex-files-backend/gen
FRONTEND_GEN := ex-files-frontend/src/lib/gen
BACKEND_DIR  := ex-files-backend
FRONTEND_DIR := ex-files-frontend
E2E_DIR      := integration-testing

# --- Proto ---

proto:
	cd $(PROTO_DIR) && buf dep update && buf generate
	@echo "Go  → $(BACKEND_GEN)"
	@echo "TS  → $(FRONTEND_GEN)"

proto-lint:
	cd $(PROTO_DIR) && buf lint

proto-breaking:
	cd $(PROTO_DIR) && buf breaking --against '.git#subdir=$(PROTO_DIR)'

# --- Local development ---

infra-up:
	docker compose up -d ex-files-pg ex-files-minio ex-files-minio-init

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

build: proto build-frontend build-backend

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
