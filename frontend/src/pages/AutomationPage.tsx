import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  CheckCircle,
  Download,
  KeyRound,
  Link,
  Network,
  Plus,
  RotateCw,
  Save,
  ShieldCheck,
  Square,
  Trash2,
  Workflow,
} from "lucide-react";
import { useEffect, useState, type FormEvent } from "react";
import {
  applyAuditRetention,
  approveApproval,
  createAuditExport,
  createSecret,
  deleteSecret,
  getAuditPolicy,
  getAuditExportAccess,
  getPluginABIProfile,
  getStorageProfile,
  listApprovals,
  listAuditExports,
  listAuditLogs,
  listMCPTools,
  listModelProviders,
  listPlugins,
  listSecrets,
  registerMCPTool,
  registerPlugin,
  rejectApproval,
  rotateSecret,
  updateAuditPolicy,
  verifyPlugin,
} from "@/modules/platform/api/platform.api";
import { ImproveProposalsPane } from "@/components/ImproveProposalsPane";
import { getCurrentSpaceId } from "@/services/http/client";
import { shortId } from "@/shared/utils/format";

function auditPayloadSnippet(payload?: string) {
  if (!payload) return "-";
  return payload.length > 132 ? payload.slice(0, 129) + "..." : payload;
}

function capabilitySummary(raw?: string) {
  if (!raw) return "-";
  try {
    const values = JSON.parse(raw);
    if (Array.isArray(values)) return values.join(", ") || "-";
  } catch {
    return raw;
  }
  return raw;
}

