import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Database, Gauge, Play } from "lucide-react";
import { useState } from "react";
import { DoctorReportView } from "@/components/DoctorReportView";
import { getDoctorReport, runDoctor, type DoctorReport } from "@/modules/doctor/api/doctor.api";
import { runMemoryMigration, sweepMemoryTTL } from "@/modules/memory/api/memory.api";
import { getPluginHealth } from "@/modules/platform/api/platform.api";
import { getReadyz } from "@/modules/health/api/health.api";
import { getScaleReadiness } from "@/modules/scale/api/scale.api";
import { getWakerQueue, getWakerStatus } from "@/modules/waker/api/waker.api";
import { getCurrentSpaceId } from "@/services/http/client";

const M3_CHECKS = [
  { id: "M3-01", title: "多租户隔离", hint: "跨 space 访问拒绝（记忆/运行）" },
  { id: "M3-02", title: "Postgres 就绪", hint: "ASH_DATABASE_URL 解析与迁移路径" },
  { id: "M3-03", title: "迁移目录", hint: "ash migrate 表清单与 schema 一致性" },
  { id: "M3-04", title: "迁移校验", hint: "ASH_MIGRATE_E2E=1 时 live postgres verify" },
  { id: "M3-05", title: "ExecGo E2E", hint: "ASH_EXECGO_E2E=1 时真实执行链路（需手动开启）" },
  { id: "M3-06", title: "Postgres RLS", hint: "ASH_POSTGRES_RLS=1 时租户策略安装与隔离" },
  { id: "M3-07", title: "ash_app 连接", hint: "ASH_DATABASE_APP_URL 时 Worker 应用角色连通性" },
  { id: "M3-08", title: "SQL 修订版本", hint: "ASH_SCHEMA_MODE=sql 时 golang-migrate 版本与 AutoMigrate 关闭" },
  { id: "M3-09", title: "运维快照契约", hint: "/readyz 与 Scale 的 dialect/otel/alerts/replay 环境一致" },
  { id: "M3-10", title: "RLS 全局表排除", hint: "memory_migrations/schema_meta 不纳入租户 RLS 策略" },
  { id: "M3-11", title: "RLS 迁移目录", hint: "迁移表全覆盖；000013–000020 与 Go catalog 一致（含 org 身份表）" },
] as const;

const TR3_CHECKS = [
  { id: "TR3-01", title: "记忆迁移兼容", hint: "catalog v2；L1/L2 默认 TTL（ASH_MEMORY_TTL_L1_DAYS / L2_DAYS 可覆盖）" },
  { id: "TR3-02", title: "灾备降级", hint: "FTS 不可用时 RAG 降级到分块检索" },
  { id: "TR3-03", title: "成本/延迟 SLO", hint: "瀑布时长与 model_cost/tool_calls 质量指标" },
  { id: "TR3-04", title: "审计可追责", hint: "traceId、事件、产物与 tool/agent 链路" },
  { id: "TR3-05", title: "指标回放一致", hint: "run_events 离线 replay 与 ash_* 计数口径一致" },
  { id: "TR3-06", title: "Postgres RAG FTS", hint: "dialect=postgres 时 tsvector 检索；SQLite 自动 skip" },
  { id: "TR3-07", title: "插件导出健康", hint: "pluginhealth 注册表 + plugin.export_failed 审计" },
  { id: "TR3-08", title: "Prometheus replay 段", hint: "ASH_METRICS_EVENT_REPLAY=1 时 /metrics 含 derive ash_* replay" },
  { id: "TR3-09", title: "OpenAPI 契约对齐", hint: "手写 /api/v1 路径与 swag 一致；2xx 无泛型 ApiResponse" },
  { id: "TR3-10", title: "Readyz 健康契约", hint: "/readyz HealthResponse 含 RLS/SQL 漂移字段；与 swag 一致" },
] as const;

