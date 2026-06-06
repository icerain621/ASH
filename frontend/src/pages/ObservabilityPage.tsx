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

export function ObservabilityPage() {
  const qc = useQueryClient();
  const [traceId, setTraceId] = useState("");
  const alertsQuery = useQuery({ queryKey: ["observability-alerts", "active"], queryFn: () => listAlerts({ status: "active", limit: 100 }) });
  const rulesQuery = useQuery({ queryKey: ["observability-rules"], queryFn: listAlertRules });
  const metricsQuery = useQuery({ queryKey: ["prometheus-text"], queryFn: getPrometheusText });
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
          <p>查看站内告警、规则评估、trace 关联记录和 Prometheus 指标快照。</p>
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
            }}
          >
            <RefreshCcw size={16} strokeWidth={1.8} />
            刷新
          </button>
        </div>
      </div>

      {(alertsQuery.error || rulesQuery.error || metricsQuery.error || evaluateMut.error || rulesMut.error || traceQuery.error) && (
        <p className="error-text">
          {(alertsQuery.error || rulesQuery.error || metricsQuery.error || evaluateMut.error || rulesMut.error || traceQuery.error as Error)?.message}
        </p>
      )}

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
