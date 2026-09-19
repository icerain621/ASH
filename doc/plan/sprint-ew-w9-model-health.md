# Sprint EW · 模型席位健康位

> 前置：Doctor `M4-MDL-01` · `GET /model-router/providers`  
> 原则：Composer 只投影状态，不改 permissionMode / providerKind 枚举。

| # | Sprint | 状态 |
|---|--------|------|
| **EW91** | ChatSeats 显示 Provider 目录健康 | ✅ |
| **EW92** | 签字 | ✅ |

## EW91

Model 席位旁读 `listModelProviders`，展示 primary/fallback 的 status（`available` / `not_configured` 等）。未配置不算错误。

## EW92

CHANGELOG + 清单勾选。
