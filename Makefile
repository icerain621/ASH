# Detect backend location (pre/post reorganize)
BACKEND_DIR := .
ifneq ($(wildcard backend/go.mod),)
BACKEND_DIR := backend
endif

.PHONY: run test swagger tidy doctor cli web web-build web-dev

run:
	cd $(BACKEND_DIR) && go run ./cmd/worker

test:
	cd $(BACKEND_DIR) && go test ./...

swagger:
	cd $(BACKEND_DIR) && go run github.com/swaggo/swag/cmd/swag@latest init \
		-g cmd/worker/main.go \
		-o internal/api/docs \
		--parseDependency \
		--parseInternal

tidy:
	cd $(BACKEND_DIR) && go mod tidy

cli:
	cd $(BACKEND_DIR) && go run ./cmd/cli doctor --suite TR0

doctor:
	cd $(BACKEND_DIR) && go run ./cmd/cli doctor --suite TR0 --format md

web-build:
	cd frontend && npm install && npm run build

web-dev:
	cd frontend && npm install && npm run dev

web:
	@echo "Build: make web-build  |  Dev: make web-dev"
	@echo "Console: http://localhost:8080/ui/ (serve frontend/dist via worker)"

reorganize:
	powershell -ExecutionPolicy Bypass -File scripts/reorganize.ps1
