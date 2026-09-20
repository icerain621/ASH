# Sprint VX35 — Backend 枚举投影 / Backend enum projection

> **前置 / Prerequisite：** VX34  
> **状态 / Status：** ⬜  
> **做法 / Approach：** 在 `/readyz` 或 harness/ABI 暴露已知 sandbox backend id 列表（投影现有 router：local/landlock/remote-mock/remote-e2b 等）；不新开厂商。无新表。  
> Expose known sandbox backend ids on `/readyz` or harness/ABI from the existing router; no new vendors. No new table.

## 任务板 / Board

| ID | 任务 / Task | 状态 |
|----|-------------|------|
| VX35-1 | 响应字段 + OpenAPI | ⬜ |
| VX35-2 | 单测与 scale/readyz 对齐 | ⬜ |
| VX35-3 | CHANGELOG | ⬜ |

## 验收 / Verify

```bash
make openapi-check
go test ./internal/api/ ./internal/sandbox/ -count=1 -run 'Readyz|Backend|Sandbox'
```
