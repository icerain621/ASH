# Sprint DL：Repo Profile + Wiki（方案 B）Implementation Plan

> **方案：** 已批准 **B** = 即时计算、不落库  
> **Goal:** `internal/knowledge` 扫描 Profile + 只读 Wiki 投影；知识 Tab；Run 准备注入 `profile:` / `wiki:` contextRefs。  
> **状态：** ✅ 完成（方案 B）

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DL-1 | Profile 扫描 + Wiki 投影 + API | ✅ |
| DL-2 | prepareExecutionContext 注入 contextRefs | ✅ |
| DL-3 | 控制台知识 Tab + vitest | ✅ |
| DL-4 | openapi / docs / 回归 | ✅ |

## API（无新表）

| Method | Path | 说明 |
|--------|------|------|
| GET | `/api/v1/repos/profile?repoRoot=` | 即时 Repo Profile |
| GET | `/api/v1/wiki/pages?repoRoot=&q=` | Wiki 页列表（投影） |
| GET | `/api/v1/wiki/pages/{pageId}` | 单页详情 |

## 退出标准

- [x] Profile 能识别常见语言/测试命令
- [x] Wiki 仅投影 approved memory（+ synthetic overview）
- [x] Run contextRefs 含 `profile:` / `wiki:` 前缀（有内容时）
- [x] `/ui/knowledge` 可用；openapi-check 绿
