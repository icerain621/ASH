# ASH GA / 上线推进计划（薄入口）

> 更新：2026-10-11 · 归属：[`plan/`](README.md)  
> 详细进度叙事见 [`PLAN-进度与里程碑.md`](PLAN-进度与里程碑.md)。  
> Updated: 2026-10-11. Narrative lives in PLAN.

## 目标 / Goal

在 **不新开功能代** 的前提下，把已冻结水位推到可人工打 tag、可云验收、可真人签字。  
Ship frozen watermarks to manual tags, cloud acceptance, and human sign-off — **no new feature generation**.

## 六步（G0–G5） / Six steps

| 步 | 内容 | 入口 | 状态 |
|----|------|------|------|
| G0 | 本地门禁绿 | `make local-readiness-gate` / `regression-short` + `web-gate` | ✅ 可重复跑 |
| G1 | 范围冻结确认 | 各 `*-release-scope.md` + `make *-signoff` | ✅ v2–v6.7 已冻；**v7** 用 `make v7-signoff`（P0–P13 测试绿） |
| G2 | 云 RDS 验收 | `config/cloud-rds.env` → `make cloud-acceptance` | ⏸ 待真实 RDS |
| G3 | 切换日 runbook | `doc/checklists/` 发布窗口 + rollback drill | ⏸ 生产日 |
| G4 | 四人真人签字 | MVP：`signoff.env` + `make signoff-gate`；**v7**：`make v7-signoff-prep` → `make v7-signoff-apply`（同 `signoff.env` 四人字段） | ⚠️ MVP dry-run 已过；v7 待填 `signoff.env` |
| G5 | 人工 tag | 例：`v6.7.0` / `v7.0.0`（须对应 signoff 绿；脚本不自动 tag） | ⏸ 可选 |

## 当前建议顺序 / Suggested order

1. 保持 `main` 门禁绿（含 v7：`make v7-signoff` / `openapi-check`）。  
2. v7 签字准备：`make v7-signoff-prep` → 四人填 [`../evidence/v7-signatures-latest.md`](../evidence/v7-signatures-template.md)。  
3. 准备 `cloud-rds.env` 跑 G2。  
4. 需要对外版本号时再打 **单个** 代际 tag（`v6.7.0` 或 `v7.0.0`），勿批量虚构中间号。

## 不做 / Out

- 为上线而新开 v6.8 功能轨。  
- 脚本自动 `git tag`（F6）。
