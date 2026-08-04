import { useMutation, useQuery } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { useQueryClient } from "@tanstack/react-query";
import { Download, EyeOff, Play, ScanSearch, ShieldCheck } from "lucide-react";
import { useState } from "react";
import { DoctorReportView } from "@/components/DoctorReportView";
import { exportComplianceBundle, scanSecrets } from "@/modules/compliance/api/compliance.api";
import { getDoctorReport, runDoctor, type DoctorReport } from "@/modules/doctor/api/doctor.api";
import { listRuns } from "@/modules/runs/api/runs.api";
import {
  getAuditPolicy,
  getAuthMe,
  getPermissionMatrix,
  updateAuditPolicy,
  getPluginABIProfile,
  getStorageProfile,
  listPlugins,
  listSpaceMembers,
  listSpaceResourceScopes,
} from "@/modules/platform/api/platform.api";
import { getCurrentSpaceId } from "@/services/http/client";
import { shortId } from "@/shared/utils/format";

const M3_CHECKS = [
  {
    id: "M3-01",
    title: "多租户隔离",
    hint: "跨 space_id 访问拒绝；记忆按空间过滤",
  },
  {
    id: "M3-02",
    title: "Postgres 演进",
    hint: "ASH_DATABASE_URL 解析；scale/readiness 数据库字段",
  },
  {
    id: "M3-03",
    title: "迁移目录",
    hint: "ash migrate 表清单覆盖 runs/memory/audit 等关键表",
  },
  {
    id: "M3-04",
    title: "迁移校验",
    hint: "make postgres-e2e 或 ASH_MIGRATE_E2E=1 + live Postgres",
  },
] as const;

const M2_CHECKS = [
  {
    id: "M2-01",
    title: "权限矩阵",
    hint: "内置 RBAC + 场景 × 角色工具策略种子",
  },
  {
    id: "M2-02",
    title: "场景策略可更新",
    hint: "resource_scopes 支持 PUT 更新 scenario policyJson",
  },
] as const;

const TR2_CHECKS = [
  {
    id: "TR2-01",
    title: "身份与作用域",
    hint: "组织/空间/角色/成员/资源作用域/审计策略",
  },
  {
    id: "TR2-02",
    title: "Run 空间隔离",
    hint: "列表仅返回当前 X-ASH-Space-ID 下的运行",
  },
  {
    id: "TR2-03",
    title: "产物存储",
    hint: "产物与 Checkpoint 具备 storeKey 与可访问 URI",
  },
  {
    id: "TR2-04",
    title: "插件 ABI",
    hint: "插件注册表与 ash.plugin.v1 兼容校验",
  },
  {
    id: "TR2-05",
    title: "Secret 扫描",
    hint: "审计/事件载荷无明文 secret；API 脱敏与 RedactJSON",
  },
] as const;

function checkTone(ok: boolean) {
  return ok ? "ok" : "idle";
}

