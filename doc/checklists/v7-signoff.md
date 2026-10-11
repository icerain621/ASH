# ASH v7 发布签字清单 / Sign-off checklist（P13）

> 目标：确认 [`v7.x-program.md`](../plan/v7.x-program.md) P0–P12 测试绿，跑通 `make v7-signoff`，**人工确认后再打 `v7.0.0` tag**。  
> Confirm P0–P12 green, pass `make v7-signoff`, then tag **`v7.0.0` by hand**.  
> 程序：[`../plan/v7.x-program.md`](../plan/v7.x-program.md) · 签字模板：[`../evidence/v7-signatures-template.md`](../evidence/v7-signatures-template.md)  
> 范围波次：`v7.0`…`v7.13-release-scope.md`（本清单对应 P13 门禁）。

## 前置 / Preconditions

| # | 项 | 命令 / 文件 | 期望 |
|---|-----|-------------|------|
| 1 | 程序水位 | `doc/plan/v7.x-program.md` | 标明 P0–P12（或更新后的）验收已绿 |
| 2 | 各波 scope | `doc/plan/v7.0`…`v7.13-release-scope.md` | §1–§5；状态含「已绿」或「已冻结」 |
| 3 | v7 门禁 | `make v7-signoff` | 烟测绿；**不**自动打 tag |
| 4 | 证据准备 | `make v7-signoff-prep` | 写出 `doc/evidence/v7-signatures-latest.md`（机器字段）；四人仍人工签 |

## 自动门禁（`make v7-signoff`）

- [x] `v7.0`…`v7.13` scope 结构与水位标记
- [x] `make openapi-check`（经 `scripts/openapi-check.sh`）
- [x] Doctor：`TestBOOTSuite` / `TestQUALITYSuite` / `TestTR0Suite`
- [x] F06 / 镜头：`TestReplan*` / `TestParseAndValidate_verifyReplan` / `TestAdmit*`
- [x] 评审 / 模板相关 Go 过滤
- [x] 前端：SessionHistoryList / ThreadTimeline / AgentTemplatesPage Vitest

## 功能收口（P0–P12 已落地，门禁只抽检）

- [x] Context Pack · 记忆引用门 · 有上限循环 · 五模板  
- [x] 真压缩 / 薄 Workflow / Doctor↔评审分离 / 插件门禁  
- [x] 三镜头观测 · G01 持久化 · 控制台 IA · `ash review` · 记忆三计数 · 有上限再规划  

## Out（本代不验收）

- 云 RDS · 自动 tag · 无审批自动重启 run · v6.8 · Cordis · YOLO · Hermes 全 IM  

## Tag 纪律

本清单与 `make v7-signoff` **不会**自动执行 `git tag`。

## 修订

| 日期 | 说明 |
|------|------|
| 2026-10-11 | P13：`make v7-signoff` + 清单 / 签字模板 |
