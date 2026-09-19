# ash.hooks.v1 — 声明式 PreToolUse / PostToolUse 策略

> **实现**：`internal/hooks` · **Sprint**：EW11 · EW111  
> **挂载**：SpacePolicy `bodyJson` 内嵌 JSON 对象键 `hooks`（非独立 HTTP 资源）

## 版本与形状

- **Schema 版本**：常量 `ash.hooks.v1`（`Config.version`）；若填写则必须精确匹配，否则加载 fail-closed。
- **顶层**：

```json
{
  "version": "ash.hooks.v1",
  "rules": [ { "...": "..." } ]
}
```

- **`rules[]`**：按数组顺序求值，**首条匹配规则**生效（不再继续扫描）。

## 规则字段

| 字段 | 说明 |
|------|------|
| `event` | `PreToolUse`（调用前）或 `PostToolUse`（成功返回后） |
| `tool` | 工具名匹配：精确名、`*`（任意）、`prefix*`、`*suffix`；空/省略 = 任意 |
| `risk` | 可选；若填写则与上下文 risk 大小写不敏感相等才匹配 |
| `action` | `allow` \| `deny` \| `ask`（大小写不敏感） |
| `reason` | 可选；决策原因/审计文案 |

## 决策语义

| action | PreToolUse | PostToolUse |
|--------|------------|-------------|
| `allow` | 继续执行工具 | 继续后续步骤 |
| `deny` | 拒绝调用 | 步骤失败并 fail run（**不回滚**已执行副作用） |
| `ask` | 进入 waiting_approval | **仅审计**（副作用已发生，不打断） |

**缺省**：无任何规则匹配 → **`allow`**（与「空间未配置 hooks」一致：不阻断）。

## Fail-closed 要点

1. **配置 version 错误**：`ParseConfigJSON` / `ConfigFromSpaceBodyJSON` 返回错误，调用方应拒绝使用该配置（不 silently 降级）。
2. **规则已匹配但 `action` 为空或未知**：求值 **deny**，理由说明 invalid/unknown action；**不得** fall through 到默认 allow。
3. **显式 `deny` 规则匹配**：deny。

未匹配字段（如未知 `event` 值）的规则在求值时跳过，不影响其他规则；这与「匹配后 action 无效必须 deny」不同。

## SpacePolicy 挂载

`bodyJson` 为 JSON 对象时，可选键：

```json
{
  "hooks": {
    "version": "ash.hooks.v1",
    "rules": []
  }
}
```

- 缺 `hooks`、为 `null`、或 `bodyJson` 为空/`{}` → 空配置（Evaluate 默认 allow）。
- `hooks` 子文档解析失败或 version 非法 → 加载错误（fail-closed）。

## 与 OpenAPI 的关系

Hooks 配置随 **既有** SpacePolicy 读写携带；不新增专用 Hooks HTTP 路径。

## 审计事件

| 时机 | 事件 type |
|------|-----------|
| PreToolUse 非默认 allow | `hook.pre_tool_use` + `hook.decision` |
| PostToolUse 非默认 allow | `hook.post_tool_use` + `hook.decision` |

**Payload 字段（稳定 JSON 键，camelCase）：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `stepId` | string | 当前 Run 步骤 ID |
| `tool` | string | 工具名 |
| `risk` | string | 工具风险级别 |
| `event` | string | `PreToolUse` 或 `PostToolUse` |
| `action` | string | `allow` \| `deny` \| `ask` |
| `reason` | string | 规则或 fail-closed 说明 |
| `ruleIndex` | number | 命中规则下标；`-1` 表示无规则匹配 |

**关联事件：**

- Pre `deny` 另发 `policy.denied`（`matrix`: `hooks.pre_tool_use`）。
- Pre `ask` 另发 `gate.waiting_approval`（`gate`: `hook_pre_tool_use`）。
- Post `deny` 另发 `policy.denied`（`matrix`: `hooks.post_tool_use`）。

控制台：**Agent Chat → Trajectory** 展示 `hook.*`；**Details** 侧栏只读 KV；**Reviews → 编排流程** 可读空间 `bodyJson.hooks` 规则列表。
