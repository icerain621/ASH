# Detect backend location (pre/post reorganize). Prefer the root module when
# it contains the actual worker entrypoint; fall back to backend/ for older
# layouts.
BACKEND_DIR := .
ifeq ($(wildcard cmd/worker/main.go),)
ifneq ($(wildcard backend/go.mod),)
BACKEND_DIR := backend
endif
endif

.PHONY: run test swagger openapi-check proto-lint proto-generate proto-check tidy doctor cli migrate-plan postgres-up postgres-down postgres-e2e test-integration execgo-bootstrap execgo-health web web-build web-dev verify

run:
	cd $(BACKEND_DIR) && go run ./cmd/worker

test:
	cd $(BACKEND_DIR) && go test ./...

swagger:
	bash scripts/regenerate-swagger.sh

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

migrate-plan:
	cd $(BACKEND_DIR) && go run ./cmd/cli migrate plan --postgres "$${ASH_DATABASE_URL}"

postgres-up:
	bash scripts/postgres-up.sh

postgres-down:
	bash scripts/postgres-down.sh

postgres-e2e:
	bash scripts/postgres-e2e-migrate.sh

test-integration:
	cd $(BACKEND_DIR) && ASH_MIGRATE_E2E=1 go test -tags=integration ./internal/store/ -run TestMigratorSQLiteToPostgresE2E -count=1

verify:
	bash scripts/verify-local.sh

web-build:
	cd frontend && npm install && npm run build

web-dev:
	cd frontend && npm install && npm run dev

web:
	@echo "Build: make web-build  |  Dev: make web-dev"
	@echo "Console: http://localhost:8080/ui/ (serve frontend/dist via worker)"

reorganize:
	powershell -ExecutionPolicy Bypass -File scripts/reorganize.ps1
