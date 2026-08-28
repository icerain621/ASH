# Sprint DH：Harness 骨架 Implementation Plan

> **For agentic workers:** 按任务勾选推进；对应设计 [`../design/HLD-Harness与沙盒.md`](../design/HLD-Harness与沙盒.md)。  
> **Goal:** Space 可 CRUD draft Harness Profile，`LoadActive` 可用，OpenAPI + 迁移 rev 21 + RLS + 烟测绿。  
> **Architecture:** `internal/harness` 服务 + `harness_profile_versions` 表（space_id 租户 RLS）+ Gin API。  
> **Tech Stack:** Go / GORM / Gin / golang-migrate / JSON Schema  
> **状态：** DH-1~DH-5 ✅；DH-6 探针以包测落地（Doctor 挂载可后续）

## Global Constraints

- SQL expectedVersion → **21**；RLS policy expected → **42**（+1）
- 新表必须进 `MigrationCatalog` + `PostgresRLSTables` + `VerifyRLSMigrationSQL`
- 公开 API：handler 注释 → swagger → openapi-check
- Commit 说明用中文；仅用户要求时 push

## 任务板（v2 开发）

| ID | Sprint | 任务 | 状态 |
|----|--------|------|------|
| DH-1 | DH | JSON Schema `ash.harness.profile.v1.json` | ✅ |
| DH-2 | DH | store 模型 + 迁移 000021 + RLS catalog | ✅ |
| DH-3 | DH | `internal/harness` Service（CRUD/LoadActive/promote 骨架） | ✅ |
| DH-4 | DH | API `/harness/profiles*` + 路由 + OpenAPI | ✅ |
| DH-5 | DH | 单测 + `make harness-smoke` | ✅ |
| DH-6 | DH | Schema/唯一性探针（包测；Doctor 挂载后续） | ✅ 包测 |
| DI-1 | DI | Loop Adapter 事件钩子 | 后续 |
| DX-1 | DX | `internal/sandbox` Docker POC | 后续 |
| DY-1 | DY | Feedback targetType 扩展 | 后续 |

---

### Task DH-1: Schema

- [x] `doc/appendices/schemas/ash.harness.profile.v1.json`
- [x] 校验 `provider.kind` / `sandbox.defaultMode` enum

### Task DH-2: Store + Migration

- [x] `HarnessProfileVersion` model
- [x] `000021_harness_profiles.up.sql` / `.down.sql`
- [x] `expectedVersion=21`；catalog + RLS；`VerifyRLSMigrationSQL` 含 000021

### Task DH-3: Service

- [x] Create / List / Get / Update (draft only) / SubmitReview / Promote / LoadActive
- [x] Default platform profile 常量（无 DB 行时回退）

### Task DH-4: API

- [x] Handlers + Register routes
- [x] Space 隔离；swagger 注释 + `doc/api/openapi-ash-v1.yaml`

### Task DH-5: Smoke

- [x] `scripts/harness-smoke.sh` + Makefile
- [x] `go test ./internal/harness/...` + API 单测

### Task DH-6: Doctor

- [x] Schema 校验 + active 唯一性：`internal/harness/service_test.go`（Doctor M4-HAR-* 挂载留给后续 Sprint）
