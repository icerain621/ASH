# Sprint DX74 — gRPC Register 签名门禁 / Signed Register gate

> **前置 / Prerequisite：** DX73  
> **状态 / Status：** ✅  
> **做法 / Approach：** 强制签名时，无签名或坏签名的 gRPC `Register` 返回 `PLUGIN_SIGNATURE_INVALID` 且**不落库**，与 HTTP 一致。合法签名与 capability 内嵌签名仍接受。  
> When signing is required, unsigned or bad gRPC `Register` returns `PLUGIN_SIGNATURE_INVALID` and does **not** persist, matching HTTP. Valid signatures and capability-embedded signatures still accept.

## 任务板 / Board

| ID | 任务 / Task | 状态 |
|----|-------------|------|
| DX74-1 | 签名失败提前返回、不 Save | ✅ |
| DX74-2 | 单测：无签名拒绝 / 字段签名 / capability 签名 | ✅ |
| DX74-3 | HTTP 注册路径不改 | ✅ |

## 验收 / Verify

```bash
go test ./internal/pluginabi/ -count=1 -run 'TestRegistryServerRejectsUnsigned|TestRegistryServerRegister|TestRegistryServerRejectsIncompatible'
```
