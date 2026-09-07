# Sprint DX39 — 策略 prefer=remote（v2.9 · D2）

> **方案：** isolated 在 `ASH_SANDBOX_REMOTE` 开启且 Available 时优先 `executor=remote`；失败回退本机或 `ON_FAIL=deny`  
> **Goal:** DefaultRouter 接线；本机仍默认（未开 remote）；无新表；Doctor 计数不变  
> **状态：** ✅ 完成 · **无新表**

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX39-1 | `RegisterRemoteAvailable` + Prefer/OnFail env | ✅ |
| DX39-2 | DefaultRouter：isolated → remote → landlock → docker → process | ✅ |
| DX39-3 | `PreferRemote` 单次覆盖；`ON_FAIL=deny` | ✅ |
| DX39-4 | 单测 + sprint / TODO / CHANGELOG | ✅ |

## Env（增量）

| 变量 | 说明 |
|------|------|
| `ASH_SANDBOX_REMOTE` | `1`/`on`/`auto` → isolated **prefer** remote（仍须 Available） |
| `ASH_SANDBOX_REMOTE_ON_FAIL` | 默认 `fallback`（回退 landlock/docker/process）；`deny`/`fail`/`refuse` → 拒绝 |
| `RouteRequest.PreferRemote` | `remote`/`1` 强制 prefer（仍须 enabled）；`local`/`0`/`never` 跳过 |

## 路由顺序（isolated）

1. remote（preferred + Available）
2. landlock（preferred + Available）
3. docker（PreferDocker）
4. process

## 验收

```bash
go test ./internal/sandbox/ ./internal/sandbox/remote/ -count=1
```
