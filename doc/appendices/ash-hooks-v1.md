# ash.hooks.v1 — 声明式 PreToolUse 策略

> **实现**：`internal/hooks` · **Sprint**：EW11  
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
| `event` | 目前仅 `PreToolUse`（工具调用前） |
| `tool` | 工具名匹配：精确名、`*`（任意）、`prefix*`、`*suffix`；空/省略 = 任意 |
| `risk` | 可选；若填写则与上下文 risk 大小写不敏感相等才匹配 |
| `action` | `allow` \| `deny` \| `ask`（大小写不敏感） |
| `reason` | 可选；决策原因/审计文案 |

## 决策语义

| action | 含义（EW12 接入 Run 后） |
|--------|-------------------------|
| `allow` | 继续执行工具 |
| `deny` | 拒绝工具调用（fail-closed） |
| `ask` | 进入既有 waiting_approval 流程 |

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

Hooks 配置随 **既有** SpacePolicy 读写携带；EW11 不新增专用 Hooks HTTP 路径。Run 侧 PreToolUse 接入见 Sprint EW12。

## 审计事件（EW12+ / EW13）

Run 在 PreToolUse 求值且 **非默认 allow**（`action != allow` 或 `ruleIndex >= 0`）时，同一 payload 追加两条事件（severity `info`）：

| 事件 type | 用途 |
|-----------|------|
| `hook.pre_tool_use` | 生命周期锚点：PreToolUse 已求值 |
| `hook.decision` | 决策锚点：与上条 payload 相同，便于管控/检索按「决策」过滤 |

**Payload 字段（稳定 JSON 键，camelCase）：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `stepId` | string | 当前 Run 步骤 ID |
| `tool` | string | 工具名 |
| `risk` | string | 工具风险级别（与 tool_chain 一致） |
| `event` | string | 固定 `PreToolUse`（hooks 事件枚举，非 SSE type） |
| `action` | string | `allow` \| `deny` \| `ask` |
| `reason` | string | 规则或 fail-closed 说明 |
| `ruleIndex` | number | 命中规则下标；`-1` 表示无规则匹配（如配置加载失败 deny） |

**关联事件：**

- `deny` 另发 `policy.denied`（`matrix`: `hooks.pre_tool_use`）。
- `ask` 另发 `gate.waiting_approval`（`gate`: `hook_pre_tool_use`）。

控制台：**Agent Chat → Trajectory** 展示 `hook.*`；**Details** 侧栏只读 KV；**Reviews → 编排流程** 可读空间 `bodyJson.hooks` 规则列表（薄投影）。交互原型 `#gov/hooks` 见 [`doc/prototypes/agent-absorb/README.md`](../prototypes/agent-absorb/README.md)。
