# Sprint DK：Quest 工作台 v1（方案 C）Implementation Plan

> **方案：** 已批准 **C** = A（独立 Quest 页 + 看板 + Diff + 步骤评分 + contextRefs）+ **深 Diff 行级批注**  
> **Goal:** `/ui/quest` 看板驱动审查；可对 unified diff 按文件/行批注；步骤可评分；manifest 写 `contextRefs`。  
> **状态：** ✅ 完成（2026-08-29）

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DK-1 | Manifest.`contextRefs` + GET diff 正文 API | ✅ |
| DK-2 | `diff_review_comments` + CRUD API（file/line） | ✅ |
| DK-3 | Quest 看板 API + QuestPage（看板/Diff/评分） | ✅ |
| DK-4 | openapi / docs / 回归 | ✅ |

## API

| Method | Path | 说明 |
|--------|------|------|
| GET | `/api/v1/quest/board` | 列：plans / running / waiting_approval / finished |
| GET | `/api/v1/runs/{runId}/diff` | 解析 unified diff → files/hunks/lines + raw |
| GET/POST | `/api/v1/runs/{runId}/diff/comments` | 行级批注 |
| POST | `/api/v1/runs/{runId}/steps/{stepId}/rate` | feedback `run_step` |

## 数据

- SQL **25** / RLS **45**：`diff_review_comments`
- Manifest 顶层 `contextRefs[]`（来自 EvidenceRefs）

## 退出标准

- [ ] 成品 run 的 manifest 含 contextRefs（有检索时非空或空数组字段存在）
- [ ] Diff 可按行发表评论并回读
- [ ] Quest 页看板 + Diff 审查 + 步骤评分可用
- [ ] openapi-check 绿
