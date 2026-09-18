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
