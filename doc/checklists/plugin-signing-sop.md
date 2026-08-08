# 插件签名密钥轮换 SOP（Sprint CR）

> 对应 ARCH §3.2 / 附录 H §6。本地烟测：`make plugin-sign-smoke`。

## 1. 算法与材料

| 项 | 值 |
|----|-----|
| 算法 | `hmac-sha256` |
| 材料 | `name\nversion\nprotocol\nabi\nendpoint`（`pluginabi.SignMaterial`） |
| HTTP | `POST /api/v1/plugins` 的 `signature`（hex） |
| gRPC | `RegisterRequest.signature` **或** capability `ash.sign.hmac=<hex>` |
| 环境 | `ASH_PLUGIN_SIGNING_KEY`；可选 `ASH_PLUGIN_SIGNING_REQUIRED=1` |

生成签名：

```bash
go run ./cmd/cli plugin-sign \
  --name otel --version 1.0.0 --protocol grpc --abi ash.plugin.v1 \
  --endpoint 127.0.0.1:7443 --key "$ASH_PLUGIN_SIGNING_KEY"
```

## 2. 首次上线

1. 生成高熵密钥（≥32 字节随机），仅存入密钥保险柜 / K8s Secret。
2. Worker 设置 `ASH_PLUGIN_SIGNING_KEY`（生产建议同时 `ASH_PLUGIN_SIGNING_REQUIRED=1`）。
3. 滚动 Worker；确认 `GET /api/v1/plugins/abi` → `signingRequired=true`、`signingKeyConfigured=true`。
4. 对每个外部插件：`plugin-sign` → 写入包内 `signature.txt` / 注册请求。
5. 跑 `make plugin-sign-smoke`；再对真实插件做一次注册验收。

## 3. 密钥轮换（双钥窗口）

目标：旧密钥签名的插件在窗口期内仍可验证，新密钥用于新注册。

| 步 | 动作 | 验证 |
|----|------|------|
| 1 | 生成 `KEY_NEW`，暂存保险柜 | — |
| 2 | 维护窗：Worker 先切到 `KEY_NEW`（短中断可接受时） | `/plugins/abi` 仍 reporting configured |
| 3 | 用 `KEY_NEW` 重签所有在用插件并重新 `POST /plugins` | 注册 201；`PLUGIN_SIGNATURE_INVALID` =0 |
| 4 | 归档旧密钥销毁记录 + 插件清单 | 变更单 / 审计 |

> MVP 单密钥：轮换即短暂拒绝旧签名。若需零中断双钥，后续迭代可加 `ASH_PLUGIN_SIGNING_KEYS`（逗号分隔，任一匹配即过）。

## 4. 生产 gRPC 暴露

| 项 | 约定 |
|----|------|
| 监听 | 显式设置 `ASH_PLUGIN_GRPC_ADDR`（勿依赖 dev 默认） |
| 网络 | 仅内网 / mTLS 前置；不对公网开放 |
| 鉴权 | 签名强制 + Space 绑定（`TraceContext.space`） |
| 回滚 | 去掉错误密钥或临时关闭 `ASH_PLUGIN_SIGNING_REQUIRED`（仅紧急） |

## 5. 打包清单（端到端验收）

插件发布目录建议：

```txt
plugin.json          # name/version/protocol/abi/endpoint/capabilities
signature.txt        # hmac hex
bin/                 # 插件二进制（可选）
README.md            # 对接 Worker 版本与 ABI
```

验收勾选：

- [ ] `make plugin-sign-smoke` 绿
- [ ] 错误签名 → `400` / `PLUGIN_SIGNATURE_INVALID`
- [ ] 正确签名 → 注册成功；`GET /plugins` 可见
- [ ] 轮换后旧签名失败、新签名成功
- [ ]（可选）Windows 外部观测插件实机包一次

## 相关

- [`H-Proto-服务定义(插件ABI)v0.1.md`](../appendices/H-Proto-服务定义(插件ABI)v0.1.md) §6  
- [`postgres-production-config.md`](postgres-production-config.md) 密钥轮换表  
- `internal/pluginabi/sign.go`  
