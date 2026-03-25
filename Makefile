.PHONY: proto proto-lint proto-breaking \
        infra-up infra-down \
        backend-dev dev

PROTO_DIR    := protocol
BACKEND_GEN  := ex-files-backend/gen
FRONTEND_GEN := ex-files-frontend/src/lib/gen
BACKEND_DIR  := ex-files-backend

# ── Proto ──────────────────────────────────────────────────────────────────────

proto:
	cd $(PROTO_DIR) && buf dep update && buf generate
	@echo "Go  → $(BACKEND_GEN)"
	@echo "TS  → $(FRONTEND_GEN)"

proto-lint:
	cd $(PROTO_DIR) && buf lint

proto-breaking:
	cd $(PROTO_DIR) && buf breaking --against '.git#subdir=$(PROTO_DIR)'

# ── Infrastructure (dev only) ──────────────────────────────────────────────────

## Start PostgreSQL + MinIO in the background
infra-up:
	docker compose up -d ex-files-pg ex-files-minio ex-files-minio-init

## Stop and remove infra containers (data volumes are preserved)
infra-down:
	docker compose down

# ── Development ────────────────────────────────────────────────────────────────

## Run the Go backend with air hot-reload (install air first: go install github.com/air-verse/air@latest)
backend-dev:
	cd $(BACKEND_DIR) && air

## Start infra then run the backend with hot-reload
dev: infra-up backend-dev
