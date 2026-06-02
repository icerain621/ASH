# 附录 F：Artifacts 规范与 Digest / 保留 / 导出 / 回放一致性（v0.1）

> 本附录定义 ASH 的交付物（Artifacts）规范：类型、元数据、存储布局、digest 算法、保留与清理策略、导出格式，以及回放一致性要求。
>
> 冻结等级：**M0 冻结**（四件套 artifacts + manifest + digest）。对象存储/多租户为 P2+。

## 1. Artifact 的定义
### 1.1 基本字段
每个 Artifact 以“**可引用 + 可校验 + 可导出**”为最低要求，建议字段：
- `id`：唯一标识（ULID/UUID）
- `type`：ArtifactType（见 2）
- `name`：可读名称（可选）
- `uri`：存储位置（本地路径/对象存储 key/URL）
- `digest`：内容摘要（sha256）
- `contentType`：`text/plain` / `application/json` / `text/markdown` 等
- `sizeBytes`：大小
- `createdAt`：时间戳
- `producer`：生成者（stepId/role/tool/model）
- `meta`：扩展字段（JSON）

## 2. ArtifactType（M0 必须）
- `diff`：代码变更（patch/diff）
- `test_report`：测试报告（结构化 JSON + 可选原始日志）
- `release_notes`：发布说明（Markdown）
- `rollback_plan`：回滚方案（Markdown）

（P1+ 可选：`adr`、`scan_report`、`incident_postmortem`、`build_manifest` 等）

## 3. 存储布局（默认本地，可配置）
### 3.1 默认本地布局（M0）
建议在工作目录或用户目录下建立可回放的 run 目录（可配置根路径）：

```text
.ash/
  runs/
    <runId>/
      run.json                 # RunSummary（含 scenarioVersion、输入摘要、引用的 memory ids）
      events.sqlite            # 可选：run_events 的导出（或直接引用主库）
      checkpoints/             # checkpoint 快照（per_step）
      artifacts/
        manifest.json          # artifacts 清单（强制）
        diff.patch
        test_report.json
        release_notes.md
        rollback_plan.md
      audit/
        audit.jsonl            # 可选：审计导出（脱敏）
```

### 3.2 可配置项
- `ASH_RUNS_DIR`：覆盖 `.ash/runs` 根目录
- `ASH_ARTIFACTS_MAX_BYTES`：单 run artifacts 总大小上限（超过则阻断或压缩）

**TODO（负责人：平台）**：确定 Windows/WSL/Linux 的默认路径策略与权限。  
**验收方式**：在三平台上创建 run 目录并可导出/清理。

## 4. `manifest.json`（强制）
每个 run 必须生成 artifact 清单，作为回放与审计锚点：

```json
{
  "runId": "run_xxx",
  "scenario": {"name": "feature_delivery", "scenarioVersion": "1.0.0"},
  "createdAt": 1714310000000,
  "artifacts": [
    {
      "id": "art_diff_01",
      "type": "diff",
      "uri": "artifacts/diff.patch",
      "digest": "sha256:...",
      "contentType": "text/plain",
      "sizeBytes": 12345,
      "producer": {"stepId": "code.implement", "role": "Coder", "by": "tool:apply_patch"}
    }
  ]
}
```

运行时校验 schema：`docs/appendices/schemas/artifact-manifest.v0.1.schema.json`。

## 5. Digest 算法（sha256）
### 5.1 通用规则
- `digest = sha256(<canonical bytes>)`
- 输出格式：`sha256:<hex>`

### 5.2 文本文件（diff/md/txt）
- canonical bytes = 文件的原始字节
- 允许平台差异风险：换行符（CRLF/LF）
  - **M0 约束**：写 artifacts 时统一使用 LF（`\n`），避免跨平台漂移。

### 5.3 JSON 文件（test_report/结构化输出）
- canonical bytes = **canonical JSON**（字段稳定排序 + 无多余空白 + 统一换行）
- **M0 约束**：由统一的 JSON 序列化器生成（禁止手写拼接）。

**TODO（负责人：后端）**：指定 canonical JSON 的实现方式（库/策略），并写入实现说明。  
**验收方式**：同对象序列化 100 次 digest 不变；跨平台一致。

## 6. 保留、清理与导出（M0 最小策略）
### 6.1 保留策略
- 默认保留最近 `N=200` 个 runs 或最近 `30` 天（可配置）
- 超出上限：按 LRU 清理最老 runs
- 清理必须保留最小索引（runId、时间、状态、artifacts digest）

### 6.2 导出格式（用于审计/复盘/对照实验）
- `ash export run <runId> --format zip`：导出 run 目录（脱敏后）
- `ash export doctor --suite TR0`：导出 doctor 报告 + 证据引用

## 7. 回放一致性要求（TR0-03）
回放一致性以“可解释差异”为准：
- **必须一致**
  - artifacts digest（diff/test_report/release_notes/rollback_plan）
  - 引用的 memory record ids（或明确标记“使用最新记忆版本”导致差异）
  - 事件序列的关键类型顺序（run/step/tool 的主干）
- **允许不一致（但必须标记）**
  - 时间戳、随机种子（若存在）
  - 模型输出的“非交付性文本”（不影响 artifacts）

