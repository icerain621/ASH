# Sprint DN：Skills 目录（方案 C）Implementation Plan

> **方案：** 已批准 **C** = `SKILL.md` 索引 + 场景/Harness 绑定 + Run `skill:` contextRefs  
> **Goal:** 扫描仓库 Skills；API 列表/详情；场景 `skills:` 与 Harness `skills[]` 绑定；prepareExecution 注入。  
> **状态：** ✅ 完成（方案 C）

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DN-1 | `internal/skills` 解析/扫描 + 样例 SKILL.md | ✅ |
| DN-2 | API `GET /skills` `/skills/{id}` + 场景 `skills:` | ✅ |
| DN-3 | prepareExecutionContext 注入 `skill:` | ✅ |
| DN-4 | Automation 页 Skills 面板 + vitest | ✅ |
| DN-5 | openapi / docs / 回归 | ✅ |

## 约定

- 路径：优先 `.ash/skills/<name>/SKILL.md`，兼容 `skills/<name>/SKILL.md`
- Frontmatter：必填 `name`、`description`（Agent Skills 子集）
- contextRef：`skill:<name>`
- 注入范围：场景 `skills` ∪ active Harness `skills`（声明为空则不注入）

## API

| Method | Path | 说明 |
|--------|------|------|
| GET | `/api/v1/skills?repoRoot=` | 列表 |
| GET | `/api/v1/skills/{skillId}?repoRoot=` | 详情（含 body） |

## 退出标准

- [x] 能扫到样例 Skill；openapi-check 绿
- [x] 场景声明的 skill 进入 Run contextRefs
- [x] Automation 页可见 Skills
