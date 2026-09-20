# Ingress / Provider / Skills Hub 探测清单 / Probe checklist（VX45）

> 范围：v6.4 薄标记与适配器缝。不计费、无公网市场、无 Hermes 全 IM。  
> Scope: v6.4 thin markers and adapter seams. No billing, no public marketplace, no Hermes full IM.

## 如何读标记 / How to read the markers

| 探针 / Probe | 字段 / Field | 期望值 / Expected | 含义 / Meaning |
|--------------|--------------|-------------------|----------------|
| `GET /readyz` · scale readiness | `ingressAdapters` | `["null","webhook-github"]` | 已知入站适配器目录；**不是**多 IM Gateway |
| `GET /api/v1/model-router/providers` | `catalog` / `billing` | `org` / `none` | 组织内 Provider 目录；不计费 |
| `GET /api/v1/skills/catalog` | `marketplace` / `hub` / `billing` | `private` / `org` / `none` | 组织 Skills Hub 姿态；非公网市场 |

## 环境 / Env

| 变量 | 作用 |
|------|------|
| `ASH_INGRESS` | `null`（默认）或 `webhook-github`；未知值 fail-closed 到 null |
| `ASH_NOTIFIER` | （v6.3）`null` / `log`；与 Ingress 独立 |

## 不做 / Out

- 飞书 / 企微 / Telegram 等 IM 通道  
- 公网 Marketplace、支付  
- Active-Active  

## 快速核对 / Quick check

```bash
curl -s localhost:8080/readyz | jq '.ingressAdapters,.sandboxBackends'
curl -s localhost:8080/api/v1/model-router/providers | jq '.catalog,.billing'
curl -s localhost:8080/api/v1/skills/catalog | jq '.hub,.marketplace,.billing'
```
