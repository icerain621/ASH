# Changelog

All notable changes to ASH will be documented in this file.

This project follows a Keep a Changelog style. Version numbers can be attached when a release tag is cut.

## [Unreleased] - 2026-06-03

### Added

- Added frontend scenario listing API and Runs page scenario picker backed by `/api/v1/scenarios`.
- Added Doctor console suite selector for TR0, TR1, TR2, and ALL.
- Added TR0-08 doctor case to verify `feature_delivery`, `hotfix`, and `security_patch` scenarios are loaded.
- Enabled dev-mode default plugin registry gRPC on `127.0.0.1:19091` when `ASH_PLUGIN_GRPC_ADDR` is unset.
- Added M1 memory governance hints on candidate creation (`governance.duplicates` / `governance.conflicts`).
- Added M1 self-iteration API (`/api/v1/improve/proposals`) with experiment replay compare, canary, promote, and rollback.
- Added Doctor TR1-05 Rules DSL schema validation case.
- Expanded Memory and Automation console panels for governance edges, memory query, and improve proposals.
- Added TR2 compliance console at `/ui/compliance` with live readiness cards and structured doctor reports.
- Added `GET /api/v1/spaces/:spaceId/resource-scopes` for resource scope visibility.
- Reworked Doctor page to render pass/fail case tables via `DoctorReportView`.
- Added secret leak scanning (`internal/security/leakscan`) with compliance API and TR2-05 doctor case.
- Added compliance audit export bundling doctor reports with audit logs.
- Applied audit log payload redaction in list API when `redactPayload` policy is enabled.
- Fixed doctor `ALL` suite to run the full TR0/TR1/TR2 case set.
- Added Doctor TR3 suite (TR3-01 memory migration, TR3-02 RAG FTS fallback, TR3-03 cost/latency SLO metrics, TR3-04 audit provenance).
- Added scale readiness API and `/ui/scale` console for GA TR3 visibility.
- Added run provenance API (`GET /api/v1/runs/:runId/provenance`).
- Embedded secret-scan summary in compliance export bundles; compliance page one-click audit redact toggle.
- Extended doctor `ALL` to include TR3 cases.
- Added Runs console provenance panel (TR3-04) wired to `GET /api/v1/runs/:runId/provenance`.
- Added API tests for scale readiness and run provenance; Swagger annotations for compliance/scale/provenance.
- Added cross-platform `scripts/verify-local.sh` for Git Bash and Linux.
- Added Swagger godoc for improve proposals API and `scripts/regenerate-swagger.sh`.
- Added `TestALLSuite` doctor regression (22 cases) and compliance export integration test.
- Compliance console export supports TR2/TR3/ALL doctor suite selection.
- Added M2 permission matrix (`internal/authz`): RBAC catalog, scenario×role tool policies, run-time enforcement, API/UI, and Doctor `M2-01`.
- Added `PUT /api/v1/spaces/:spaceId/resource-scopes/:scopeId` for scenario tool policy updates; Doctor `M2-02`; M2 cases in `ALL` suite (24 cases).
- Space console scenario policy editor; compliance console M2 readiness cards.
- Audit log on scenario policy updates (`scope.policy_updated`); Doctor M2-03 runtime `POLICY_DENIED` enforcement.
- M3 tenant isolation helpers (`store.EnforceSpaceAccess`), Postgres profile/readiness, Doctor M3 suite, and migration guide `doc/05-M3-多租户与Postgres演进.md`.
- Scale readiness exposes `databaseDialect`, `postgresConfigured`, `migrationReady`; `scripts/postgres-smoke.sh`.
- Postgres migration CLI: `ash migrate plan|copy|verify|sync` and `ash migrate dual-write enable|disable|status|sync`; runtime mirror via `ASH_DUAL_WRITE_POSTGRES_URL`; `scripts/migrate-postgres.sh`.
- Doctor M3-03 migration catalog check; scale readiness exposes migration table count, dual-write status, and last sync time; Scale/Compliance consoles updated.
- API tenant enforcement helpers (`requireRequestSpace` / `requireTargetSpace`) across space-param routes and cross-space writes; Worker auto-loads dual-write from `.ash/migration/dual-write.json`; `doc/TODO.md` tracks Postgres e2e migrate validation.
- Docker Postgres dev stack (`docker-compose.postgres.yml`); `scripts/postgres-e2e-migrate.sh` and `make postgres-e2e`; Doctor M3-04 live migrate verify (`ASH_MIGRATE_E2E=1`); integration test tag `integration`.
- Runs API returns `201` with `executionError` when run is created but policy/execution fails; `actorRole` on run summary.
- Cross-space run/memory access returns `403`; Runs console actor-role picker; Scale page database/M3 section.
- Switched default SQLite driver to pure-Go `github.com/glebarez/sqlite` (no CGO on Windows).
- Fixed JSON secret detection/redaction in `leakscan`; dev auth honors `X-ASH-Space-ID`; test DB auto-close via `store.OpenTest`.
- Added an ExecGo/Codex agent execution boundary under `internal/agentexec`, including executor contracts, a Codex-backed ExecGo adapter, cancellation/status hooks, and a deterministic static executor for tests and smoke runs.
- Added `agent` step support to the Rules DSL, with agent adapter metadata, capabilities, prompts, timeout configuration, retry metadata, and approval metadata on gates.
- Added a QA verification step to `scenarios/feature_delivery.yaml`, separating Codex implementation from test execution evidence.
- Added lightweight repository RAG indexing and query support under `internal/rag`, with line-range citations, content digests, space scoping, and simple term scoring.
- Added persistent run observability models for run steps, tool calls, agent tasks, artifact indexes, checkpoints, audit logs, RAG documents/chunks, usage metrics, quality metrics, feedback, organization/space membership, roles, scopes, audit exports, and plugin registry records.
- Added run timeline, tool call, agent task, cancel, approval, and RAG service surfaces in the run service layer.
- Added richer run artifact bundle metadata, including repo root, issue/spec, event range, agent task id, evidence references, generated release notes, rollback plan, diff capture, and default test report generation.
- Added SPA fallback coverage for `/ui/` routes so direct navigation such as `/ui/runs` serves the frontend shell while still serving static assets.
- Added frontend dependencies for TanStack Router, TanStack Table, and lucide-react icons.
- Added Vite type declarations and lockfiles for reproducible frontend and Go dependency installs.

