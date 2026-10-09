# ASH GA / 上线推进计划（薄入口）

> 更新：2026-10-08 · 归属：[`plan/`](README.md)  
> 详细进度叙事见 [`PLAN-进度与里程碑.md`](PLAN-进度与里程碑.md)。  
> Updated: 2026-10-08. Narrative lives in PLAN.

## 目标 / Goal

在 **不新开功能代** 的前提下，把已冻结水位推到可人工打 tag、可云验收、可真人签字。  
Ship frozen watermarks to manual tags, cloud acceptance, and human sign-off — **no new feature generation**.

## 六步（G0–G5） / Six steps

| 步 | 内容 | 入口 | 状态 |
|----|------|------|------|
| G0 | 本地门禁绿 | `make local-readiness-gate` / `regression-short` + `web-gate` | ✅ 可重复跑 |
| G1 | 范围冻结确认 | 各 `*-release-scope.md` + `make *-signoff` | ✅ v2–v6.7 已冻 |
| G2 | 云 RDS 验收 | `config/cloud-rds.env` → `make cloud-acceptance` | ⏸ 待真实 RDS |
| G3 | 切换日 runbook | `doc/checklists/` 发布窗口 + rollback drill | ⏸ 生产日 |
| G4 | 四人真人签字 | `signoff.env` + `make signoff-gate` | ⚠️ dry-run 已过 |
| G5 | 人工 tag | 例：`v6.7.0`（须对应 signoff 绿） | ⏸ 可选 |

## 当前建议顺序 / Suggested order

1. 保持 `main` 门禁绿（含冻结后 P02 / 推理等级变更后的 `openapi-check`）。  
2. 准备 `cloud-rds.env` 跑 G2。  
3. 需要对外版本号时再打 **单个** 代际 tag（优先最近冻结的 `v6.7.0`），勿批量虚构中间号。

## 不做 / Out

- 为上线而新开 v6.8 功能轨。  
- 脚本自动 `git tag`（F6）。
