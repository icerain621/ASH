# 发布验收证据目录

本目录存放 **可提交的签字摘要**；原始日志在 `.ash/evidence/`（gitignore）。

## 生成

```bash
# MVP 发布签字（静态门禁 + 可选 Docker/云）
make mvp-signoff

# 仅 H-01~H-03 云 RDS（需 ASH_DATABASE_URL）
make cloud-acceptance

# 发布窗口本地聚合（§8，不依赖云）
make release-window-gate

# Live Worker 全量（可选，约 2min）
ASH_RELEASE_WINDOW_LIVE=1 make release-window-gate

# 含完整 mvp-signoff（约 6min，本地发布前）
ASH_RELEASE_WINDOW_INCLUDE_MVP=1 make release-window-gate
```

## 文件

| 文件 | 说明 |
|------|------|
| `mvp-signoff-latest.md` | 最近一次 `make mvp-signoff` 摘要 |
| `mvp-signoff-YYYY-MM-DD.md` | 按日归档 |
| `cloud-h01-h03-latest.md` | 云 RDS 验收摘要（`make cloud-acceptance` 后） |
| `release-window-latest.md` | 发布窗口门禁摘要（`make release-window-gate` 后） |

签字完成后更新 §11 与 [`h01-h03-cloud-signoff.md`](../checklists/h01-h03-cloud-signoff.md) 表格，并提交本目录变更。
