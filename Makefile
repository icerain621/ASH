# Detect backend location (pre/post reorganize). Prefer the root module when
# it contains the actual worker entrypoint; fall back to backend/ for older
# layouts.
BACKEND_DIR := .
ifeq ($(wildcard cmd/worker/main.go),)
ifneq ($(wildcard backend/go.mod),)
BACKEND_DIR := backend
endif
endif

.PHONY: run test swagger openapi-check proto-lint proto-generate proto-check tidy doctor cli execgo-bootstrap execgo-health web web-build web-dev

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

openapi-check:
	cd $(BACKEND_DIR) && tmp=$$(mktemp -d) && \
		trap 'rm -rf "$$tmp"' EXIT && \
		cp -R internal/api/docs "$$tmp/docs" && \
		go run github.com/swaggo/swag/cmd/swag@latest init \
			-g cmd/worker/main.go \
			-o internal/api/docs \
			--parseDependency \
			--parseInternal >/dev/null && \
		diff -ru "$$tmp/docs" internal/api/docs

proto-lint:
	cd $(BACKEND_DIR) && go run github.com/bufbuild/buf/cmd/buf@latest lint proto

proto-generate:
	cd $(BACKEND_DIR) && go run github.com/bufbuild/buf/cmd/buf@latest generate proto --template proto/buf.gen.yaml

proto-check: proto-lint
	cd $(BACKEND_DIR) && tmp=$$(mktemp -d) && \
		trap 'rm -rf "$$tmp"' EXIT && \
		template=$$(pwd)/proto/buf.gen.yaml && \
		cp -R proto "$$tmp/proto" && \
		rm -f "$$tmp"/proto/ash/v1/*.pb.go && \
		(cd "$$tmp" && go run github.com/bufbuild/buf/cmd/buf@latest generate proto --template "$$template") && \
		diff -ru proto/ash/v1 "$$tmp/proto/ash/v1"

tidy:
	cd $(BACKEND_DIR) && go mod tidy

cli:
	cd $(BACKEND_DIR) && go run ./cmd/cli doctor --suite TR0

execgo-bootstrap:
	bash scripts/bootstrap-execgo.sh

execgo-health:
	bash scripts/execgo-health.sh

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