export function ScalePage() {
  const qc = useQueryClient();
  const activeSpaceId = getCurrentSpaceId();
  const [report, setReport] = useState<DoctorReport | null>(null);
  const [migrateMessage, setMigrateMessage] = useState("");

  const readinessQuery = useQuery({
    queryKey: ["scale", "readiness", activeSpaceId],
    queryFn: getScaleReadiness,
  });
  const pluginHealthQuery = useQuery({
    queryKey: ["plugin-health", activeSpaceId],
    queryFn: getPluginHealth,
  });
  const readyzQuery = useQuery({
    queryKey: ["health", "readyz"],
    queryFn: getReadyz,
    refetchInterval: 30_000,
  });
  const wakerStatusQuery = useQuery({
    queryKey: ["waker", "status", activeSpaceId],
    queryFn: () => getWakerStatus(activeSpaceId),
  });
  const wakerQueueQuery = useQuery({
    queryKey: ["waker", "queue", activeSpaceId],
    queryFn: () => getWakerQueue({ spaceId: activeSpaceId, limit: 5 }),
  });

  const migrateMut = useMutation({
    mutationFn: () => runMemoryMigration({ dryRun: false }),
    onSuccess: (result) => {
      setMigrateMessage(
        result.alreadyCurrent
          ? "记忆 catalog 已是最新版本。"
          : `迁移完成 v${result.fromVersion}→v${result.toVersion}，更新 ${result.recordsUpdated} 条。`,
      );
      qc.invalidateQueries({ queryKey: ["scale", "readiness"] });
    },
    onError: (err: Error) => setMigrateMessage(err.message),
  });

  const ttlSweepMut = useMutation({
    mutationFn: () => sweepMemoryTTL({ dryRun: false }),
    onSuccess: (result) => {
      setMigrateMessage(
        result.deprecated > 0
          ? `TTL sweep：已弃用 ${result.deprecated} 条；复核队列 ${result.reviewDue} 条。`
          : `TTL sweep：无到期记录；复核队列 ${result.reviewDue} 条。`,
      );
      qc.invalidateQueries({ queryKey: ["scale", "readiness"] });
    },
    onError: (err: Error) => setMigrateMessage(err.message),
  });

  const doctorMut = useMutation({
    mutationFn: async (suite: "TR3" | "M3") => {
      const { reportId } = await runDoctor(suite);
      return getDoctorReport(reportId);
    },
    onSuccess: setReport,
  });

  const r = readinessQuery.data;
  const z = readyzQuery.data;
  const wakerEnabledCount = wakerStatusQuery.data?.duties.filter((d) => d.enabled).length ?? 0;
  const wakerQueueCount = wakerQueueQuery.data?.count ?? 0;
  const wakerTicker = wakerStatusQuery.data?.interval;

  return (
    <section className="panel active">
      <div className="page-kicker">
        <Gauge size={17} strokeWidth={1.8} />
        TR3 规模化
      </div>
      <div className="page-heading">
        <div>
          <h1>规模化就绪</h1>
          <p>查看 GA 目标相关的记忆、RAG、成本与审计基线，并运行 TR3 诊断套件。</p>
          <span className="scope-badge">Space: {activeSpaceId}</span>
        </div>
        <div className="toolbar">
          <button
            className="btn icon-btn"
            disabled={doctorMut.isPending}
            onClick={() => doctorMut.mutate("M3")}
          >
            <Play size={16} strokeWidth={1.8} />
            {doctorMut.isPending ? "运行 M3…" : "运行 M3"}
          </button>
          <button
            className="btn primary icon-btn"
            disabled={doctorMut.isPending}
            onClick={() => doctorMut.mutate("TR3")}
          >
            <Play size={16} strokeWidth={1.8} />
            {doctorMut.isPending ? "运行 TR3…" : "运行 TR3"}
          </button>
          {(r?.memoryPendingMigrationRecords ?? 0) > 0 && (
            <button
              className="btn icon-btn"
              disabled={migrateMut.isPending}
              onClick={() => migrateMut.mutate()}
              type="button"
            >
              <Database size={16} strokeWidth={1.8} />
              {migrateMut.isPending ? "迁移中…" : `记忆迁移 (${r!.memoryPendingMigrationRecords})`}
            </button>
          )}
          {((r?.memoryTTLExpiredPendingCount ?? 0) > 0 ||
            (r?.memoryTTLReviewDueCount ?? 0) > 0) && (
            <button
              className="btn icon-btn"
              disabled={ttlSweepMut.isPending}
              onClick={() => ttlSweepMut.mutate()}
              type="button"
            >
              <Database size={16} strokeWidth={1.8} />
              {ttlSweepMut.isPending
                ? "TTL 处理中…"
                : `TTL sweep (${r!.memoryTTLExpiredPendingCount ?? 0}/${r!.memoryTTLReviewDueCount ?? 0})`}
            </button>
          )}
        </div>
      </div>

      <div className="split tr2-grid">
        {M3_CHECKS.map((check) => (
          <div className="pane tr2-card" key={check.id}>
            <div className="pane-title">
              <h2>{check.id}</h2>
            </div>
            <p className="tr2-card-title">{check.title}</p>
            <p className="muted-line">{check.hint}</p>
          </div>
        ))}
      </div>

      <div className="split tr2-grid">
        {TR3_CHECKS.map((check) => (
          <div className="pane tr2-card" key={check.id}>
            <div className="pane-title">
              <h2>{check.id}</h2>
            </div>
            <p className="tr2-card-title">{check.title}</p>
            <p className="muted-line">{check.hint}</p>
          </div>
        ))}
      </div>

      <div className="pane">
        <div className="pane-title">
          <h2>就绪指标</h2>
          <span>{readinessQuery.isFetching ? "刷新中" : "快照"}</span>
        </div>
        {readinessQuery.isError && (
          <p className="error-text">{(readinessQuery.error as Error).message}</p>
        )}
        {(r?.readinessWarnings?.length ?? 0) > 0 && (
          <ul className="muted-line">
            {r!.readinessWarnings!.map((msg) => (
              <li key={msg} className="error-text">
                {msg}
              </li>
            ))}
          </ul>
        )}
        {migrateMessage && <p className="muted-line">{migrateMessage}</p>}
        <table className="table">
          <tbody>
            <tr>
              <td>记忆 Schema</td>
              <td>
                v{r?.memorySchemaVersion ?? "-"}
                {r?.memoryCatalogVersion != null
                  ? ` · catalog v${r.memoryCatalogVersion}`
                  : null}
                {(r?.memoryPendingMigrationRecords ?? 0) > 0
                  ? ` · 待迁移 ${r!.memoryPendingMigrationRecords}`
                  : null}
                {(r?.memoryTTLReviewDueCount ?? 0) > 0
                  ? ` · TTL 复核 ${r!.memoryTTLReviewDueCount}（${r!.memoryTTLReviewLeadDays ?? 7}d 窗口）`
                  : null}
                {(r?.memoryTTLExpiredPendingCount ?? 0) > 0
                  ? ` · TTL 待 sweep ${r!.memoryTTLExpiredPendingCount}`
                  : null}
              </td>
            </tr>
            <tr>
              <td>已批准记忆</td>
              <td>{r?.memoryApprovedCount ?? "-"}</td>
            </tr>
            <tr>
              <td>运行积压（inflight）</td>
              <td>
                {r
                  ? `${r.runInflightCount ?? (r.runRunningCount ?? 0) + (r.runWaitingApprovalCount ?? 0)}（running ${r.runRunningCount ?? 0} · waiting ${r.runWaitingApprovalCount ?? 0}）`
                  : "-"}
              </td>
            </tr>
            <tr>
              <td>RAG 文档 / 分块</td>
              <td>
                {r ? `${r.ragDocumentCount} / ${r.ragChunkCount}` : "-"}
                {r && (r.ragPathEntryCount != null || r.ragSymbolCount != null)
                  ? ` · 路径 ${r.ragPathEntryCount ?? 0} / 符号 ${r.ragSymbolCount ?? 0}`
                  : null}
              </td>
            </tr>
            <tr>
              <td>RAG 检索模式</td>
              <td>
                {r?.ragDefaultRetrievalMode ?? "-"}
                {r?.ragHybridAvailable ? " · Hybrid 可用" : null}
                {r?.ragFtsEngine
                  ? ` · ${r.ragFtsEngine}`
                  : r?.ragFtsAvailable != null
                    ? ` · FTS ${r.ragFtsAvailable ? "可用" : "不可用（chunk 降级）"}`
                    : null}
                {(r?.ragFallbackQueryCount ?? 0) > 0
                  ? ` · 历史降级 ${r!.ragFallbackQueryCount} 次`
                  : null}
              </td>
            </tr>
            <tr>
              <td>可观测性</td>
              <td>
                {r
                  ? `OTel ${r.otelEnabled ? "启用" : "关闭"}${r.alertsEvalInterval ? ` · 告警评估 ${r.alertsEvalInterval}` : ""}${r.memoryTTLSweepInterval ? ` · TTL sweep ${r.memoryTTLSweepInterval}` : ""}${r.metricsEventReplayEnabled ? " · 指标 replay" : ""}`
                  : "-"}
              </td>
            </tr>
            <tr>
              <td>数据保留（附录 J）</td>
              <td>
                {r
                  ? `事件 ${r.retentionEventsDays ?? "-"}d · 审计 ${r.retentionAuditDays ?? "-"}d · 产物 ${r.retentionArtifactsDays ?? "-"}d / 最近 ${r.retentionArtifactsMaxRuns ?? "-"} runs`
                  : "-"}
              </td>
            </tr>
            <tr>
              <td>插件导出健康</td>
              <td>
                {pluginHealthQuery.data
                  ? `${pluginHealthQuery.data.pluginCount} 个 · 错误 ${pluginHealthQuery.data.exportErrorsTotal} · 丢弃 ${pluginHealthQuery.data.dropCountTotal} · 过期 ${pluginHealthQuery.data.staleExportCount}`
                  : pluginHealthQuery.isLoading
                    ? "加载中"
                    : "-"}
              </td>
            </tr>
            <tr>
              <td>Waker</td>
              <td>
                {wakerStatusQuery.isLoading || wakerQueueQuery.isLoading
                  ? "加载中"
                  : wakerStatusQuery.isError || wakerQueueQuery.isError
                    ? "—"
                    : `duties enabled: ${wakerEnabledCount} · queue: ${wakerQueueCount}${wakerTicker ? ` · ticker ${wakerTicker}` : ""}`}
              </td>
            </tr>
            <tr>
              <td>模型用量 / 成本 (micros)</td>
              <td>
                {r ? `${r.modelUsageRows} / ${r.modelCostMicrosTotal}` : "-"}
              </td>
            </tr>
            <tr>
              <td>质量指标 / 审计日志</td>
              <td>
                {r ? `${r.qualityMetricRows} / ${r.auditLogRows}` : "-"}
              </td>
            </tr>
            <tr>
              <td>数据库方言</td>
              <td>{r?.databaseDialect ?? "-"}</td>
            </tr>
            <tr>
              <td>Postgres / 迁移就绪</td>
              <td>
                {r
                  ? `${r.postgresConfigured ? "postgres" : "sqlite-dev"} / ${r.migrationReady ? "yes" : "no"}`
                  : "-"}
              </td>
            </tr>
            <tr>
              <td>Worker 连接角色</td>
              <td>
                {r?.workerConnectionRole ?? "—"}
                {r?.runtimeDsnHint ? (
                  <>
                    {" "}
                    <code>{r.runtimeDsnHint}</code>
                  </>
                ) : null}
              </td>
            </tr>
            <tr>
              <td>ash_app 配置</td>
              <td>{r?.postgresAppUrlConfigured ? "ASH_DATABASE_APP_URL" : "—"}</td>
            </tr>
            <tr>
              <td>双写影子库</td>
              <td>
                {r?.dualWriteShadowUrlHint ? <code>{r.dualWriteShadowUrlHint}</code> : "—"}
              </td>
            </tr>
            <tr>
              <td>Schema 模式 / SQL 修订</td>
              <td>
                {r?.schemaMode
                  ? `${r.schemaMode} · v${r.sqlMigrationVersion ?? "?"} / ${r.sqlMigrationExpected ?? "?"}`
                  : "-"}
                {r?.schemaMode === "sql" && r.autoMigrateEnabled === false
                  ? " (AutoMigrate off)"
                  : r?.autoMigrateEnabled
                    ? " (AutoMigrate on)"
                    : null}
              </td>
            </tr>
            <tr>
              <td>迁移表数量</td>
              <td>{r?.migrationTableCount ?? "-"}</td>
            </tr>
            <tr>
              <td>双写（配置 / 运行时）</td>
              <td>
                {r
                  ? `${r.dualWriteEnabled ? "on" : "off"} / ${r.dualWriteRuntime ? (r.dualWriteSource || "on") : "off"}`
                  : "-"}
              </td>
            </tr>
            <tr>
              <td>上次 migrate sync</td>
              <td>
                {r?.lastMigrationSyncAtMs
                  ? new Date(r.lastMigrationSyncAtMs).toLocaleString()
                  : "—"}
              </td>
            </tr>
            <tr>
              <td>最近 sync 失败</td>
              <td>
                {r?.lastMigrationSyncError ? (
                  <>
                    <span className="error-text">{r.lastMigrationSyncError}</span>
                    {r.lastMigrationSyncErrorAtMs ? (
                      <>
                        {" "}
                        <span className="muted-line">
                          ({new Date(r.lastMigrationSyncErrorAtMs).toLocaleString()})
                        </span>
                      </>
                    ) : null}
                  </>
                ) : (
                  "—"
                )}
              </td>
            </tr>
            <tr>
              <td>SQLite 路径</td>
              <td>
                <code>{r?.sqlitePath ?? "-"}</code>
              </td>
            </tr>
            <tr>
              <td>Postgres RLS</td>
              <td>
                {r
                  ? `${r.postgresRLSEnabled ? "on" : "off"}${r.postgresRLSForce ? " (force)" : ""}`
                  : "-"}
              </td>
            </tr>
            <tr>
              <td>RLS 策略数</td>
              <td>
                {r?.postgresRLSEnabled
                  ? `${r.postgresRLSPolicyCount ?? 0} / ${r.postgresRLSPolicyExpected ?? 41}`
                  : "—"}
                {r?.rlsCatalogSummary ? (
                  <>
                    <br />
                    <span className="muted-line">{r.rlsCatalogSummary}</span>
                  </>
                ) : null}
              </td>
            </tr>
          </tbody>
        </table>
        <p className="muted-line">
          Postgres 迁移见 <code>doc/design/M3-多租户与Postgres演进.md</code>；云 RDS 验收见{" "}
          <code>doc/checklists/postgres-rds-e2e.md</code>（<code>make postgres-rds-e2e</code>）；本地{" "}
          <code>make postgres-e2e</code>；CLI <code>ash migrate plan|copy|verify|sync</code>。
        </p>
      </div>

      <div className="pane">
        <div className="pane-title">
          <h2>Worker /readyz</h2>
          <span>{readyzQuery.isFetching ? "刷新中" : z?.status ?? "—"}</span>
        </div>
        {readyzQuery.isError && (
          <p className="error-text">{(readyzQuery.error as Error).message}</p>
        )}
        {(z?.readinessWarnings?.length ?? 0) > 0 && (
          <ul className="muted-line">
            {z!.readinessWarnings!.map((msg) => (
              <li key={msg} className="error-text">
                {msg}
              </li>
            ))}
          </ul>
        )}
        <table className="table">
          <tbody>
            <tr>
              <td>状态 / 方言</td>
              <td>
                {z ? `${z.status} · ${z.dialect ?? "—"}` : "—"}
              </td>
            </tr>
            <tr>
              <td>Schema / SQL 修订</td>
              <td>
                {z?.schemaMode
                  ? `${z.schemaMode} · v${z.sqlMigrationVersion ?? "?"} / ${z.sqlMigrationExpected ?? "?"}`
                  : "—"}
              </td>
            </tr>
            <tr>
              <td>Postgres RLS</td>
              <td>
                {z
                  ? `${z.postgresRLSEnabled ? "active" : "off"} · ${z.postgresRLSPolicyCount ?? 0} / ${z.postgresRLSPolicyExpected ?? "—"}`
                  : "—"}
                {z?.rlsCatalogSummary ? (
                  <>
                    <br />
                    <span className="muted-line">{z.rlsCatalogSummary}</span>
                  </>
                ) : null}
              </td>
            </tr>
            <tr>
              <td>Live gates (M3)</td>
              <td>
                {(z?.liveGateHints?.length ?? 0) > 0 ? (
                  <ul className="muted-line" style={{ margin: 0, paddingLeft: "1.2rem" }}>
                    {z!.liveGateHints!.map((hint) => (
                      <li key={hint}>{hint}</li>
                    ))}
                  </ul>
                ) : (
                  <span className="muted-line">未启用 live 门禁（静态 Doctor 默认 skip M3-04/05/06/07）</span>
                )}
              </td>
            </tr>
            <tr>
              <td>可观测性</td>
              <td>
                {z
                  ? `OTel ${z.otelEnabled ? "启用" : "关闭"}${z.alertsEvalInterval ? ` · 告警 ${z.alertsEvalInterval}` : ""}${z.memoryTTLSweepInterval ? ` · TTL sweep ${z.memoryTTLSweepInterval}` : ""}${z.metricsEventReplayEnabled ? " · replay" : ""}`
                  : "—"}
              </td>
            </tr>
            <tr>
              <td>数据保留</td>
              <td>
                {z
                  ? `事件 ${z.retentionEventsDays ?? "-"}d · 审计 ${z.retentionAuditDays ?? "-"}d · 产物 ${z.retentionArtifactsDays ?? "-"}d / ${z.retentionArtifactsMaxRuns ?? "-"} runs`
                  : "—"}
              </td>
            </tr>
            <tr>
              <td>与 Scale 一致性</td>
              <td>
                {r && z
                  ? [
                      r.databaseDialect === z.dialect ? "dialect ✓" : `dialect ✗ (${r.databaseDialect} vs ${z.dialect})`,
                      r.otelEnabled === z.otelEnabled ? "otel ✓" : "otel ✗",
                      (r.sqlMigrationExpected ?? 0) === (z.sqlMigrationExpected ?? 0) ? "sqlExpected ✓" : "sqlExpected ✗",
                    ].join(" · ")
                  : "—"}
              </td>
            </tr>
          </tbody>
        </table>
        <p className="muted-line">
          无需认证的 Worker 探针；与 M3-09 / TR3-10 契约对齐。K8s 就绪探针请使用 <code>GET /readyz</code>。
        </p>
      </div>

      <div className="pane">
        <div className="pane-title">
          <h2>TR3 报告</h2>
          <span>{doctorMut.isPending ? "运行中" : report ? `${report.summary.pass} 通过` : "就绪"}</span>
        </div>
        <DoctorReportView report={report} />
      </div>
    </section>
  );
}
