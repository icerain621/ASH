# MVP v0.1 发布签字组织清单

> **目标**：在自动化门禁通过后，一次性完成 **§11 四人签字** 与 **范围冻结**，并同步到可提交的 `doc/evidence/` 摘要。  
> **入口**：`config/signoff.env` → `make signoff-apply` → `make signoff-gate`

## 1. 前置条件

| # | 项 | 命令 | 通过标准 |
|---|-----|------|----------|
| 1 | 自动化门禁 | `make release-window-gate` | `release-window-latest.md` 本地项 ✅ |
| 2 | MVP 静态签字 | `make mvp-signoff` | 除云 RDS 外自动化 ✅ |
| 3 | 本地 live（推荐） | `make local-readiness-gate` | H-04~H-09 live ✅ |
| 4 | Git 锚点 | `git rev-parse --short HEAD` | 与证据报告一致 |

## 2. 评审会议议程（建议 30min）

1. 过一遍 [`mvp-release-scope.md`](../mvp-release-scope.md) §2/§3（包含/不包含）
2. 展示 [`mvp-signoff-latest.md`](../evidence/mvp-signoff-latest.md) 自动化验收表
3. 展示 [`release-window-latest.md`](../evidence/release-window-latest.md) 切换检查
4. 确认 [`11-mvp-release-checklist.md`](../11-mvp-release-checklist.md) 未勾项仅为云 RDS / 值班表
5. 四人当场确认并填写 `config/signoff.env`
6. 产品负责人设置 `ASH_SCOPE_FREEZE_SIGNED=1`

## 3. 操作步骤

```bash
cp config/signoff.env.example config/signoff.env
# 编辑四人姓名、日期；产品确认后 ASH_SCOPE_FREEZE_SIGNED=1
# 含空格的值请加引号，例如：ASH_SIGNOFF_PRODUCT_NAME="张三"

set -a && source config/signoff.env && set +a
make signoff-apply
make signoff-gate
make scope-freeze-gate    # 应无 WARN
```

## 4. 签字后提交

```bash
git add doc/evidence/mvp-signatures-latest.md \
        doc/evidence/mvp-signoff-latest.md \
        doc/11-mvp-release-checklist.md \
        doc/mvp-release-scope.md
git commit -m "docs: MVP v0.1 sign-off roster and scope freeze"
```

> `config/signoff.env` 含姓名，默认 **不入库**（加入 `.gitignore` 若团队需要）。

## 5. 文档映射

| 角色 | 回填位置 |
|------|----------|
| 产品 | `mvp-release-scope.md` §6 · 范围冻结状态 |
| 技术 | `mvp-signatures-latest.md` · `mvp-signoff-latest.md` |
| 测试 | 同上 |
| 发布 | 同上 · 可选 `release-window-runbook.md` roster |

## 6. 与 MVP 清单对应

| MVP § | 项 | 本清单 |
|-------|-----|--------|
| 1 | 范围确认 / 冻结 | §2 议程 + `signoff-apply` |
| 8 | 发布窗口值班 | `signoff.env` 可选字段 |
| 11 | 四人签字 | `signoff-apply` + `signoff-gate` |

相关：[`h01-h03-cloud-signoff.md`](h01-h03-cloud-signoff.md)（云切换签字，发布窗口另做）
