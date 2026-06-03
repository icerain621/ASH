# Claude Code Guide for ASH

This file gives Claude Code the repository context and operating rules for working on ASH.

## Repository Summary

ASH is an AI assistant orchestration project. The current implementation is a Go backend Worker/CLI plus a Vite React TypeScript console.

Primary stack:

- Go 1.26 module: `github.com/ash-repwiki/ash`
- Backend libraries: Gin, GORM, SQLite, Prometheus client, swaggo
- Frontend: Vite, React 18, TypeScript, TanStack Router, TanStack Query, TanStack Table, Zustand, lucide-react
- Local database: SQLite, commonly under `.ash/ash.db`

## Important Paths

- `cmd/worker/main.go` - Worker HTTP service entrypoint.
- `cmd/cli/main.go` - CLI entrypoint for diagnostics such as TR0 doctor runs.
- `internal/api/` - HTTP handlers, routes, Swagger models, static UI serving, run/memory/RAG APIs.
- `internal/runs/` - run execution, resume/replay/control, metadata, event emission integration.
- `internal/events/` - event stream service.
- `internal/memory/` - memory candidate lifecycle, review, query, hit-used audit.
- `internal/rag/` - retrieval/query service.
- `internal/rules/` - scenario DSL loading, parsing, validation, engine types.
- `internal/store/` - GORM models and SQLite setup.
- `internal/toolbus/` - built-in tool execution and Git tool adapter.
- `internal/artifacts/` - artifact bundling and manifest logic.
- `frontend/src/` - React console source.
- `scenarios/` - YAML scenarios such as `feature_delivery.yaml`, `hotfix.yaml`, and `security_patch.yaml`.
- `doc/` - product, architecture, implementation, API, database, and milestone documentation.

## Commands

Run these from the repository root unless noted.

```bash
make tidy
make test
make run
make doctor
make swagger
make web-build
make web-dev
```

Backend equivalents:

```bash
go mod tidy
go test ./...
go run ./cmd/worker
go run ./cmd/cli doctor --suite TR0 --format md
```

Frontend equivalents:

```bash
cd frontend
npm install
npm run dev
npm run build
```

Useful local URLs:

- Worker: `http://localhost:8080`
- Swagger UI: `http://localhost:8080/docs`
- Console served by worker: `http://localhost:8080/ui/`
- Vite dev console: `http://127.0.0.1:5173/ui/`

## Development Rules

- Prefer existing package boundaries and Makefile targets.
- Do not overwrite or revert user changes. Always inspect `git status --short` before making broad edits.
- Keep backend HTTP concerns in `internal/api/`; keep business behavior in the relevant service package.
- Keep persistent model changes in `internal/store/` and check tests plus API response models.
- When changing run lifecycle behavior, inspect `internal/runs/`, `internal/events/`, and any affected API handlers together.
- When changing memory behavior, inspect `internal/memory/`, `internal/api/memory.go`, and any SSE emission paths.
- When changing scenario validation or YAML shape, inspect `internal/rules/` and the files in `scenarios/`.
- When changing frontend API calls, check `frontend/src/modules/*/api/` and related pages.
- When adding or modifying public APIs, update Swagger annotations/models and any matching frontend client code.
- Keep comments sparse and only add them when they clarify non-obvious behavior.

## Verification Matrix

Choose the smallest useful verification set:

- Backend logic: `make test`
- Doctor/TR0 behavior: `make doctor`
- Swagger or route changes: `make swagger`
- Frontend UI/client changes: `make web-build`
- Full local smoke test: run `make run`, then open `/docs` or `/ui/`

For documentation-only changes, tests are optional; mention that no tests were run because only docs changed.

## Git Guidance

- Stage only files that belong to the current task.
- Split commits by feature or concern.
- Use Chinese commit messages encoded as UTF-8.
- After code or documentation edits, ask the user whether they want a commit.

## Response Style

- Be direct and practical.
- When uncertain about product or architecture intent, ask for the boundary instead of guessing too far.
- Report changed files and verification results clearly.

