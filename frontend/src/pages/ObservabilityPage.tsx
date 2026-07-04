import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Activity, RefreshCcw, Search, ShieldAlert } from "lucide-react";
import { useState } from "react";
import {
  evaluateAlerts,
  getPrometheusText,
  getTrace,
  listAlertRules,
  listAlerts,
  putAlertRules,
  type AlertRule,
} from "@/modules/closure/api/closure.api";
import { getPluginHealth } from "@/modules/platform/api/platform.api";
import { getOtelStatus, getRagProfile } from "@/modules/observability/api/observability.api";
import { getScaleReadiness } from "@/modules/scale/api/scale.api";
import { getCurrentSpaceId } from "@/services/http/client";

const GOVERNANCE_METRIC_HINTS: Record<string, string> = {
  memory_unreviewed_backlog: "未评审记忆候选（status=candidate）总数",
  rag_fts_fallback_rate: "窗口内 RAG chunk 降级查询占比",
  plugin_export_failures: "窗口内 plugin.export_failed 审计事件数",
  run_failure_rate: "运行失败 / 取消占比",
  api_error_rate: "审计日志中含 failed 的事件占比",
  queue_backlog_minutes: "长时间未开始的运行步骤数",
  low_feedback_rate: "低分反馈占比",
  postgres_live_gate: "Postgres E2E 审计证据",
  execgo_live_gate: "ExecGo live smoke 审计证据",
};

