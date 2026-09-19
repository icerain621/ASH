# v4.1 企业 Agentic 设计 / Enterprise Agentic design（B · DX67–DX72）

> 状态 / Status: **范围稿已批准** / **approved for scope write-up**（2026-09-19 · approach A）  
> 程序 / Program: [`doc/plan/v4.x-program.md`](../../../doc/plan/v4.x-program.md) · 范围 / Scope: [`doc/plan/v4.1-release-scope.md`](../../../doc/plan/v4.1-release-scope.md)

## 意图 / Intent

在已冻结的 v4.0 Auth 之后，交付企业 Agentic **可配置、可看见、fail-closed 占位**：  
After frozen v4.0 Auth, deliver **enterprise Agentic visibility and fail-closed placeholders**:

1. Org/space **配额 / quotas**（并发 Run · token 代理预算）— 配置 + Create/Spawn 门禁  
2. **审计薄报表 / thin audit report** — 聚合现有 audit（无新表 / no new tables）  
3. **spawn 策略与预算 / spawn policy & budget** — SpacePolicy 覆盖 Harness；投影可见  

标签目标 / Tag target: **`v4.1.0`**（人工 / manual；F6）。

## 方案 / Approach（A）

六板对齐 v4.0 节奏。优先扩展 `SpacePolicyPack.bodyJson` + 只读 API，避免新表（F4）。  
Six sprints mirroring v4.0. Prefer `bodyJson` + read APIs over new tables (F4).

## 非目标 / Non-goals

- 真计费 / Real billing  
- 跨 org 大盘 / Cross-org rollups  
- SAML / 强制默认 OIDC  
- Plugin gRPC / 外置 Indexer → **v4.2**  
- Marketplace / 多区域 → **v4.3**  
- 收紧配额时强杀存量 Run / Force-kill running runs  
- 语音 / IM 网关  

## 数据与合并 / Data & merge

| Key | 位置 / Where | 说明 / Notes |
|-----|--------|--------|
| `quotas.maxConcurrentRuns` | `bodyJson.quotas` | `0` / 省略 = 不限 / unlimited |
| `quotas.tokenBudgetProxy` | `bodyJson.quotas` | 代理预算占位；DX67 解析，用量投影见 DX68 |
| `subRun.maxDepth` | `bodyJson.subRun` | 覆盖 Harness（DX71） |
| `subRun.allowedTools` | `bodyJson.subRun` | 子 Run 白名单仍 fail-closed |
| `subRun.tokenBudgetProxy` | `bodyJson.subRun` | 子 Run 可见预算 |

合并：platform/org-template → space kind → pack → ResourceScope；**更严优先**。未配置配额 = 不限。  
Merge: defaults → pack → ResourceScope; **stricter wins**. Unconfigured = unlimited.

门禁：Create/Spawn 时统计 space 内 `running`/`waiting_approval`；超限拒绝，稳定错误码 `SPACE_QUOTA_EXCEEDED`。  
Enforcement: count active runs at Create/Spawn; reject with `SPACE_QUOTA_EXCEEDED`.

## API 草图 / APIs (sketch)

| Method | Path | Sprint |
|--------|------|--------|
| PUT/GET policy | `…/spaces/{id}/policy` | DX67 body |
| GET | `…/spaces/{id}/quotas` | DX68 |
| GET | `…/spaces/{id}/audit-report?window=` | DX69 |
| POST sub-runs | enrich | DX71 |

## Sprint 图 / Sprint map

| ID | 主题 / Theme |
|----|--------|
| DX67 | quotas schema + Create/Spawn gate |
| DX68 | quotas read API + console |
| DX69 | audit-report API |
| DX70 | audit report UI |
| DX71 | bodyJson.subRun + visibility |
| DX72 | freeze + `make v4.1-signoff` |

## 验证 / Verification

```bash
make openapi-check
go test ./internal/spacepolicy/ ./internal/runs/ ./internal/api/ -count=1
make v4.1-signoff   # after DX72
```
