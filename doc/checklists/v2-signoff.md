# v2.0 发布签字清单（Sprint DV）

> 目标：冻结 [`v2-release-scope.md`](../plan/v2-release-scope.md)，跑通 `make v2-signoff`，**人工确认后再打 `v2.0.0` tag**。

## 前置

| # | 项 | 命令 / 文件 | 期望 |
|---|-----|-------------|------|
| 1 | 范围文档 | `doc/plan/v2-release-scope.md` | 含 §1–§6；状态「已冻结」 |
| 2 | 结构门禁 | `make scope-freeze-gate` | MVP + v2 均 OK |
| 3 | v2 门禁 | `make v2-signoff` | Doctor ALL / openapi / KPI / session 绿 |
| 4 | （可选）全量 | `make mvp-signoff` | 除云 RDS 外自动化 ✅ |

## 签字会议（建议 20–30min）

1. 过一遍 v2 scope §2 In / §3 Out  
2. 确认 Out 项（云 RDS、H-06 live）不阻塞功能 tag  
3. 填写 [`../evidence/v2-signatures-template.md`](../evidence/v2-signatures-template.md) → 复制为 `v2-signatures-latest.md`  
4. 产品确认范围冻结

## Tag（人工）

```bash
git status -sb
git tag -a v2.0.0 -m "ASH v2.0.0 dual-core GA (DH–DU frozen)"
git push origin v2.0.0
```

本清单与 `make v2-signoff` **不会**自动执行 `git tag`。

## 相关

- 计划：`doc/plan/sprint-dv-v2-signoff.md`
- MVP 对照：`mvp-signoff-roster.md` / `make signoff-gate`
