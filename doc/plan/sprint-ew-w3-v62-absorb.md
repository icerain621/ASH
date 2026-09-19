# Sprint EW · W3 v6.2 吸收任务板

> 波次：**W3**（[`ash-feature-inventory.md`](ash-feature-inventory.md)）  
> 前置：**W0–W2 ✅**  
> 原则：不新造执行面；只投影已有 Run 树与事件。中文 commit。

| # | Sprint | 状态 |
|---|--------|------|
| **EW31** | Trajectory 显示子 Run 谱系 | ✅ |
| **EW32** | Thread 时间线过滤器 | ✅ |
| **EW33** | Provider / Doctor 探针 | ✅ |
| **EW34** | W3 签字 | ✅ |

## EW31

Chat Trajectory 在有 `runId` 时拉 `GET /runs/{id}/tree`，列出 root 与子 Run。

## EW32

时间线可按全部 / 模型可见 / 界面 / 工具 / 门禁 / Hook / Compact 过滤。

## EW33

`M4-MDL-01`：Doctor 读取 `modelrouter.NewFromEnv()` 目录。未配置记为 pass（`available=0`），目录为空才 fail。M4 **12/12**，ALL **62/62**。

## EW34

CHANGELOG 与本板勾选。无新 HTTP 路径。