### Changed

- Switched the frontend router from `react-router-dom` to TanStack Router with `/ui` as the basepath.
- Reworked the ASH Console layout with icon navigation, page headings, denser panes, status pills, empty states, and table-based Runs rendering.
- Expanded the Runs page to use TanStack Table and clearer controls for refresh, new run, resume, and replay actions.
- Refined Memory and Doctor pages with structured headers, action icons, report panes, candidate counts, and explicit empty states.
- Expanded run execution to persist step state, checkpoints, tool call results, agent execution records, audit entries, RAG/memory evidence, and artifact indexes during scenario execution.
- Changed artifact generation from M0 stubs toward evidence-backed release notes, rollback plans, git diff capture, test report preservation, and manifest producer metadata.
- Updated `feature_delivery` from a direct tool-chain implementation step to an agent implementation step followed by QA test verification.
- Updated frontend documentation to reflect the new Vite + React + TypeScript stack, TanStack Router/Query/Table usage, lucide icons, and `/ui/` dev route.

### Fixed

- Fixed `runs.Service.eventsFor` infinite recursion when bound to a request context (approval/cancel paths stack overflow).
- Fixed malformed return statements in the SQLite store initialization path.
- Fixed backend static UI routing so nested SPA routes fall back to `index.html` without allowing path traversal outside the web directory.
- Fixed event spelling for memory review request emission.
- Added regression coverage for static UI SPA fallback behavior.
- Updated run control and doctor tests to account for the expanded execution model and static agent executor.

### Notes

- The working tree also contains IDE project metadata under `.idea/`. Keep those files staged only if the repository intentionally tracks editor configuration.
- This entry describes the current unreleased workspace changes. Replace `[Unreleased]` with a tagged version when cutting a release.
