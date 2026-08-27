# 设计归属（design）

**本目录只放「系统应如何构造」的设计正文。**  
排期与完成度 → [`../plan/`](../plan/README.md)；发布勾选与证据 → [`../progress/`](../progress/README.md)。

## 阅读顺序

1. [`PRD-需求文档.md`](PRD-需求文档.md) — 场景与范围  
2. [`HLD-总体设计.md`](HLD-总体设计.md) — 模块与数据  
3. [`HLD-Harness与沙盒.md`](HLD-Harness与沙盒.md) — **v2** Harness + 沙盒（DH/DX）  
4. [`ARCH-架构与技术选型.md`](ARCH-架构与技术选型.md) — 技术选型与演进  
4. [`M3-多租户与Postgres演进.md`](M3-多租户与Postgres演进.md) — 存储/租户专项  
5. [`../appendices/`](../appendices/README.md) — 可执行规范与 Schema  

## 归属边界

| 属于 design | 不属于（请放到别处） |
|-------------|----------------------|
| 需求、架构、选型、演进路线 | Sprint 排期、TODO → `plan/` |
| 协议/Schema 说明（appendices） | 发布勾选、烟测 → `progress/` / `checklists/` |
| 设计未决 TODO（带负责人） | 门禁证据 → `evidence/` |

## 路径说明

规范资产仍在 `doc/appendices/`（脚本与历史链接依赖），逻辑上归属设计域，由本 README 索引。
