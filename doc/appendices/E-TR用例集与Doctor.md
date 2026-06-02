# 附录 E：TR 用例集与 `ash doctor`（v0.1）

> 本附录定义 ASH 的验证用例集（TR0–TR3）与 `ash doctor` 输出规范，用于持续集成门禁与发布准入。
>
> 冻结等级：**M0 冻结**（TR0）。TR1/TR2 为 M1 目标；TR3 为 GA 目标。

## 1. `ash doctor` 目标
- 一键运行验证集合并输出结构化报告（pass/fail + 证据链接/定位）。
- 把“质量门禁”从人工检查变为自动化、可回归、可追责。

## 2. TR 用例集
### 2.1 TR0（M0 必须）
- **TR0-01 交付闭环**
  - **目标**：从输入到输出四件套 artifacts：`diff + test_report + release_notes + rollback_plan`
  - **证据**：RunSummary（含 artifacts digest）+ run_events 片段
- **TR0-02 事件流可观测**
  - **目标**：SSE 实时事件可消费；断线后 Last-Event-ID 续传可追平
  - **证据**：SSE 录制（含 lastEventId/seq）+ run_events 表
- **TR0-03 回放基础**
  - **目标**：同一 `runId` 可回放；关键 artifacts digest 一致（允许时间戳差异）
  - **证据**：两次回放的 digest 对照表

### 2.2 TR1（M1）
- **TR1-01 Provider 替换**
  - 同一场景用两家 provider 跑通；降级策略生效并可解释
- **TR1-02 Tool/DSL Schema 校验**
  - tool 入参/DSL 非法必须被拒绝并产生可读错误
- **TR1-03 引用门禁**
  - requireCitations 的 step 缺引用必须阻断或降级为 human confirm
- **TR1-04 Checkpoint 恢复强化**
  - kill worker 后 resume 能继续完成，且状态一致
- **TR1-05 记忆候选链路**
  - L1+ 只能写 candidate；必须 review 后才能 approved

### 2.3 TR2（M1，安全与合规）
- **TR2-01 Prompt 注入红队**
  - 恶意文档/issue 不得触发危险 tool；拦截率 100%（用例集）
- **TR2-02 Secret 落盘扫描**
  - 事件/日志/记忆/审计中不出现明文 secret（或必须脱敏）
- **TR2-03 权限矩阵**
  - 角色/场景不同工具权限不同，deny 有证据
- **TR2-04 MCP 隔离**
  - MCP 工具超时/异常不污染主事件流，不导致 run 崩溃

### 2.4 TR3（GA，规模化）
- **TR3-01 记忆迁移兼容**
  - vN→vN+1 后旧记录可读，检索语义不变（回归集）
- **TR3-02 灾备降级**
  - 向量库不可用时降级到 FTS/BM25 仍可完成交付
- **TR3-03 成本/延迟 SLO**
  - token/任务与 P95 延迟不超阈值，超阈值自动告警
- **TR3-04 审计可追责**
  - 任一交付物可追溯到 runId/traceId/tool calls/model choices

## 3. `ash doctor` 命令与输出规范（建议）
### 3.1 CLI 形式
- `ash doctor --suite TR0`
- `ash doctor --suite TR1`
- `ash doctor --all`
- `ash doctor --format json|md --out path`

### 3.2 输出（JSON）
```json
{
  "suite": "TR0",
  "startedAt": 1714310000000,
  "finishedAt": 1714310123000,
  "results": [
    {
      "id": "TR0-01",
      "status": "pass",
      "runId": "run_xxx",
      "evidence": [
        {"kind": "artifact", "ref": "diff", "digest": "sha256:..."},
        {"kind": "eventRange", "ref": "run_events:seq=1..120"}
      ]
    }
  ],
  "summary": {
    "pass": 3,
    "fail": 0
  }
}
```

### 3.3 输出（Markdown）
- 列出 pass/fail
- fail 必须包含：
  - 失败原因（可读）
  - 关联 runId/traceId
  - 可复现步骤（如何重跑）
  - 关键证据链接（event seq 范围、artifacts digest）

**TODO（负责人：测试/平台）**：定义 TR0 的合成探测输入 repo 与固定数据集（避免漂移）。  
**验收方式**：同一输入在 7 天内重复执行，TR0 结果稳定。

