# Detect backend location (pre/post reorganize). Prefer the root module when
# it contains the actual worker entrypoint; fall back to backend/ for older
# layouts.
BACKEND_DIR := .
ifeq ($(wildcard cmd/worker/main.go),)
ifneq ($(wildcard backend/go.mod),)
BACKEND_DIR := backend
endif
endif

.PHONY: run test swagger openapi-check proto-lint proto-generate proto-check tidy doctor cli migrate-plan migrate-schema postgres-up postgres-down postgres-roles postgres-e2e postgres-sql-schema-e2e postgres-rls-e2e postgres-rds-e2e test-integration test-rls execgo-bootstrap execgo-health execgo-live-smoke web web-build web-dev verify regression-short

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

postgres-rds-e2e:
	bash scripts/postgres-rds-e2e.sh

ci-fixture-smoke:
	bash scripts/ci-fixture-smoke.sh

verify:
	bash scripts/verify-local.sh

regression-short:
	cd $(BACKEND_DIR) && go test ./internal/doctor/... -run 'TestM3Suite|TestTR3Suite|TestTR3PrometheusReplaySegmentWhenEnabled' -count=1
	cd $(BACKEND_DIR) && go test ./internal/alerts/... -count=1
	cd $(BACKEND_DIR) && go test ./internal/api/... -run 'TestHealthzAndReadyzSQLite|TestReadyzOpsSnapshot|TestReadyzIncludesRLSCatalogWhenEnabled|TestReadyzLiveGateHints|TestCISyncRunsWithFixture|TestReleaseSampling' -count=1
	cd $(BACKEND_DIR) && go test ./internal/memory/... -run 'TestRunMigrations|TestDefaultTTLForLayer|TestEffectiveTTL|TestTTLQueue|TestClassifyTTL' -count=1
	cd $(BACKEND_DIR) && go test ./internal/ci/... -run 'TestFixtureProvider|TestDiagnoseLogClassifiesTestFailure|TestServiceSyncJobsDiagnose' -count=1
	cd $(BACKEND_DIR) && go test ./internal/opsenv/... -count=1
	cd $(BACKEND_DIR) && go test ./internal/memory/... -count=1 -short
	cd $(BACKEND_DIR) && go test ./internal/openapicheck -run 'TestContractMatchesSwagger|TestApiV1SuccessResponsesAvoidGenericEnvelope|TestValidateContract|TestValidateReadyzContract' -count=1
	cd $(BACKEND_DIR) && go test ./internal/store -run 'TestMigrationCatalog_RLSCoverage|TestVerifyRLSMigrationSQL|TestRLSExpectedPolicyCount' -count=1

web-build:
	cd frontend && npm install && npm run build

web-dev:
	cd frontend && npm install && npm run dev

web:
	@echo "Build: make web-build  |  Dev: make web-dev"
	@echo "Console: http://localhost:8080/ui/ (serve frontend/dist via worker)"

reorganize:
	powershell -ExecutionPolicy Bypass -File scripts/reorganize.ps1