export function ObservabilityPage() {
  const qc = useQueryClient();
  const activeSpaceId = getCurrentSpaceId();
  const [traceId, setTraceId] = useState("");
  const alertsQuery = useQuery({ queryKey: ["observability-alerts", "active"], queryFn: () => listAlerts({ status: "active", limit: 100 }) });
  const rulesQuery = useQuery({ queryKey: ["observability-rules"], queryFn: listAlertRules });
  const metricsQuery = useQuery({
    queryKey: ["prometheus-text", activeSpaceId],
    queryFn: () => getPrometheusText(activeSpaceId),
  });
  const otelQuery = useQuery({
    queryKey: ["otel-status"],
    queryFn: getOtelStatus,
  });
  const ragQuery = useQuery({
    queryKey: ["rag-profile", activeSpaceId],
    queryFn: getRagProfile,
  });
  const scaleQuery = useQuery({
    queryKey: ["scale", "readiness", activeSpaceId],
    queryFn: getScaleReadiness,
  });
  const pluginHealthQuery = useQuery({
    queryKey: ["plugin-health", activeSpaceId],
    queryFn: getPluginHealth,
  });
  const otelExporter = pluginHealthQuery.data?.items.find((p) => p.id === "ash-otel-exporter");
  const traceQuery = useQuery({
    queryKey: ["trace", traceId],
    queryFn: () => getTrace(traceId),
    enabled: Boolean(traceId),
  });
  const evaluateMut = useMutation({
    mutationFn: evaluateAlerts,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["observability-alerts"] });
      qc.invalidateQueries({ queryKey: ["prometheus-text"] });
    },
  });
  const rulesMut = useMutation({
    mutationFn: (items: AlertRule[]) => putAlertRules(items),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["observability-rules"] }),
  });

  function toggleRule(rule: AlertRule) {
    rulesMut.mutate([{ ...rule, enabled: !rule.enabled }]);
  }

  return (
    <section className="panel active">
      <div className="page-kicker">
        <ShieldAlert size={17} strokeWidth={1.8} />
        Observability
      </div>
      <div className="page-heading">
        <div>
          <h1>可观测与告警</h1>
          <p>查看站内告警、规则评估、trace 关联记录和当前空间的 Prometheus 指标快照。</p>
          <span className="scope-badge">Space: {activeSpaceId}</span>
        </div>
        <div className="toolbar metrics-toolbar">
          <button className="btn primary icon-btn" onClick={() => evaluateMut.mutate()} disabled={evaluateMut.isPending}>
            <Activity size={16} strokeWidth={1.8} />
            评估告警
          </button>
          <button
            className="btn icon-btn"
            onClick={() => {
              alertsQuery.refetch();
              rulesQuery.refetch();
              metricsQuery.refetch();
              otelQuery.refetch();
              ragQuery.refetch();
              scaleQuery.refetch();
              pluginHealthQuery.refetch();
            }}
          >
            <RefreshCcw size={16} strokeWidth={1.8} />
            刷新
          </button>
        </div>
      </div>

      {(alertsQuery.error || rulesQuery.error || metricsQuery.error || otelQuery.error || ragQuery.error || evaluateMut.error || rulesMut.error || traceQuery.error) && (
        <p className="error-text">
          {(alertsQuery.error || rulesQuery.error || metricsQuery.error || otelQuery.error || ragQuery.error || evaluateMut.error || rulesMut.error || traceQuery.error as Error)?.message}
        </p>
      )}

      <div className="split ops-split">
      <div className="pane">
        <div className="pane-title">
          <h2>OTel 导出</h2>
          <span className={"status-pill " + (otelQuery.data?.enabled ? "ok" : "idle")}>
            <span className="status-dot" />
            {otelQuery.data?.enabled ? "enabled" : "disabled"}
          </span>
        </div>
        <table className="table">
          <tbody>
            <tr>
              <td>Service</td>
              <td>{otelQuery.data?.serviceName ?? "-"}</td>
            </tr>
            <tr>
              <td>Endpoint</td>
              <td>{otelQuery.data?.endpoint || "-"}</td>
            </tr>
            <tr>
              <td>Insecure</td>
              <td>{otelQuery.data?.insecure ? "yes" : "no"}</td>
            </tr>
            <tr>
              <td>Exporter 健康</td>
              <td>
                {otelExporter
                  ? `错误 ${otelExporter.exportErrors} · 丢弃 ${otelExporter.dropCount}`
                  : pluginHealthQuery.isLoading
                    ? "加载中"
                    : "未注册"}
              </td>
            </tr>
            <tr>
              <td>后台告警</td>
              <td>{scaleQuery.data?.alertsEvalInterval || "未配置"}</td>
            </tr>
            <tr>
              <td>后台 TTL sweep</td>
              <td>{scaleQuery.data?.memoryTTLSweepInterval || "未配置"}</td>
            </tr>
            <tr>
              <td>指标 replay</td>
              <td>{scaleQuery.data?.metricsEventReplayEnabled ? "ASH_METRICS_EVENT_REPLAY=1" : "off"}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <div className="pane">
        <div className="pane-title">
          <h2>RAG 检索</h2>
          <span className={"status-pill " + (ragQuery.data?.ftsAvailable ? "ok" : "idle")}>
            <span className="status-dot" />
            {ragQuery.data?.defaultRetrievalMode ?? "-"}
          </span>
        </div>
        <table className="table">
          <tbody>
            <tr>
              <td>FTS</td>
              <td>
                {ragQuery.data?.ftsAvailable ? "可用" : "不可用"}
                {ragQuery.data?.ftsEngine ? ` (${ragQuery.data.ftsEngine})` : null}
              </td>
            </tr>
            <tr>
              <td>文档 / 分块</td>
              <td>
                {ragQuery.data ? `${ragQuery.data.documentCount} / ${ragQuery.data.chunkCount}` : "-"}
              </td>
            </tr>
            <tr>
              <td>历史降级</td>
              <td>{ragQuery.data?.fallbackQueryCount ?? "-"}</td>
            </tr>
            <tr>
              <td>方言</td>
              <td>{ragQuery.data?.databaseDialect ?? "-"}</td>
            </tr>
          </tbody>
        </table>
      </div>
      </div>

      <div className="pane">
        <div className="pane-title">
          <h2>治理指标告警</h2>
          <span>记忆 / RAG / 插件</span>
        </div>
        <table className="table">
          <thead>
            <tr>
              <th>metric</th>
              <th>说明</th>
              <th>threshold</th>
              <th>enabled</th>
            </tr>
          </thead>
          <tbody>
            {(rulesQuery.data?.items ?? [])
              .filter((rule) => rule.metric in GOVERNANCE_METRIC_HINTS)
              .map((rule) => (
                <tr key={`gov-${rule.id}`}>
                  <td>{rule.metric}</td>
                  <td title={rule.description}>{GOVERNANCE_METRIC_HINTS[rule.metric] ?? rule.description ?? "-"}</td>
                  <td>
                    {rule.condition} {rule.threshold}
                  </td>
                  <td>
                    <button
                      className={`btn mini ${rule.enabled ? "ok" : ""}`}
                      onClick={() => toggleRule(rule)}
                      disabled={rulesMut.isPending}
                    >
                      {rule.enabled ? "on" : "off"}
                    </button>
                  </td>
                </tr>
              ))}
          </tbody>
        </table>
      </div>

      <div className="split ops-split">
        <div className="pane">
          <div className="pane-title">
            <h2>Active Alerts</h2>
            <span>{alertsQuery.data?.items.length ?? 0} 条</span>
          </div>
          <table className="table">
            <thead>
              <tr>
                <th>rule</th>
                <th>severity</th>
                <th>target</th>
              </tr>
            </thead>
            <tbody>
              {(alertsQuery.data?.items ?? []).length === 0 && (
                <tr className="empty-row">
                  <td colSpan={3}>当前没有 active alert。</td>
                </tr>
              )}
              {(alertsQuery.data?.items ?? []).map((item) => (
                <tr key={item.id}>
                  <td>{item.ruleName || item.message}</td>
                  <td>
                    <StatusPill value={item.severity} />
                  </td>
                  <td>
                    {item.targetType}:{item.targetId}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        <div className="pane">
          <div className="pane-title">
            <h2>Alert Rules</h2>
            <span>{rulesQuery.data?.items.length ?? 0} 条</span>
          </div>
          <table className="table">
            <thead>
              <tr>
                <th>metric</th>
                <th>threshold</th>
                <th>enabled</th>
              </tr>
            </thead>
            <tbody>
              {(rulesQuery.data?.items ?? []).map((rule) => (
                <tr key={rule.id}>
                  <td>{rule.metric}</td>
                  <td>
                    {rule.condition} {rule.threshold}
                  </td>
                  <td>
                    <button className={`btn mini ${rule.enabled ? "ok" : ""}`} onClick={() => toggleRule(rule)} disabled={rulesMut.isPending}>
                      {rule.enabled ? "on" : "off"}
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      <div className="split ops-split">
        <div className="pane">
          <div className="pane-title">
            <h2>Trace 查询</h2>
            <span>{traceQuery.data?.runs.length ?? 0} runs</span>
          </div>
          <form
            className="inline-form"
            onSubmit={(event) => {
              event.preventDefault();
              const value = String(new FormData(event.currentTarget).get("traceId") || "");
              setTraceId(value.trim());
            }}
          >
            <input name="traceId" placeholder="trace_..." />
            <button className="btn icon-btn" type="submit">
              <Search size={16} strokeWidth={1.8} />
              查询
            </button>
          </form>
          <pre className="code-block tall">
            {traceQuery.data
              ? JSON.stringify(
                  {
                    runs: traceQuery.data.runs.length,
                    events: traceQuery.data.events.length,
                    toolCalls: traceQuery.data.toolCalls.length,
                    agentTasks: traceQuery.data.agentTasks.length,
                    auditLogs: traceQuery.data.auditLogs.length,
                  },
                  null,
                  2,
                )
              : "输入 traceId 后显示关联记录数量。"}
          </pre>
        </div>

        <div className="pane">
          <div className="pane-title">
            <h2>Prometheus</h2>
            <span>{metricsQuery.isFetching ? "loading" : "snapshot"}</span>
          </div>
          <pre className="code-block tall">{metricsQuery.data || "暂无指标文本。"}</pre>
        </div>
      </div>

      <div className="pane">
        <div className="pane-title">
          <h2>最近评估</h2>
          <span>{evaluateMut.data?.evaluatedAt ?? "idle"}</span>
        </div>
        <table className="table">
          <thead>
            <tr>
              <th>metric</th>
              <th>status</th>
              <th>value</th>
              <th>message</th>
            </tr>
          </thead>
          <tbody>
            {(evaluateMut.data?.results ?? []).length === 0 && (
              <tr className="empty-row">
                <td colSpan={4}>点击评估告警后显示规则结果。</td>
              </tr>
            )}
            {(evaluateMut.data?.results ?? []).map((item) => (
              <tr key={item.ruleId}>
                <td>{item.metric}</td>
                <td>
                  <StatusPill value={item.status} />
                </td>
                <td>{item.value.toFixed(3)}</td>
                <td>{item.message}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}

function StatusPill({ value }: { value: string }) {
  const lowered = value.toLowerCase();
  const tone = ["ok", "pass", "active"].includes(lowered) ? "ok" : ["alert", "critical", "error", "block"].includes(lowered) ? "err" : "idle";
  return (
    <span className={`status-pill ${tone}`}>
      <span className="status-dot" />
      {value || "unknown"}
    </span>
  );
}
