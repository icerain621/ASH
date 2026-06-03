# Changelog

All notable changes to ASH will be documented in this file.

This project follows a Keep a Changelog style. Version numbers can be attached when a release tag is cut.

## [Unreleased] - 2026-06-03

### Added

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

- Fixed malformed return statements in the SQLite store initialization path.
- Fixed backend static UI routing so nested SPA routes fall back to `index.html` without allowing path traversal outside the web directory.
- Fixed event spelling for memory review request emission.
- Added regression coverage for static UI SPA fallback behavior.
- Updated run control and doctor tests to account for the expanded execution model and static agent executor.

### Notes

- The working tree also contains IDE project metadata under `.idea/`. Keep those files staged only if the repository intentionally tracks editor configuration.
- This entry describes the current unreleased workspace changes. Replace `[Unreleased]` with a tagged version when cutting a release.
