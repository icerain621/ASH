# ash.execpolicy.v1 — 声明式执行能力地板

> **实现**：`internal/execpolicy` · **Sprint**：EW14  
> **挂载**：SpacePolicy `bodyJson.execPolicy`；可选从 Harness Profile SpecJSON 合成  
> **非目标（EW15）**：尚未接入 `sandbox.ResolveSandboxMode*`

## 版本与形状

- **Schema 版本**：常量 `ash.execpolicy.v1`（`Policy.version`）；若填写则必须精确匹配，否则加载 fail-closed。
- **顶层**：

```json
{
  "version": "ash.execpolicy.v1",
  "network": { "egress": "deny" },
  "fs": { "mode": "read-only" },
  "process": { "exec": "ask" }
}
```

空文档 / 缺字段 = **未声明地板**（不降低现有隔离；见合并规则）。

## 能力字段

| 轴 | 字段 | 取值（严 → 松） | 说明 |
|----|------|-----------------|------|
| 网络 | `network.egress` | `deny` › `ask` › `allow` | 出站网络 |
| 文件系统 | `fs.mode` | `none` › `read-only` › `workspace-write` › `unrestricted` | 与沙箱 FS 语义对齐；`none` ≈ isolated 卷外不可写 |
| 进程 | `process.exec` | `deny` › `ask` › `allow` | 是否允许拉起子进程/工具 exec |

大小写不敏感；未知枚举 → 解析错误。

## 合并（V6-D3：更严者优先）

`Merge(a, b)` 对每个轴独立取更严值：

- 一侧 **unset（空）** 不投票 → 保留另一侧；
- 两侧均 unset → 结果 unset；
- 结果非空时 `version` 规范为 `ash.execpolicy.v1`。

典型用法：`Merge(FromSpaceBodyJSON(...), FromHarnessSpecJSON(...))`。

## SpacePolicy 挂载

```json
{
  "execPolicy": {
    "version": "ash.execpolicy.v1",
    "network": { "egress": "deny" },
    "fs": { "mode": "none" },
    "process": { "exec": "deny" }
  }
}
```

- 缺 `execPolicy`、为 `null`、或 `bodyJson` 为空/`{}` → 空策略。
- Put SpacePolicy 时若存在 `execPolicy` 则必须可解析（`spacepolicy.validateBodyJSON` fail-closed）。
- Helper：`spacepolicy.ExecPolicyFromBodyJSON` / `execpolicy.FromSpaceBodyJSON`。

## Harness Profile（可选）

`FromHarnessSpecJSON(specJSON)`：

1. 若顶层有 `execPolicy` 对象 → 按 v1 解析（扩展字段；不写入 harness JSON Schema）；
2. 否则从 `spec.sandbox.defaultMode` / `spec.sandbox.network` **合成**地板：
   - `network` ← `sandbox.network`（`deny`|`allow`，亦可 `ask`）；
   - `fs.mode` ← `off→unrestricted` · `read-only` · `workspace-write` · `isolated→none`；
   - `process.exec` ← 有模式时默认 `allow`（隔离由 FS/网络表达）。

## 与 OpenAPI / EW15

配置随既有 SpacePolicy（及 Harness Spec）读写携带；EW14 **不**新增专用 HTTP 路径。  
EW15：合并结果喂入沙箱地板 + Doctor 能力位；无配置不得降级现有隔离。
