# ASH 附录（Appendices）

> **逻辑归属**：设计域（见 [`../design/README.md`](../design/README.md)）  
> **路径稳定**：本目录勿移动（契约与历史链接依赖）。

本目录存放“规范/Schema/指标/验收用例”等稳定材料，避免主文档过长。

- `A-事件协议与Schema.md`：事件类型表、payload schema、SSE 续传规则
- `B-RulesDSL规范与Schema.md`：`ash.rules/v0.1` DSL 语义与 JSON Schema
- `C-MemorySchema(SQLite)+迁移.md`：DDL、评审流、迁移与兼容策略
- `D-Observability-指标与告警.md`：插件配置 schema、指标清单、告警规则建议
- `E-TR用例集与Doctor.md`：TR0~TR3 用例、`ash doctor` 输出格式与门禁
- `F-Artifacts规范与Digest.md`：Artifacts 类型、manifest、digest、保留/导出、回放一致性
- `G-OpenAPI-端点清单(M0).md`：M0 必需 HTTP API 端点边界（Swagger/OpenAPI）
- `H-Proto-服务定义(插件ABI)v0.1.md`：gRPC/Buf 插件 ABI 草案与目录结构
- `I-GORM-模型映射与迁移策略.md`：GORM 模型映射与迁移路线（SQLite→Postgres）
- `J-数据分级与保留期.md`：分级表、默认保留期、脱敏样例、审计导出流程（PRD §8）
- `K-演进平面-v2.md`：**v2** 统一 Feedback、双评审队列、Improve 自进化状态机与流程图

