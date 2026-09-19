# Sprint DX81 — 控制台私有提示 / Private catalog line

> **前置 / Prerequisite：** DX79；[`v4.3-release-scope.md`](v4.3-release-scope.md)  
> **状态 / Status：** ✅  
> **做法 / Approach：** 现有组织 Catalog 面板加一句「私有 · 不计费」。不新开市场页。  
> Existing org catalog panels show “private, no billing”. No new marketplace page.

## 任务板 / Board

| ID | 任务 / Task | 状态 |
|----|-------------|------|
| DX81-1 | Agent Skills 组织 Catalog 一句 | ✅ |
| DX81-2 | 自动化页组织 Catalog 同一句 | ✅ |

## 验收 / Verify

```bash
cd frontend && npx vitest run src/modules/agent-session/components/SkillsCatalogPanel.test.tsx src/pages/AutomationPage.test.tsx
```
