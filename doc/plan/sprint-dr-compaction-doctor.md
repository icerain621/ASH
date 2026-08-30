# Sprint DR：compaction + Doctor M4/M5（方案 C）Implementation Plan

> **方案：** 已批准 **C** = tool spill + compaction 事件 + Doctor M4 全量 + M5-EVO + ALL 计数  
> **Goal:** 大工具结果落盘；上下文估测超阈触发 `harness.compaction`；M4/M5 探针可跑。  
> **状态：** ✅ 本批完成 · **无新表**

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DR-1 | spill + compaction 运行时 | ✅ |
| DR-2 | Doctor M4 HAR+SBX | ✅ |
| DR-3 | Doctor M4-EVO + M5-EVO；canary ≤10% | ✅ |
| DR-4 | ALL 计数 / docs | ✅ |

## 行为

- 工具输出 JSON &gt; `sandbox.spillMaxBytes` → `artifacts/spill_*.json` + `tool.spilled`；`tool.result.output` 换摘要
- compaction.enabled 且估测 token / 32k ≥ `triggerTokenRatio` → `harness.compaction`（每 run 至多一次）
- canary percent 默认上限 **10**（M5-EVO-04）
- Doctor suites：`M4`（6）+ `M5`（4）；ALL **53**
