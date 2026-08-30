# ASH v2.0 发布范围（冻结）

> 状态：**已冻结**（2026-08-30，产品占位确认；真人签字见 [`../checklists/v2-signoff.md`](../checklists/v2-signoff.md)）  
> 归属：[`plan/`](README.md)  
> 版本锚点：Doctor ALL **53/53** · SQL rev **27** · RLS **46** · Sprint **DH–DU**  
> 关联：[`v2-dual-core-evolution-plan.md`](v2-dual-core-evolution-plan.md) · [`mvp-release-scope.md`](mvp-release-scope.md)

## 1. 发布目标

在 v1 MVP 底座上交付 **v2 双核心（Agent × Memory）+ Harness/沙盒/演进平面 + 多入口委派**，达到可打 **`v2.0.0`** 标签的功能与文档水位（云 live / ExecGo live 仍可为环境门禁项）。

## 2. 包含范围（P0 · 本版本 In）

| 域 | 交付物（Sprint） |
|----|------------------|
| Harness | Profile CRUD/promote、Loop 事件、Provider 选型（DH / DI / DQ） |
| 沙盒 | process/Docker、danger/`off` 拒绝、hotfix/security **isolated** 地板、KPI-19（DX / DX2） |
| 演进 | feedback 类型、Reviews 队列、scenario patch、canary≤10%、KPI-17/18（DY / DZ / DR / DU） |
| Quest / Goal | from-goal、Plan approve、Quest 看板、Diff 批注（DJ / DK） |
| 知识 | Repo Profile / Wiki 即时投影、Skills、Space Rules（DL / DN / DM） |
| 编排深度 | Sub-run、verify、spill/compaction（DO / DP / DR） |
| 多入口 | CI Webhook→Run、Session API + `ash session rpc`、移动审阅（DS / DT / DU） |
| Doctor | M4/M5 探针；ALL **53** |

## 3. 不包含范围（本版本 Out）

- 云 RDS 生产切流签字（H-01～H-03 仍 ⏸，本地 Postgres 门禁已有）
- ExecGo/Codex **真实 live** 强制（H-06；代码就绪，需 `ASH_EXECGO_E2E=1`）
- ACP SDK 真接线（~~`acp_sdk` 仍回退 static~~ → **v2.1 DW 骨架已接线**；完整协议/生产 ACP 服务仍后置）
- 多区域 Active-Active / 计费 / 外部 IdP 联邦
- Skill Marketplace 全量、向量库主路径、IDE 补全

## 4. 变更冻结规则

- 冻结后仅接受 **P0 缺陷**、**文档/门禁**、以及 Out 项的环境验收证据
- 新表 DDL 必须同步 SQL rev + RLS + Doctor（若触及）
- 破坏性 API 须 OpenAPI + `make openapi-check`
- 打 **`v2.0.0` tag** 前须完成本文件签字表 + `make v2-signoff` 绿

## 5. 验收门禁（自动化）

```bash
make scope-freeze-gate    # MVP + v2 scope 结构
make v2-signoff           # v2 结构门禁 + 短回归钩子
make openapi-check
go test ./internal/doctor/... -run TestALLSuite -count=1
# 可选全量（环境许可时）：
make mvp-signoff
make release-window-gate
```

## 6. 评审签字

| 角色 | 姓名 | 日期 | 结论 |
|------|------|------|------|
| 产品 | 产品负责人（占位） | 2026-08-30 | 范围确认 / 冻结 |
| 技术 | 技术负责人（占位） | 2026-08-30 | 门禁达标（代码水位） |
| 测试 | 测试负责人（占位） | 2026-08-30 | 回归通过（Doctor ALL） |
| 发布 | 发布负责人（占位） | 2026-08-30 | 可打 tag（待真人确认） |

### Tag 步骤（人工确认后执行，本 Sprint **不自动打标**）

```bash
git tag -a v2.0.0 -m "ASH v2.0.0 dual-core GA (DH–DU frozen)"
git push origin v2.0.0
```