export function AutomationPage() {
  const qc = useQueryClient();
  const activeSpaceId = getCurrentSpaceId();
  const [retentionDays, setRetentionDays] = useState("365");
  const [redactPayload, setRedactPayload] = useState(false);
  const [retentionMessage, setRetentionMessage] = useState("");
  const [exportMessage, setExportMessage] = useState("");
  const [secretName, setSecretName] = useState("");
  const [secretValue, setSecretValue] = useState("");
  const [secretDescription, setSecretDescription] = useState("");
  const [secretMessage, setSecretMessage] = useState("");
  const [rotateValues, setRotateValues] = useState<Record<string, string>>({});
  const [approvalReasons, setApprovalReasons] = useState<Record<string, string>>({});
  const [approvalMessage, setApprovalMessage] = useState("");
  const [toolMessage, setToolMessage] = useState("");
  const [pluginMessage, setPluginMessage] = useState("");
  const providersQuery = useQuery({
    queryKey: ["model-providers", activeSpaceId],
    queryFn: listModelProviders,
  });
  const toolsQuery = useQuery({
    queryKey: ["mcp-tools", activeSpaceId],
    queryFn: listMCPTools,
  });
  const pluginsQuery = useQuery({
    queryKey: ["plugins", activeSpaceId],
    queryFn: listPlugins,
  });
  const pluginABIQuery = useQuery({
    queryKey: ["plugin-abi", activeSpaceId],
    queryFn: getPluginABIProfile,
  });
  const storageQuery = useQuery({
    queryKey: ["storage-profile", activeSpaceId],
    queryFn: getStorageProfile,
  });
  const secretsQuery = useQuery({
    queryKey: ["secrets", activeSpaceId],
    queryFn: listSecrets,
  });
  const approvalsQuery = useQuery({
    queryKey: ["approvals", activeSpaceId],
    queryFn: () => listApprovals({ status: "pending", limit: 100 }),
  });
  const auditQuery = useQuery({
    queryKey: ["audit-logs", activeSpaceId],
    queryFn: () => listAuditLogs({ limit: 50 }),
  });
  const auditExportsQuery = useQuery({
    queryKey: ["audit-exports", activeSpaceId],
    queryFn: listAuditExports,
  });
  const auditPolicyQuery = useQuery({
    queryKey: ["audit-policy", activeSpaceId],
    queryFn: getAuditPolicy,
  });

  useEffect(() => {
    if (!auditPolicyQuery.data) return;
    setRetentionDays(String(auditPolicyQuery.data.retentionDays || 365));
    setRedactPayload(auditPolicyQuery.data.redactPayload);
  }, [auditPolicyQuery.data]);

  const refreshGovernance = async () => {
    await Promise.all([
      qc.invalidateQueries({ queryKey: ["approvals"] }),
      qc.invalidateQueries({ queryKey: ["audit-logs"] }),
      qc.invalidateQueries({ queryKey: ["audit-exports"] }),
      qc.invalidateQueries({ queryKey: ["audit-policy"] }),
      qc.invalidateQueries({ queryKey: ["plugins"] }),
      qc.invalidateQueries({ queryKey: ["plugin-abi"] }),
      qc.invalidateQueries({ queryKey: ["secrets"] }),
      qc.invalidateQueries({ queryKey: ["storage-profile"] }),
      qc.invalidateQueries({ queryKey: ["runs"] }),
    ]);
  };

  const approveMut = useMutation({
    mutationFn: ({ approvalId, reason }: { approvalId: string; reason: string }) =>
      approveApproval(approvalId, {
        actorId: "console",
        reason,
      }),
    onSuccess: async () => {
      setApprovalMessage("approval accepted");
      await refreshGovernance();
    },
    onError: (error) => setApprovalMessage(error instanceof Error ? error.message : "approval failed"),
  });

  const cancelMut = useMutation({
    mutationFn: ({ approvalId, reason }: { approvalId: string; reason: string }) =>
      rejectApproval(approvalId, {
        actorId: "console",
        reason,
      }),
    onSuccess: async () => {
      setApprovalMessage("approval rejected");
      await refreshGovernance();
    },
    onError: (error) => setApprovalMessage(error instanceof Error ? error.message : "rejection failed"),
  });

  const registerToolMut = useMutation({
    mutationFn: registerMCPTool,
    onSuccess: async (tool) => {
      setToolMessage(`registered ${tool.name}`);
      await refreshGovernance();
    },
    onError: (error) => setToolMessage(error instanceof Error ? error.message : "mcp register failed"),
  });

  const registerPluginMut = useMutation({
    mutationFn: registerPlugin,
    onSuccess: async (plugin) => {
      setPluginMessage(`registered ${plugin.name}`);
      await refreshGovernance();
    },
    onError: (error) => setPluginMessage(error instanceof Error ? error.message : "plugin register failed"),
  });

  const updatePolicyMut = useMutation({
    mutationFn: () =>
      updateAuditPolicy({
        retentionDays: Number(retentionDays || 365),
        redactPayload,
      }),
    onSuccess: async (policy) => {
      setRetentionMessage(`policy saved: ${policy.retentionDays}d`);
      await refreshGovernance();
    },
    onError: (error) => setRetentionMessage(error instanceof Error ? error.message : "policy update failed"),
  });

  const retentionMut = useMutation({
    mutationFn: (dryRun: boolean) => applyAuditRetention({ dryRun }),
    onSuccess: async (res) => {
      setRetentionMessage(
        res.dryRun ? `dry run matched ${res.matched}` : `deleted ${res.deleted} of ${res.matched}`,
      );
      await refreshGovernance();
    },
    onError: (error) => setRetentionMessage(error instanceof Error ? error.message : "retention failed"),
  });

  const verifyPluginMut = useMutation({
    mutationFn: (pluginId: string) => verifyPlugin(pluginId),
    onSuccess: refreshGovernance,
  });

  const createAuditExportMut = useMutation({
    mutationFn: createAuditExport,
    onSuccess: async (item) => {
      setExportMessage(`exported ${shortId(item.id)}`);
      await refreshGovernance();
    },
    onError: (error) => setExportMessage(error instanceof Error ? error.message : "audit export failed"),
  });

  const auditExportAccessMut = useMutation({
    mutationFn: (exportId: string) => getAuditExportAccess(exportId),
    onSuccess: (access) => {
      setExportMessage(`access ${shortId(access.exportId)} ${shortId(access.digest.replace("sha256:", ""))}`);
    },
    onError: (error) => setExportMessage(error instanceof Error ? error.message : "audit export access failed"),
  });

  const createSecretMut = useMutation({
    mutationFn: () =>
      createSecret({
        name: secretName.trim(),
        value: secretValue,
        description: secretDescription.trim(),
        scope: { runtime: "execgo" },
      }),
    onSuccess: async (secret) => {
      setSecretName("");
      setSecretValue("");
      setSecretDescription("");
      setSecretMessage(`created ${secret.name}`);
      await refreshGovernance();
    },
    onError: (error) => setSecretMessage(error instanceof Error ? error.message : "secret create failed"),
  });

  const rotateSecretMut = useMutation({
    mutationFn: ({ id, value }: { id: string; value: string }) => rotateSecret(id, { value }),
    onSuccess: async (secret) => {
      setRotateValues((current) => ({ ...current, [secret.id]: "" }));
      setSecretMessage(`rotated ${secret.name}`);
      await refreshGovernance();
    },
    onError: (error) => setSecretMessage(error instanceof Error ? error.message : "secret rotate failed"),
  });

  const deleteSecretMut = useMutation({
    mutationFn: (secretId: string) => deleteSecret(secretId),
    onSuccess: async () => {
      setSecretMessage("deleted");
      await refreshGovernance();
    },
    onError: (error) => setSecretMessage(error instanceof Error ? error.message : "secret delete failed"),
  });

  const submitSecret = () => {
    if (!secretName.trim() || !secretValue) {
      setSecretMessage("name and value required");
      return;
    }
    createSecretMut.mutate();
  };

  const setRotateValue = (secretId: string, value: string) => {
    setRotateValues((current) => ({ ...current, [secretId]: value }));
  };

  const setApprovalReason = (approvalId: string, value: string) => {
    setApprovalReasons((current) => ({ ...current, [approvalId]: value }));
  };

  const decideApproval = (approvalId: string, decision: "approve" | "reject") => {
    const reason = (approvalReasons[approvalId] || "").trim();
    if (!reason) {
      setApprovalMessage("reason required");
      return;
    }
    if (decision === "approve") approveMut.mutate({ approvalId, reason });
    else cancelMut.mutate({ approvalId, reason });
  };

  const submitMCPTool = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const formData = new FormData(event.currentTarget);
    registerToolMut.mutate({
      name: String(formData.get("name") || ""),
      server: String(formData.get("server") || ""),
      risk: String(formData.get("risk") || "medium"),
    });
  };

  const submitPlugin = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const formData = new FormData(event.currentTarget);
    const capabilities = String(formData.get("capabilities") || "")
      .split(/[\n,]/)
      .map((item) => item.trim())
      .filter(Boolean);
    registerPluginMut.mutate({
      name: String(formData.get("name") || ""),
      version: String(formData.get("version") || ""),
      protocol: String(formData.get("protocol") || "http"),
      abi: String(formData.get("abi") || "") || undefined,
      endpoint: String(formData.get("endpoint") || ""),
      capabilities,
    });
  };

  return (
    <section className="panel active">
      <div className="page-kicker">
        <Workflow size={17} strokeWidth={1.8} />
        Automation
      </div>
      <div className="page-heading">
        <div>
          <h1>自动化</h1>
          <p>查看模型路由、MCP 工具和执行边界状态。</p>
          <span className="scope-badge">Space: {activeSpaceId}</span>
        </div>
      </div>
      <div className="split">
        <div className="pane">
          <div className="pane-title">
            <h2>Model Router</h2>
            <span>{providersQuery.data?.items.length ?? 0} 个 provider</span>
          </div>
          <table className="table">
            <thead>
              <tr>
                <th>ID</th>
                <th>Provider</th>
                <th>Role</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              {(providersQuery.data?.items ?? []).map((provider) => (
                <tr key={provider.id}>
                  <td>{provider.id}</td>
                  <td>{provider.provider}</td>
                  <td>{provider.role}</td>
                  <td>{provider.status}</td>
                </tr>
              ))}
              {!providersQuery.data?.items.length && (
                <tr className="empty-row">
                  <td colSpan={4}>暂无 provider 配置。</td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
        <div className="pane">
          <div className="pane-title">
            <h2>MCP Tools</h2>
            <span>{toolsQuery.data?.items.length ?? 0} 个工具</span>
          </div>
          <form className="form-grid compact-form" onSubmit={submitMCPTool}>
            <label>
              Name
              <input name="name" required placeholder="repo.search" />
            </label>
            <label>
              Server
              <input name="server" required placeholder="https://mcp.example.com/rpc" />
            </label>
            <label>
              Risk
              <select name="risk" defaultValue="medium">
                <option value="low">low</option>
                <option value="medium">medium</option>
                <option value="high">high</option>
              </select>
            </label>
            <button className="btn primary" type="submit" disabled={registerToolMut.isPending}>
              <Plus size={15} strokeWidth={1.8} />
              Register
            </button>
            <span className="form-message">{toolMessage || activeSpaceId}</span>
          </form>
          <table className="table">
            <thead>
              <tr>
                <th>Tool</th>
                <th>Server</th>
                <th>Risk</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              {(toolsQuery.data?.items ?? []).map((tool) => (
                <tr key={tool.id}>
                  <td>
                    <Network size={14} strokeWidth={1.8} /> {tool.name}
                  </td>
                  <td>{tool.server}</td>
                  <td>{tool.risk}</td>
                  <td>{tool.status}</td>
                </tr>
              ))}
              {!toolsQuery.data?.items.length && (
                <tr className="empty-row">
                  <td colSpan={4}>暂无 MCP 工具。</td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
        <div className="pane">
          <div className="pane-title">
            <h2>Plugins</h2>
            <span>{pluginABIQuery.data?.currentAbi ?? `${pluginsQuery.data?.items.length ?? 0} 个插件`}</span>
          </div>
          <div className="abi-strip">
            <span>{pluginABIQuery.data?.supportedProtocols.join("/") ?? "-"}</span>
            <span>{pluginABIQuery.data?.breakingPolicy ?? "-"}</span>
            <span title={pluginABIQuery.data?.protoFiles.map((file) => `${file.path} ${file.digest}`).join("\n")}>
              {pluginABIQuery.data?.protoFiles.length ?? 0} proto
            </span>
          </div>
          <form className="form-grid compact-form" onSubmit={submitPlugin}>
            <label>
              Name
              <input name="name" required placeholder="delivery-observer" />
            </label>
            <label>
              Version
              <input name="version" required placeholder="0.1.0" />
            </label>
            <label>
              Protocol
              <select name="protocol" defaultValue="http">
                <option value="http">http</option>
                <option value="grpc">grpc</option>
                <option value="mcp">mcp</option>
              </select>
            </label>
            <label>
              ABI
              <input name="abi" placeholder={pluginABIQuery.data?.currentAbi ?? "ash.plugin/v0.1"} />
            </label>
            <label className="wide-field">
              Endpoint
              <input name="endpoint" required placeholder="http://127.0.0.1:19091" />
            </label>
            <label className="wide-field">
              Capabilities
              <input name="capabilities" placeholder="observability, release.gate" />
            </label>
            <button className="btn primary" type="submit" disabled={registerPluginMut.isPending}>
              <Plus size={15} strokeWidth={1.8} />
              Register
            </button>
            <span className="form-message">{pluginMessage || activeSpaceId}</span>
          </form>
          <table className="table">
            <thead>
              <tr>
                <th>Name</th>
                <th>ABI</th>
                <th>Status</th>
                <th>Capability</th>
                <th>Action</th>
              </tr>
            </thead>
            <tbody>
              {(pluginsQuery.data?.items ?? []).map((plugin) => (
                <tr key={plugin.id}>
                  <td title={plugin.endpoint}>{plugin.name}</td>
                  <td>
                    {plugin.protocol}/{plugin.abi}
                  </td>
                  <td>
                    <span className={"status-pill " + (plugin.compatible ? "ok" : "err")}>
                      <span className="status-dot" />
                      {plugin.status || (plugin.compatible ? "verified" : "incompatible")}
                    </span>
                  </td>
                  <td title={plugin.capabilities}>{capabilitySummary(plugin.capabilities)}</td>
                  <td>
                    <button
                      className="btn mini"
                      disabled={verifyPluginMut.isPending}
                      type="button"
                      onClick={() => verifyPluginMut.mutate(plugin.id)}
                    >
                      Verify
                    </button>
                  </td>
                </tr>
              ))}
              {!pluginsQuery.data?.items.length && (
                <tr className="empty-row">
                  <td colSpan={5}>暂无插件注册。</td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
        <div className="pane">
          <div className="pane-title">
            <h2>Storage</h2>
            <span>{storageQuery.data?.database.dialect ?? "-"}</span>
          </div>
          <table className="table">
            <thead>
              <tr>
                <th>Layer</th>
                <th>Profile</th>
                <th>Status</th>
                <th>Detail</th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td>Database</td>
                <td>{storageQuery.data?.database.dialect ?? "-"}</td>
                <td>
                  <span className="status-pill ok">
                    <span className="status-dot" />
                    ready
                  </span>
                </td>
                <td title={storageQuery.data?.database.dataDir}>
                  {storageQuery.data?.database.urlConfigured ? "url" : "dev sqlite"}
                </td>
              </tr>
              <tr>
                <td>Artifacts</td>
                <td>{storageQuery.data?.artifactStore.kind ?? "-"}</td>
                <td>
                  <span className={"status-pill " + (storageQuery.data?.artifactStore.ready ? "ok" : "err")}>
                    <span className="status-dot" />
                    {storageQuery.data?.artifactStore.ready ? "ready" : "not ready"}
                  </span>
                </td>
                <td title={storageQuery.data?.artifactStore.uri || storageQuery.data?.artifactStore.error}>
                  {storageQuery.data?.artifactStore.objectStore ? "object store" : "local fs"}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div className="pane">
          <div className="pane-title">
            <h2>
              <KeyRound size={15} strokeWidth={1.8} />
              Secrets
            </h2>
            <span>{secretsQuery.data?.items.length ?? 0} 个 secret</span>
          </div>
          <div className="secret-form">
            <label>
              Name
              <input
                autoComplete="off"
                placeholder="OPENAI_API_KEY"
                value={secretName}
                onChange={(event) => setSecretName(event.target.value)}
              />
            </label>
            <label>
              Value
              <input
                autoComplete="new-password"
                placeholder="********"
                type="password"
                value={secretValue}
                onChange={(event) => setSecretValue(event.target.value)}
              />
            </label>
            <label>
              Description
              <input
                autoComplete="off"
                placeholder="provider"
                value={secretDescription}
                onChange={(event) => setSecretDescription(event.target.value)}
              />
            </label>
            <button
              className="btn mini icon-only"
              disabled={createSecretMut.isPending}
              title="Create secret"
              type="button"
              onClick={submitSecret}
            >
              <Plus size={13} strokeWidth={1.8} />
            </button>
            <span>{secretMessage || activeSpaceId}</span>
          </div>
          <table className="table secret-table">
            <thead>
              <tr>
                <th>Name</th>
                <th>Digest</th>
                <th>Rotate</th>
                <th>Action</th>
              </tr>
            </thead>
            <tbody>
              {(secretsQuery.data?.items ?? []).map((secret) => (
                <tr key={secret.id}>
                  <td title={secret.description || secret.id}>
                    <span className="secret-name">{secret.name}</span>
                  </td>
                  <td title={secret.valueDigest}>
                    <code>{secret.valueDigest ? shortId(secret.valueDigest.replace("sha256:", "")) : "-"}</code>
                  </td>
                  <td>
                    <input
                      autoComplete="new-password"
                      className="inline-secret-input"
                      placeholder="********"
                      type="password"
                      value={rotateValues[secret.id] ?? ""}
                      onChange={(event) => setRotateValue(secret.id, event.target.value)}
                    />
                  </td>
                  <td>
                    <div className="row-actions">
                      <button
                        className="btn mini icon-only"
                        disabled={rotateSecretMut.isPending || !(rotateValues[secret.id] ?? "")}
                        title="Rotate secret"
                        type="button"
                        onClick={() => rotateSecretMut.mutate({ id: secret.id, value: rotateValues[secret.id] ?? "" })}
                      >
                        <RotateCw size={13} strokeWidth={1.8} />
                      </button>
                      <button
                        className="btn mini err icon-only"
                        disabled={deleteSecretMut.isPending}
                        title="Delete secret"
                        type="button"
                        onClick={() => deleteSecretMut.mutate(secret.id)}
                      >
                        <Trash2 size={13} strokeWidth={1.8} />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
              {!secretsQuery.data?.items.length && (
                <tr className="empty-row">
                  <td colSpan={4}>暂无 secret。</td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>
      <div className="split governance-grid">
        <div className="pane">
          <div className="pane-title">
            <h2>
              <ShieldCheck size={15} strokeWidth={1.8} />
              Approval Queue
            </h2>
            <span>{approvalMessage || `${approvalsQuery.data?.items.length ?? 0} 个待处理`}</span>
          </div>
          <table className="table">
            <thead>
              <tr>
                <th>Approval</th>
                <th>Run</th>
                <th>Gate</th>
                <th>Reason</th>
                <th>Decision reason</th>
                <th>Created</th>
                <th>Action</th>
              </tr>
            </thead>
            <tbody>
              {(approvalsQuery.data?.items ?? []).map((approval) => (
                <tr key={approval.id}>
                  <td title={approval.id}>{shortId(approval.id)}</td>
                  <td title={approval.runId}>{shortId(approval.runId)}</td>
                  <td>
                    <span className="status-pill idle">
                      <span className="status-dot" />
                      {approval.gate || "human"}
                    </span>
                  </td>
                  <td title={approval.reason}>{approval.reason || approval.stepId}</td>
                  <td>
                    <input
                      className="inline-secret-input"
                      placeholder="审批理由"
                      value={approvalReasons[approval.id] ?? ""}
                      onChange={(event) => setApprovalReason(approval.id, event.target.value)}
                    />
                  </td>
                  <td>{approval.createdAt ? new Date(approval.createdAt).toLocaleString() : "-"}</td>
                  <td>
                    <div className="row-actions">
                      <button
                        className="btn mini ok icon-only"
                        type="button"
                        title="Approve"
                        disabled={approveMut.isPending || cancelMut.isPending}
                        onClick={() => decideApproval(approval.id, "approve")}
                      >
                        <CheckCircle size={14} strokeWidth={1.8} />
                      </button>
                      <button
                        className="btn mini err icon-only"
                        type="button"
                        title="Reject"
                        disabled={approveMut.isPending || cancelMut.isPending}
                        onClick={() => decideApproval(approval.id, "reject")}
                      >
                        <Square size={13} strokeWidth={1.8} />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
              {!approvalsQuery.data?.items.length && (
                <tr className="empty-row">
                  <td colSpan={7}>暂无待审批请求。</td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
        <div className="pane">
          <div className="pane-title">
            <h2>Audit Logs</h2>
            <span>{auditQuery.data?.items.length ?? 0} 条事件</span>
          </div>
          <div className="audit-policy">
            <label>
              Retention
              <input
                min={1}
                max={3650}
                type="number"
                value={retentionDays}
                onChange={(event) => setRetentionDays(event.target.value)}
              />
            </label>
            <label className="check-row">
              <input
                checked={redactPayload}
                type="checkbox"
                onChange={(event) => setRedactPayload(event.target.checked)}
              />
              Redact
            </label>
            <button
              className="btn mini icon-only"
              disabled={updatePolicyMut.isPending}
              title="Save audit policy"
              type="button"
              onClick={() => updatePolicyMut.mutate()}
            >
              <Save size={13} strokeWidth={1.8} />
            </button>
            <button
              className="btn mini"
              disabled={retentionMut.isPending}
              type="button"
              onClick={() => retentionMut.mutate(true)}
            >
              Dry run
            </button>
            <button
              className="btn mini err icon-only"
              disabled={retentionMut.isPending}
              title="Apply retention"
              type="button"
              onClick={() => retentionMut.mutate(false)}
            >
              <Trash2 size={13} strokeWidth={1.8} />
            </button>
            <span>{retentionMessage || `${auditPolicyQuery.data?.retentionDays ?? 365}d`}</span>
          </div>
          <div className="pane-title subhead">
            <h3>Exports</h3>
            <div className="row-actions">
              <span>{exportMessage || `${auditExportsQuery.data?.items.length ?? 0} 个导出`}</span>
              <button
                className="btn mini icon-only"
                disabled={createAuditExportMut.isPending}
                title="Create audit export"
                type="button"
                onClick={() => createAuditExportMut.mutate()}
              >
                <Download size={13} strokeWidth={1.8} />
              </button>
            </div>
          </div>
          <table className="table audit-export-table">
            <thead>
              <tr>
                <th>Export</th>
                <th>Status</th>
                <th>Digest</th>
                <th>Size</th>
                <th>Action</th>
              </tr>
            </thead>
            <tbody>
              {(auditExportsQuery.data?.items ?? []).map((item) => (
                <tr key={item.id}>
                  <td title={item.uri}>{shortId(item.id)}</td>
                  <td>
                    <span className={"status-pill " + (item.status === "completed" ? "ok" : "idle")}>
                      <span className="status-dot" />
                      {item.status}
                    </span>
                  </td>
                  <td title={item.digest}>
                    <code>{item.digest ? shortId(item.digest.replace("sha256:", "")) : "-"}</code>
                  </td>
                  <td>{item.sizeBytes ? `${item.sizeBytes}b` : "-"}</td>
                  <td>
                    <button
                      className="btn mini icon-only"
                      disabled={auditExportAccessMut.isPending || item.status !== "completed"}
                      title="Get export access"
                      type="button"
                      onClick={() => auditExportAccessMut.mutate(item.id)}
                    >
                      <Link size={13} strokeWidth={1.8} />
                    </button>
                  </td>
                </tr>
              ))}
              {!auditExportsQuery.data?.items.length && (
                <tr className="empty-row">
                  <td colSpan={5}>暂无导出。</td>
                </tr>
              )}
            </tbody>
          </table>
          <table className="table audit-table">
            <thead>
              <tr>
                <th>Event</th>
                <th>Actor</th>
                <th>Run</th>
                <th>Created</th>
                <th>Payload</th>
              </tr>
            </thead>
            <tbody>
              {(auditQuery.data?.items ?? []).map((entry) => (
                <tr key={entry.id}>
                  <td>{entry.eventType}</td>
                  <td>{entry.actorId || "-"}</td>
                  <td title={entry.runId}>{entry.runId ? shortId(entry.runId) : "-"}</td>
                  <td>{entry.createdAt ? new Date(entry.createdAt).toLocaleString() : "-"}</td>
                  <td>
                    <code title={entry.payloadJSON}>{auditPayloadSnippet(entry.payloadJSON)}</code>
                  </td>
                </tr>
              ))}
              {!auditQuery.data?.items.length && (
                <tr className="empty-row">
                  <td colSpan={5}>暂无审计事件。</td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>
      <div className="split governance-grid">
        <ImproveProposalsPane />
      </div>
    </section>
  );
}
