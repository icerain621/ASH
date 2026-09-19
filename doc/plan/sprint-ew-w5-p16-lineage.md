# Sprint EW · P16 子代理谱系台

> 前置：**W3 谱系只读 ✅** · **P15 ✅**  
> 原则：复用 `POST /runs/{id}/sub-runs`；不新造执行面。

| # | Sprint | 状态 |
|---|--------|------|
| **EW51** | Trajectory 派生子 Run | ✅ |
| **EW52** | 签字 | ✅ |

## EW51

Chat Trajectory 谱系区：从当前 `runId` 派生（复用父 scenario）；成功后刷新树。无 scenario 时禁用按钮。节点可点选；「回主」聚焦 root。

## EW52

CHANGELOG + 板勾选。无新 HTTP 路径。
