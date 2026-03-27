.PHONY: proto proto-lint proto-breaking \
        infra-up infra-down \
        backend-dev dev

PROTO_DIR    := protocol
BACKEND_GEN  := ex-files-backend/gen
FRONTEND_GEN := ex-files-frontend/src/lib/gen
BACKEND_DIR  := ex-files-backend

proto:
	cd $(PROTO_DIR) && buf dep update && buf generate
	@echo "Go  → $(BACKEND_GEN)"
	@echo "TS  → $(FRONTEND_GEN)"

proto-lint:
	cd $(PROTO_DIR) && buf lint

proto-breaking:
	cd $(PROTO_DIR) && buf breaking --against '.git#subdir=$(PROTO_DIR)'

infra-up:
	docker compose up -d ex-files-pg ex-files-minio ex-files-minio-init

infra-down:
	docker compose down

backend-dev:
	cd $(BACKEND_DIR) && air

dev: infra-up backend-dev
