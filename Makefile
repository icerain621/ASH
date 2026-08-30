# Detect backend location (pre/post reorganize). Prefer the root module when
# it contains the actual worker entrypoint; fall back to backend/ for older
# layouts.
BACKEND_DIR := .
ifeq ($(wildcard cmd/worker/main.go),)
ifneq ($(wildcard backend/go.mod),)
BACKEND_DIR := backend
endif
endif

.PHONY: run test swagger openapi-check proto-lint proto-generate proto-check tidy doctor cli migrate-plan migrate-schema postgres-up postgres-down postgres-roles postgres-e2e postgres-sql-schema-e2e postgres-rls-e2e postgres-rds-e2e postgres-local-rds-e2e postgres-app-gate test-integration test-rls execgo-bootstrap execgo-health execgo-live-smoke secret-rotate-smoke plugin-sign-smoke harness-smoke sandbox-smoke release-sampling release-sampling-static release-sampling-smoke live-smoke smoke-static web web-build web-dev web-lint web-test web-gate sse-browser-e2e r08-cross-space-gate verify regression-short cloud-acceptance mvp-signoff v2-signoff production-config-gate rollback-drill queue-gate t0-alert-gate data-backup data-backup-verify data-backup-smoke worker-local-gate worker-production-gate pre-migrate-gate t1-metrics-gate scope-freeze-gate config-env-gate release-window-prefill release-window-gate bootstrap-local-ash-db local-readiness-gate signoff-apply signoff-gate kpi-reconcile-gate evidence-sha-gate

run:
	cd $(BACKEND_DIR) && go run ./cmd/worker

test:
	cd $(BACKEND_DIR) && go test ./...

swagger:
	bash scripts/regenerate-swagger.sh

openapi-check:
	bash scripts/openapi-check.sh

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

execgo-live-smoke:
	bash scripts/execgo-live-smoke.sh

secret-rotate-smoke:
	bash scripts/secret-rotate-smoke.sh

plugin-sign-smoke:
	bash scripts/plugin-sign-smoke.sh

harness-smoke:
	bash scripts/harness-smoke.sh

sandbox-smoke:
	bash scripts/sandbox-smoke.sh

release-sampling:
	bash scripts/release-sampling.sh

release-sampling-static:
	bash scripts/release-sampling-static.sh

release-sampling-smoke:
	bash scripts/release-sampling-smoke.sh

live-smoke:
	bash scripts/live-smoke.sh

smoke-static:
	bash scripts/smoke-static.sh

doctor:
	cd $(BACKEND_DIR) && go run ./cmd/cli doctor --suite TR0 --format md

migrate-plan:
	cd $(BACKEND_DIR) && go run ./cmd/cli migrate plan --postgres "$${ASH_DATABASE_URL}"

migrate-schema:
	cd $(BACKEND_DIR) && go run ./cmd/cli migrate schema up --postgres "$${ASH_DATABASE_URL}"

postgres-up:
	bash scripts/postgres-up.sh

postgres-down:
	bash scripts/postgres-down.sh

postgres-roles:
	bash scripts/postgres-ensure-app-role.sh

postgres-e2e:
	bash scripts/postgres-e2e-migrate.sh

postgres-sql-schema-e2e:
	bash scripts/postgres-sql-schema-e2e.sh

test-integration:
	cd $(BACKEND_DIR) && ASH_MIGRATE_E2E=1 go test -tags=integration ./internal/api/ -run TestPostgresReadyzProbe -count=1
	cd $(BACKEND_DIR) && ASH_MIGRATE_E2E=1 go test -tags=integration ./internal/store/ -run TestMigratorSQLiteToPostgresE2E -count=1

test-rls:
	cd $(BACKEND_DIR) && ASH_POSTGRES_RLS=1 go test -tags=integration ./internal/store/ -run TestPostgresRLS -count=1

postgres-rls-e2e:
	bash scripts/postgres-rls-e2e.sh

postgres-app-gate:
	bash scripts/postgres-app-gate.sh

postgres-rds-e2e:
	bash scripts/postgres-rds-e2e.sh

# Docker local stand-in for cloud RDS e2e (needs make postgres-up + .ash/ash.db).
postgres-local-rds-e2e:
	bash scripts/postgres-local-rds-e2e.sh

ci-fixture-smoke:
	bash scripts/ci-fixture-smoke.sh

release-window-audit:
	bash scripts/release-window-audit.sh

verify:
	bash scripts/verify-local.sh

regression-short:
	bash scripts/regression-short.sh

cloud-acceptance:
	bash scripts/cloud-acceptance-gate.sh

mvp-signoff:
	bash scripts/mvp-signoff-gate.sh

v2-signoff:
	bash scripts/v2-signoff-gate.sh

production-config-gate:
	bash scripts/production-config-gate.sh

rollback-drill:
	bash scripts/rollback-drill-gate.sh

queue-gate:
	bash scripts/queue-gate.sh

t0-alert-gate:
	bash scripts/t0-alert-gate.sh

data-backup:
	bash scripts/ash-data-backup.sh

data-backup-verify:
	bash scripts/ash-data-backup-verify.sh

data-backup-smoke:
	bash scripts/ash-data-backup-smoke.sh

worker-local-gate:
	bash scripts/worker-local-gate.sh

pre-migrate-gate:
	bash scripts/pre-migrate-gate.sh

t1-metrics-gate:
	bash scripts/t1-metrics-gate.sh

kpi-reconcile-gate:
	bash scripts/kpi-reconcile-gate.sh

scope-freeze-gate:
	bash scripts/scope-freeze-gate.sh

config-env-gate:
	bash scripts/config-env-gate.sh

release-window-prefill:
	bash scripts/release-window-prefill.sh

release-window-gate:
	bash scripts/release-window-gate.sh

bootstrap-local-ash-db:
	bash scripts/bootstrap-local-ash-db.sh

local-readiness-gate:
	bash scripts/local-readiness-gate.sh

signoff-apply:
	bash scripts/signoff-apply.sh

signoff-gate:
	bash scripts/signoff-gate.sh

evidence-sha-gate:
	bash scripts/evidence-sha-gate.sh


worker-production-gate:
	bash scripts/worker-production-gate.sh

web-build:
	cd frontend && npm install && npm run build

web-lint:
	cd frontend && npm install && npm run lint

web-test:
	cd frontend && npm install && npm run test:run

web-gate:
	bash scripts/web-gate.sh

# Optional P2-4 browser SSE smoke (Playwright + ephemeral Worker). Not part of web-gate.
sse-browser-e2e:
	bash scripts/sse-browser-e2e.sh

# R-08 cross-space / RLS sampling (API always; Postgres live when Docker/URL available).
r08-cross-space-gate:
	bash scripts/r08-cross-space-gate.sh

web-dev:
	cd frontend && npm install && npm run dev

web:
	@echo "Build: make web-build  |  Dev: make web-dev"
	@echo "Console: http://localhost:8080/ui/ (serve frontend/dist via worker)"

reorganize:
	powershell -ExecutionPolicy Bypass -File scripts/reorganize.ps1
