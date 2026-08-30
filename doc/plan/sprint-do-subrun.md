# Sprint DO：Sub-run（方案 C）Implementation Plan

> **方案：** 已批准 **C** = spawn 执行 + 深度/工具白名单 + 事件树 + Quest 树视图  
> **Goal:** 父 Run 可 spawn 子 Run；`parentRunId`/`rootRunId`/`depth`；工具白名单闸门；`run.spawned` 事件与 `/runs/{id}/tree`。  
> **状态：** ✅ 完成（方案 C）

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DO-1 | runs 谱系列 SQL 27 + Create/Spawn | ✅ |
| DO-2 | 工具白名单 + maxDepth Policy | ✅ |
| DO-3 | `GET /runs/{id}/tree` + 事件 | ✅ |
| DO-4 | Quest 页 Run 树 + vitest | ✅ |
| DO-5 | openapi / docs / 回归 | ✅ |

## API

| Method | Path | 说明 |
|--------|------|------|
| POST | `/api/v1/runs/{runId}/sub-runs` | Spawn 子 Run |
| GET | `/api/v1/runs/{runId}/tree` | 以 root 为根的 Run 树 |

## Policy

- `depth+1 > harness.subRun.maxDepth`（默认 2）→ 拒绝
- 子 Run 默认工具白名单：`read,grep,find,ls,git.status,git.diff`（请求可收紧，不可无声明放宽到 danger）
- 事件：父 `run.spawned`；子 `run.started` 含 `parentRunId`

## 退出标准

- [x] SQL 27；openapi-check 绿
- [x] 超深 spawn 被拒；白名单外工具 deny
- [x] Quest 可选中 Run 看子树