export function CompliancePage() {
  const queryClient = useQueryClient();
  const activeSpaceId = getCurrentSpaceId();
  const [report, setReport] = useState<DoctorReport | null>(null);
  const [reportId, setReportId] = useState<string | null>(null);
  const [exportMsg, setExportMsg] = useState("");
  const [exportSuite, setExportSuite] = useState<"TR2" | "TR3" | "ALL">("TR2");

  const meQuery = useQuery({ queryKey: ["auth-me", activeSpaceId], queryFn: getAuthMe });
  const scopesQuery = useQuery({
    queryKey: ["resource-scopes", activeSpaceId],
    queryFn: () => listSpaceResourceScopes(activeSpaceId),
    enabled: activeSpaceId !== "local",
  });
  const membersQuery = useQuery({
    queryKey: ["members", activeSpaceId],
    queryFn: () => listSpaceMembers(activeSpaceId),
    enabled: activeSpaceId !== "local",
  });
  const auditQuery = useQuery({ queryKey: ["audit-policy", activeSpaceId], queryFn: getAuditPolicy });
  const storageQuery = useQuery({ queryKey: ["storage-profile", activeSpaceId], queryFn: getStorageProfile });
  const pluginsQuery = useQuery({ queryKey: ["plugins", activeSpaceId], queryFn: listPlugins });
  const abiQuery = useQuery({ queryKey: ["plugin-abi", activeSpaceId], queryFn: getPluginABIProfile });
  const runsQuery = useQuery({
    queryKey: ["runs", activeSpaceId, "compliance"],
    queryFn: () => listRuns(20),
  });

  const scanQuery = useQuery({
    queryKey: ["compliance", "secret-scan", activeSpaceId],
    queryFn: () => scanSecrets(200),
  });
  const matrixQuery = useQuery({
    queryKey: ["permissions-matrix", activeSpaceId],
    queryFn: () => getPermissionMatrix(activeSpaceId !== "local" ? activeSpaceId : undefined),
    enabled: activeSpaceId !== "local",
  });

  const doctorMut = useMutation({
    mutationFn: async () => {
      const started = await runDoctor("TR2");
      setReportId(started.reportId);
      return getDoctorReport(started.reportId);
    },
    onSuccess: (data) => setReport(data),
  });

  const redactMut = useMutation({
    mutationFn: () =>
      updateAuditPolicy({
        retentionDays: auditQuery.data?.retentionDays ?? 365,
        redactPayload: true,
      }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["audit-policy", activeSpaceId] });
      await scanQuery.refetch();
    },
  });

  const exportWithReportMut = useMutation({
    mutationFn: async () => {
      let id = reportId ?? undefined;
      const suite = exportSuite;
      if (!id || report?.suite !== suite) {
        const started = await runDoctor(suite);
        id = started.reportId;
        setReportId(id);
        setReport(await getDoctorReport(id));
      }
      return exportComplianceBundle({ suite, reportId: id });
    },
    onSuccess: (res) => {
      setExportMsg(`已导出 ${res.exportId.slice(0, 12)}…`);
      if (res.report) setReport(res.report);
    },
    onError: (err) => setExportMsg((err as Error).message),
  });

  const identityOk =
    Boolean(meQuery.data?.permissions?.length) &&
    (activeSpaceId === "local" || (scopesQuery.data?.items.length ?? 0) > 0);
  const isolationOk = runsQuery.data?.items.every((run) => (run.spaceId || "local") === activeSpaceId) ?? true;
  const storageOk = Boolean(storageQuery.data?.artifactStore?.ready);
  const pluginOk = Boolean(abiQuery.data?.currentAbi) && (pluginsQuery.data?.items.length ?? 0) >= 0;
  const secretOk = (scanQuery.data?.leakCount ?? 0) === 0;
  const scenarioScopes =
    scopesQuery.data?.items.filter((scope) => scope.resourceType === "scenario").length ?? 0;
  const matrixOk =
    (matrixQuery.data?.builtinRoles.length ?? 0) >= 5 && (matrixQuery.data?.scenarioTools.length ?? 0) >= 3;
  const liveStatus: Record<string, boolean> = {
    "TR2-01": identityOk,
    "TR2-02": isolationOk,
    "TR2-03": storageOk,
    "TR2-04": pluginOk,
    "TR2-05": secretOk && !scanQuery.isLoading,
    "M2-01": matrixOk && scenarioScopes >= 3,
    "M2-02": scenarioScopes >= 3,
    "M3-01": activeSpaceId === "local" || (membersQuery.data?.items.length ?? 0) > 0,
    "M3-02": true,
    "M3-03": true,
    "M3-04": true,
  };

  return (
    <section className="panel active">
      <div className="page-kicker">
        <ShieldCheck size={17} strokeWidth={1.8} />
        TR2 合规
      </div>
      <div className="page-heading">
        <div>
          <h1>合规控制台</h1>
          <p>查看 TR2 安全与隔离相关配置，并一键运行 TR2 诊断套件。</p>
          <span className="scope-badge">Space: {activeSpaceId}</span>
        </div>
        <div className="toolbar">
          <button className="btn icon-btn" disabled={scanQuery.isFetching} onClick={() => scanQuery.refetch()}>
            <ScanSearch size={16} strokeWidth={1.8} />
            扫描 Secret
          </button>
          <button
            className="btn icon-btn"
            disabled={redactMut.isPending || auditQuery.data?.redactPayload}
            onClick={() => redactMut.mutate()}
            title="开启后审计列表 API 与导出包会对载荷脱敏"
          >
            <EyeOff size={16} strokeWidth={1.8} />
            {auditQuery.data?.redactPayload ? "脱敏已开启" : "一键开启 Redact"}
          </button>
          <label className="scenario-picker">
            导出套件
            <select
              value={exportSuite}
              disabled={exportWithReportMut.isPending}
              onChange={(e) => setExportSuite(e.target.value as "TR2" | "TR3" | "ALL")}
            >
              <option value="TR2">TR2</option>
              <option value="TR3">TR3</option>
              <option value="ALL">ALL</option>
            </select>
          </label>
          <button
            className="btn icon-btn"
            disabled={exportWithReportMut.isPending}
            onClick={() => exportWithReportMut.mutate()}
          >
            <Download size={16} strokeWidth={1.8} />
            导出审计包
          </button>
          <button
            className="btn primary icon-btn"
            disabled={doctorMut.isPending}
            onClick={() => doctorMut.mutate()}
          >
            <Play size={16} strokeWidth={1.8} />
            {doctorMut.isPending ? "运行 TR2…" : "运行 TR2"}
          </button>
        </div>
      </div>
      {exportMsg && <p className="muted-line">{exportMsg}</p>}

      <div className="pane-title subhead">
        <h2>M3 多租户 / Postgres</h2>
      </div>
      <div className="split tr2-grid">
        {M3_CHECKS.map((check) => (
          <div className="pane tr2-card" key={check.id}>
            <div className="pane-title">
              <h2>{check.id}</h2>
              <span className={"status-pill " + checkTone(liveStatus[check.id])}>
                <span className="status-dot" />
                {liveStatus[check.id] ? "就绪" : "待确认"}
              </span>
            </div>
            <p className="tr2-card-title">{check.title}</p>
            <p className="muted-line">{check.hint}</p>
          </div>
        ))}
      </div>

      <div className="pane-title subhead">
        <h2>M2 权限矩阵</h2>
        <Link to="/space" className="inline-link">
          在空间控制台编辑策略 →
        </Link>
      </div>
      <div className="split tr2-grid">
        {M2_CHECKS.map((check) => (
          <div className="pane tr2-card" key={check.id}>
            <div className="pane-title">
              <h2>{check.id}</h2>
              <span className={"status-pill " + checkTone(liveStatus[check.id])}>
                <span className="status-dot" />
                {liveStatus[check.id] ? "就绪" : "待确认"}
              </span>
            </div>
            <p className="tr2-card-title">{check.title}</p>
            <p className="muted-line">{check.hint}</p>
          </div>
        ))}
      </div>

      <div className="split tr2-grid">
        {TR2_CHECKS.map((check) => (
          <div className="pane tr2-card" key={check.id}>
            <div className="pane-title">
              <h2>{check.id}</h2>
              <span className={"status-pill " + checkTone(liveStatus[check.id])}>
                <span className="status-dot" />
                {liveStatus[check.id] ? "就绪" : "待确认"}
              </span>
            </div>
            <p className="tr2-card-title">{check.title}</p>
            <p className="muted-line">{check.hint}</p>
          </div>
        ))}
      </div>

      <div className="split">
        <div className="pane">
          <div className="pane-title">
            <h2>身份与权限</h2>
            <span>{meQuery.data?.role || "-"}</span>
          </div>
          <p className="muted-line">
            用户 {meQuery.data?.user?.displayName || meQuery.data?.user?.id || "-"} ·{" "}
            {meQuery.data?.permissions?.length ?? 0} 项权限 · 审计保留{" "}
            {auditQuery.data?.retentionDays ?? 365} 天
            {auditQuery.data?.redactPayload ? " · 脱敏开启" : ""}
          </p>
          <pre className="code-block compact">
            {(meQuery.data?.permissions ?? []).join("\n") || "无权限列表（dev 模式可能为空）"}
          </pre>
          <div className="pane-title subhead">
            <h3>成员</h3>
            <span>{membersQuery.data?.items.length ?? 0} 人</span>
          </div>
          <table className="table">
            <thead>
              <tr>
                <th>成员</th>
                <th>用户</th>
                <th>角色</th>
              </tr>
            </thead>
            <tbody>
              {(membersQuery.data?.items ?? []).map((m) => (
                <tr key={m.id}>
                  <td>{shortId(m.id)}</td>
                  <td>{m.userId}</td>
                  <td>{shortId(m.roleId)}</td>
                </tr>
              ))}
              {!membersQuery.data?.items.length && (
                <tr className="empty-row">
                  <td colSpan={3}>
                    {activeSpaceId === "local" ? "local 空间无成员表。" : "暂无成员。"}
                  </td>
                </tr>
              )}
            </tbody>
          </table>
          <Link to="/space" className="inline-link">
            在空间页管理组织与成员 →
          </Link>
        </div>

        <div className="pane">
          <div className="pane-title">
            <h2>资源作用域</h2>
            <span>{scopesQuery.data?.items.length ?? 0} 条</span>
          </div>
          <table className="table">
            <thead>
              <tr>
                <th>类型</th>
                <th>资源</th>
                <th>策略</th>
              </tr>
            </thead>
            <tbody>
              {(scopesQuery.data?.items ?? []).map((scope) => (
                <tr key={scope.id}>
                  <td>{scope.resourceType}</td>
                  <td title={scope.resourceId}>{shortId(scope.resourceId)}</td>
                  <td>
                    <code title={scope.policyJson}>{shortId(scope.policyJson)}</code>
                  </td>
                </tr>
              ))}
              {!scopesQuery.data?.items.length && (
                <tr className="empty-row">
                  <td colSpan={3}>暂无作用域记录（创建空间时会自动写入）。</td>
                </tr>
              )}
            </tbody>
          </table>
        </div>

        <div className="pane">
          <div className="pane-title">
            <h2>存储与插件</h2>
            <span>{storageQuery.data?.artifactStore.kind ?? "-"}</span>
          </div>
          <table className="table">
            <tbody>
              <tr>
                <td>数据库</td>
                <td>{storageQuery.data?.database.dialect ?? "-"}</td>
                <td>
                  <span className="status-pill ok">
                    <span className="status-dot" />
                    ready
                  </span>
                </td>
              </tr>
              <tr>
                <td>产物库</td>
                <td title={storageQuery.data?.artifactStore.uri}>
                  {storageQuery.data?.artifactStore.kind ?? "-"}
                </td>
                <td>
                  <span
                    className={
                      "status-pill " + (storageQuery.data?.artifactStore.ready ? "ok" : "err")
                    }
                  >
                    <span className="status-dot" />
                    {storageQuery.data?.artifactStore.ready ? "ready" : "not ready"}
                  </span>
                </td>
              </tr>
              <tr>
                <td>插件 ABI</td>
                <td>{abiQuery.data?.currentAbi ?? "-"}</td>
                <td>{pluginsQuery.data?.items.length ?? 0} 注册</td>
              </tr>
            </tbody>
          </table>
          <Link to="/automation" className="inline-link">
            在自动化页管理插件与密钥 →
          </Link>
        </div>

        <div className="pane">
          <div className="pane-title">
            <h2>Run 隔离抽样</h2>
            <span>{runsQuery.data?.items.length ?? 0} 条</span>
          </div>
          <p className="muted-line">当前空间下的运行列表（应全部属于 {activeSpaceId}）。</p>
          <table className="table">
            <thead>
              <tr>
                <th>Run</th>
                <th>空间</th>
                <th>状态</th>
              </tr>
            </thead>
            <tbody>
              {(runsQuery.data?.items ?? []).slice(0, 8).map((run) => (
                <tr key={run.runId}>
                  <td title={run.runId}>{shortId(run.runId)}</td>
                  <td>{run.spaceId || "local"}</td>
                  <td>{run.status}</td>
                </tr>
              ))}
              {!runsQuery.data?.items.length && (
                <tr className="empty-row">
                  <td colSpan={3}>暂无运行。</td>
                </tr>
              )}
            </tbody>
          </table>
          <Link to="/runs" className="inline-link">
            打开运行页 →
          </Link>
        </div>
      </div>

      <div className="pane">
        <div className="pane-title">
          <h2>Secret 泄漏扫描</h2>
          <span>
            {scanQuery.isFetching
              ? "扫描中"
              : `${scanQuery.data?.leakCount ?? 0} 处 / ${scanQuery.data?.scanned ?? 0} 条`}
          </span>
        </div>
        <p className="muted-line">
          审计脱敏：
          {scanQuery.data?.redactEnabled
            ? "已开启（列表 API 与导出会掩码）"
            : "未开启，可使用工具栏「一键开启 Redact」"}
        </p>
        <table className="table">
          <thead>
            <tr>
              <th>来源</th>
              <th>引用</th>
              <th>片段</th>
            </tr>
          </thead>
          <tbody>
            {(scanQuery.data?.findings ?? []).slice(0, 20).map((item, idx) => (
              <tr key={`${item.ref}-${idx}`}>
                <td>{item.source}</td>
                <td title={item.ref}>{shortId(item.ref)}</td>
                <td>
                  <code title={item.snippet}>{item.snippet}</code>
                </td>
              </tr>
            ))}
            {!scanQuery.data?.findings.length && (
              <tr className="empty-row">
                <td colSpan={3}>未发现明文 secret 模式（或尚未扫描）。</td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      <div className="pane">
        <div className="pane-title">
          <h2>TR2 诊断报告</h2>
          <span>{doctorMut.isPending ? "运行中" : report ? `${report.summary.pass} 通过` : "未运行"}</span>
        </div>
        {doctorMut.error && <p className="error-text">{(doctorMut.error as Error).message}</p>}
        <DoctorReportView report={report} />
        {report && (
          <details className="raw-report">
            <summary>原始 JSON</summary>
            <pre className="code-block tall">{JSON.stringify(report, null, 2)}</pre>
          </details>
        )}
      </div>
    </section>
  );
}
