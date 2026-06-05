import { useMutation, useQuery } from "@tanstack/react-query";
import { Gauge, Play } from "lucide-react";
import { useState } from "react";
import { DoctorReportView } from "@/components/DoctorReportView";
import { getDoctorReport, runDoctor, type DoctorReport } from "@/modules/doctor/api/doctor.api";
import { getScaleReadiness } from "@/modules/scale/api/scale.api";
import { getCurrentSpaceId } from "@/services/http/client";

const M3_CHECKS = [
  { id: "M3-01", title: "多租户隔离", hint: "跨 space 访问拒绝（记忆/运行）" },
  { id: "M3-02", title: "Postgres 就绪", hint: "ASH_DATABASE_URL 解析与迁移路径" },
  { id: "M3-03", title: "迁移目录", hint: "ash migrate 表清单与 schema 一致性" },
  { id: "M3-04", title: "迁移校验", hint: "ASH_MIGRATE_E2E=1 时 live postgres verify" },
] as const;

const TR3_CHECKS = [
  { id: "TR3-01", title: "记忆迁移兼容", hint: "Schema v1 记录可读且检索语义不变" },
  { id: "TR3-02", title: "灾备降级", hint: "FTS 不可用时 RAG 降级到分块检索" },
  { id: "TR3-03", title: "成本/延迟 SLO", hint: "瀑布时长与 model_cost/tool_calls 质量指标" },
  { id: "TR3-04", title: "审计可追责", hint: "traceId、事件、产物与 tool/agent 链路" },
] as const;

export function ScalePage() {
  const activeSpaceId = getCurrentSpaceId();
  const [report, setReport] = useState<DoctorReport | null>(null);

  const readinessQuery = useQuery({
    queryKey: ["scale", "readiness", activeSpaceId],
    queryFn: getScaleReadiness,
  });

  const doctorMut = useMutation({
    mutationFn: async (suite: "TR3" | "M3") => {
      const { reportId } = await runDoctor(suite);
      return getDoctorReport(reportId);
    },
    onSuccess: setReport,
  });

  const r = readinessQuery.data;

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
        <table className="table">
          <tbody>
            <tr>
              <td>记忆 Schema</td>
              <td>v{r?.memorySchemaVersion ?? "-"}</td>
            </tr>
            <tr>
              <td>已批准记忆</td>
              <td>{r?.memoryApprovedCount ?? "-"}</td>
            </tr>
            <tr>
              <td>RAG 文档 / 分块</td>
              <td>
                {r ? `${r.ragDocumentCount} / ${r.ragChunkCount}` : "-"}
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
              <td>SQLite 路径</td>
              <td>
                <code>{r?.sqlitePath ?? "-"}</code>
              </td>
            </tr>
          </tbody>
        </table>
        <p className="muted-line">
          Postgres 迁移见 <code>doc/05-M3-多租户与Postgres演进.md</code>；CLI{" "}
          <code>ash migrate plan|copy|verify|sync</code>；脚本{" "}
          <code>bash scripts/migrate-postgres.sh</code>、<code>postgres-smoke.sh</code>。
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
