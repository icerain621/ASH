# ExecGo / Codex Live Smoke（M3-05）

> 本地分类健康检查：`make execgo-health`（无需 `ASH_EXECGO_E2E`）。  
> Doctor **M3-05** live 断言需显式开启 `ASH_EXECGO_E2E=1` 且 `--agent execgo_codex`。

## 前置

1. `make execgo-bootstrap`（或已有 `execgocli` + Codex CLI）
2. ExecGo 与 runtime 可达（默认 `EXECGO_URL=http://127.0.0.1:8080`、`EXECGO_RUNTIME_URL=http://127.0.0.1:18080`）
3. `ASH_CODEX_BIN` 指向 Codex 可执行文件（`execgo-health.sh` 会校验）

## 步骤

| # | 动作 | 期望 |
|---|------|------|
| 1 | `make execgo-health` | 分类通过（execgocli / runtime / codex 可达） |
| 1b | `curl -s 'http://<worker>/api/v1/providers/agent' \| jq .` | `execGo.ok=true`；`selection` 反映 harness/pin |
| 2 | `ASH_EXECGO_E2E=1 make execgo-live-smoke` | `execgo-health` + Doctor **M3-05** passed（非 skipped） |
| 2b | 等价 CLI | `ASH_EXECGO_E2E=1 go run ./cmd/cli doctor --suite M3 --require M3-05 --agent execgo_codex` |
| 3 | `curl -s http://<worker>/readyz \| jq .liveGateHints` | 含 `ASH_EXECGO_E2E=1` 条目（Worker 进程需带该 env） |

> Sprint DQ：未钉死 agent 时，Worker 按 Harness `provider.kind` 选型；`execgo` 探测失败则回退 `static` 并写 `provider.fallback`。Live Doctor 仍须 `--agent execgo_codex` 钉死真实桥接。

## 失败分类（execgo-health）

| 类别 | 含义 | 下一步 |
|------|------|--------|
| `execgocli missing` | CLI 未安装 | `make execgo-bootstrap` |
| `codex cli missing` | Codex 未在 PATH | 设置 `ASH_CODEX_BIN` |
| runtime / URL 错误 | 服务未起或端口不对 | 检查 ExecGo 进程与 `EXECGO_*_URL` |

## 与 verify-local 集成

`scripts/verify-local.sh` 在检测到 `execgocli`（或 `.ash/execgo/.../execgocli`）时可选运行 `make execgo-health`；失败不阻断其余门禁。

## 相关

- Doctor M3-05 实现：`internal/doctor/service.go`
- `README.md` — ExecGo doctor 示例
- Scale `/readyz` 面板 — `liveGateHints` 展示 M3-04..08 与 `ASH_CI_FIXTURE`
