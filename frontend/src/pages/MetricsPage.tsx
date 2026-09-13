import { useQuery } from "@tanstack/react-query";
import { BarChart3, CalendarDays, RefreshCcw } from "lucide-react";
import { useMemo, useState } from "react";
import { getMetricsOverview, getSpaceEvaluation, type MetricCard, type MetricTrend } from "@/modules/metrics/api/metrics.api";
import { getCurrentSpaceId } from "@/services/http/client";

const KPI_ORDER = [
  "KPI-01",
  "KPI-02",
  "KPI-03",
  "KPI-04",
  "KPI-05",
  "KPI-06",
  "KPI-07",
  "KPI-08",
  "KPI-09",
  "KPI-10",
  "KPI-11",
  "KPI-17",
  "KPI-18",
  "KPI-19",
];

const EVOLVE_KPI = new Set(["KPI-17", "KPI-18", "KPI-19"]);

export function MetricsPage() {
  const activeSpaceId = getCurrentSpaceId();
  const [period, setPeriod] = useState<"day" | "week">("day");
  const [projectId, setProjectId] = useState("");
  const [days, setDays] = useState(7);
  const range = useMemo(() => {
    const to = new Date();
    const from = new Date(to.getTime() - days * 24 * 60 * 60 * 1000);
    return { from: from.toISOString(), to: to.toISOString() };
  }, [days]);

  const overviewQuery = useQuery({
    queryKey: ["metrics", "overview", activeSpaceId, period, projectId, days],
    queryFn: () => getMetricsOverview({ spaceId: activeSpaceId, period, projectId, ...range }),
  });
  const evaluationQuery = useQuery({
    queryKey: ["metrics", "evaluation", activeSpaceId, days],
    queryFn: () => getSpaceEvaluation(activeSpaceId, { from: range.from, to: range.to }),
  });
  const overview = overviewQuery.data;
  const evaluation = evaluationQuery.data;
  const cards = KPI_ORDER.map((id) => overview?.summary.find((item) => item.id === id)).filter(Boolean) as MetricCard[];
  const deliveryCards = cards.filter((c) => !EVOLVE_KPI.has(c.id));
  const evolveCards = cards.filter((c) => EVOLVE_KPI.has(c.id));

  return (
    <section className="panel active">
      <div className="page-kicker">
        <BarChart3 size={17} strokeWidth={1.8} />
        KPI 指标
      </div>
      <div className="page-heading">
        <div>
          <h1>指标看板</h1>
          <p>
            按 ASH KPI 口径查看交付、CI、反馈、记忆与场景可重复性（R-02）聚合结果；演进区含 KPI-17~19；空间测评为四维双核评分（GV06）。
          </p>
          <span className="scope-badge">Space: {activeSpaceId}</span>
        </div>
        <div className="toolbar metrics-toolbar">
          <label className="scenario-picker">
            <CalendarDays size={15} strokeWidth={1.8} />
            <select value={days} onChange={(e) => setDays(Number(e.target.value))}>
              <option value={7}>最近 7 天</option>
              <option value={14}>最近 14 天</option>
              <option value={30}>最近 30 天</option>
            </select>
          </label>
          <label className="scenario-picker">
            周期
            <select value={period} onChange={(e) => setPeriod(e.target.value as "day" | "week")}>
              <option value="day">按天</option>
              <option value="week">按周</option>
            </select>
          </label>
          <input
            className="metric-filter"
            value={projectId}
            placeholder="repo connection id"
            onChange={(e) => setProjectId(e.target.value)}
          />
          <button
            className="btn icon-btn"
            onClick={() => {
              overviewQuery.refetch();
              evaluationQuery.refetch();
            }}
            disabled={overviewQuery.isFetching || evaluationQuery.isFetching}
          >
            <RefreshCcw size={16} strokeWidth={1.8} />
            {overviewQuery.isFetching || evaluationQuery.isFetching ? "刷新中" : "刷新"}
          </button>
        </div>
      </div>

      {overviewQuery.isError && <p className="error-text">{(overviewQuery.error as Error).message}</p>}
      {evaluationQuery.isError && <p className="error-text">{(evaluationQuery.error as Error).message}</p>}

      <div className="metrics-evaluation" data-testid="metrics-evaluation-section">
        <div className="pane-title">
          <h2>空间测评（四维）</h2>
          <span>{evaluation?.dimensions?.length ?? 0} 维</span>
        </div>
        <div className="metrics-grid">
          {(evaluation?.dimensions ?? []).map((dim) => (
            <div className="pane metric-card" key={dim.id} data-testid={`eval-dim-${dim.id}`}>
              <div className="pane-title">
                <h2>{dim.label}</h2>
                <span className={`status-pill ${dim.status === "ok" ? "ok" : "idle"}`}>
                  <span className="status-dot" />
                  {dim.status}
                </span>
              </div>
              <div className="metric-value">{`${Math.round(dim.score * 1000) / 10}%`}</div>
              <p className="muted-line">
                {(dim.signals ?? [])
                  .slice(0, 2)
                  .map((s) => s.label)
                  .join(" · ") || "暂无信号"}
              </p>
            </div>
          ))}
          {!evaluationQuery.isLoading && !(evaluation?.dimensions?.length) ? (
            <p className="muted-line">暂无测评样本。</p>
          ) : null}
        </div>
        <div className="split metrics-split" style={{ marginTop: "0.75rem" }}>
          <div className="pane" data-testid="metrics-evaluation-health">
            <div className="pane-title">
              <h2>关联健康度</h2>
              <span>seal / replay / citation</span>
            </div>
            <table className="table compact">
              <tbody>
                <tr>
                  <td>Thread 封印率</td>
                  <td>
                    {evaluation
                      ? `${Math.round((evaluation.health.threadSealRate.value || 0) * 1000) / 10}% (${evaluation.health.threadSealRate.numerator ?? 0}/${evaluation.health.threadSealRate.denominator ?? 0})`
                      : "—"}
                  </td>
                </tr>
                <tr>
                  <td>Replay mismatch</td>
                  <td>
                    {evaluation
                      ? `${Math.round((evaluation.health.replayMismatchRate.value || 0) * 1000) / 10}%`
                      : "—"}
                  </td>
                </tr>
                <tr>
                  <td>缺引用事件</td>
                  <td>{evaluation?.health.citationMissingTotal ?? "—"}</td>
                </tr>
              </tbody>
            </table>
          </div>
          <div className="pane" data-testid="metrics-evaluation-scenarios">
            <div className="pane-title">
              <h2>场景排行</h2>
              <span>{evaluation?.scenarios?.length ?? 0} 组</span>
            </div>
            <table className="table compact">
              <thead>
                <tr>
                  <th>场景</th>
                  <th>样本</th>
                  <th>综合分</th>
                </tr>
              </thead>
              <tbody>
                {(evaluation?.scenarios ?? []).map((s) => (
                  <tr key={`${s.scenario}@${s.version ?? ""}`}>
                    <td>
                      {s.scenario}
                      {s.version ? `@${s.version}` : ""}
                    </td>
                    <td>{s.runCount}</td>
                    <td>{`${Math.round(s.rankScore * 1000) / 10}%`}</td>
                  </tr>
                ))}
                {!(evaluation?.scenarios?.length) ? (
                  <tr className="empty-row">
                    <td colSpan={3}>暂无场景样本。</td>
                  </tr>
                ) : null}
              </tbody>
            </table>
          </div>
        </div>
      </div>

      <div className="metrics-grid">
        {deliveryCards.map((card) => (
          <MetricSummaryCard key={card.id} card={card} />
        ))}
      </div>

      {evolveCards.length > 0 ? (
        <div className="metrics-evolve" data-testid="metrics-evolve-section">
          <div className="pane-title">
            <h2>演进 KPI（17–19）</h2>
            <span>{evolveCards.length} 项</span>
          </div>
          <div className="metrics-grid">
            {evolveCards.map((card) => (
              <MetricSummaryCard key={card.id} card={card} />
            ))}
          </div>
        </div>
      ) : null}

      <div className="split metrics-split">
        <div className="pane">
          <div className="pane-title">
            <h2>趋势</h2>
            <span>{overview?.period ?? period}</span>
          </div>
          <TrendTable trends={overview?.trends ?? []} />
        </div>
        <div className="pane">
          <div className="pane-title">
            <h2>数据质量</h2>
            <span>{overview?.generatedAt ? new Date(overview.generatedAt).toLocaleTimeString() : "pending"}</span>
          </div>
          <table className="table">
            <thead>
              <tr>
                <th>指标</th>
                <th>状态</th>
                <th>说明</th>
              </tr>
            </thead>
            <tbody>
              {(overview?.dataQuality ?? []).length === 0 && (
                <tr className="empty-row">
                  <td colSpan={3}>当前窗口没有数据质量告警。</td>
                </tr>
              )}
              {(overview?.dataQuality ?? []).map((item) => (
                <tr key={`${item.metricId}-${item.message}`}>
                  <td>{item.metricId}</td>
                  <td>
                    <span className={`status-pill ${item.status === "unavailable" ? "idle" : "ok"}`}>
                      <span className="status-dot" />
                      {item.status}
                    </span>
                  </td>
                  <td>{item.message}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      <div className="split metrics-split">
        {(overview?.breakdowns ?? []).map((group) => (
          <div className="pane" key={group.id} data-testid={`metrics-breakdown-${group.id}`}>
            <div className="pane-title">
              <h2>{group.label}</h2>
              <span>{group.items.length} 项</span>
            </div>
            <table className="table">
              <thead>
                <tr>
                  <th>分类</th>
                  <th>值</th>
                </tr>
              </thead>
              <tbody>
                {group.items.length === 0 && (
                  <tr className="empty-row">
                    <td colSpan={2}>暂无样本。</td>
                  </tr>
                )}
                {group.items.map((item) => {
                  const unstable =
                    group.id === "scenarioStability" &&
                    item.unit === "ratio" &&
                    !item.key.endsWith(":n") &&
                    !item.key.endsWith(":low") &&
                    item.value < 0.85;
                  return (
                    <tr key={item.key} className={unstable ? "error" : undefined}>
                      <td>{item.label}</td>
                      <td>
                        {formatNumber(item.value)} {item.unit}
                        {unstable ? " · 低于门槛" : ""}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        ))}
      </div>
    </section>
  );
}

function MetricSummaryCard({ card }: { card: MetricCard }) {
  return (
    <div className="pane metric-card">
      <div className="pane-title">
        <h2>{card.id}</h2>
        <span className={`status-pill ${card.status === "ok" ? "ok" : "idle"}`}>
          <span className="status-dot" />
          {card.status}
        </span>
      </div>
      <div className="metric-value">{formatMetricValue(card)}</div>
      <p className="tr2-card-title">{card.label}</p>
      <p className="muted-line">
        {card.denominator ? `${card.numerator ?? 0} / ${card.denominator}` : card.description || "暂无样本"}
      </p>
    </div>
  );
}

function TrendTable({ trends }: { trends: MetricTrend[] }) {
  const rows = trends.flatMap((trend) =>
    trend.points.map((point) => ({
      id: `${trend.metricId}-${point.periodStart}`,
      metricId: trend.metricId,
      periodStart: point.periodStart,
      value: point.value,
      status: point.status,
    })),
  );
  return (
    <table className="table">
      <thead>
        <tr>
          <th>指标</th>
          <th>周期</th>
          <th>值</th>
          <th>状态</th>
        </tr>
      </thead>
      <tbody>
        {rows.length === 0 && (
          <tr className="empty-row">
            <td colSpan={4}>暂无趋势样本。</td>
          </tr>
        )}
        {rows.map((row) => (
          <tr key={row.id}>
            <td>{row.metricId}</td>
            <td>{new Date(row.periodStart).toLocaleDateString()}</td>
            <td>{formatNumber(row.value)}</td>
            <td>{row.status}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

function formatMetricValue(card: MetricCard) {
  if (card.status === "unavailable") return "N/A";
  if (card.unit === "ratio") return `${Math.round(card.value * 1000) / 10}%`;
  if (card.unit === "ms") return `${formatNumber(card.value)} ms`;
  return formatNumber(card.value);
}

function formatNumber(value: number) {
  return new Intl.NumberFormat(undefined, { maximumFractionDigits: 2 }).format(value || 0);
}
