.PHONY: proto proto-lint proto-breaking

PROTO_DIR    := protocol
BACKEND_GEN  := ex-files-backend/gen
FRONTEND_GEN := ex-files-frontend/src/lib/gen

proto:
	cd $(PROTO_DIR) && buf dep update && buf generate
	@echo "Go  → $(BACKEND_GEN)"
	@echo "TS  → $(FRONTEND_GEN)"

proto-lint:
	cd $(PROTO_DIR) && buf lint

proto-breaking:
	cd $(PROTO_DIR) && buf breaking --against '.git#subdir=$(PROTO_DIR)'
