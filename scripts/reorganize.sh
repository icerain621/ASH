#!/usr/bin/env bash
# DEPRECATED: 早期 monorepo 布局脚本。当前仓库已采用根目录 cmd/internal + 扁平 doc/。
# 勿再执行本脚本（会错误拆分现行文档）。文档归档见 doc/archive/README.md。
set -euo pipefail
echo "reorganize.sh is deprecated (2026-08-08)." >&2
echo "Canonical layout: cmd/ internal/ frontend/ doc/ (see README.md and doc/README.md)." >&2
echo "Archived product drafts: doc/archive/product-mvp-draft/" >&2
exit 1
