import { useQuery } from "@tanstack/react-query";
import { BarChart3, CalendarDays, RefreshCcw } from "lucide-react";
import { useMemo, useState } from "react";
import { getMetricsOverview, type MetricCard, type MetricTrend } from "@/modules/metrics/api/metrics.api";
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
];

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
  const overview = overviewQuery.data;
  const cards = KPI_ORDER.map((id) => overview?.summary.find((item) => item.id === id)).filter(Boolean) as MetricCard[];

  return (
    <section className="panel active">
      <div className="page-kicker">
        <BarChart3 size={17} strokeWidth={1.8} />
        KPI 指标
      </div>
      <div className="page-heading">
        <div>
          <h1>指标看板</h1>
          <p>按 ASH KPI 口径查看交付、CI、反馈、记忆与场景可重复性（R-02）聚合结果。</p>
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
          <button className="btn icon-btn" onClick={() => overviewQuery.refetch()} disabled={overviewQuery.isFetching}>
            <RefreshCcw size={16} strokeWidth={1.8} />
            {overviewQuery.isFetching ? "刷新中" : "刷新"}
          </button>
        </div>
      </div>

      {overviewQuery.isError && <p className="error-text">{(overviewQuery.error as Error).message}</p>}

      <div className="metrics-grid">
        {cards.map((card) => (
          <MetricSummaryCard key={card.id} card={card} />
        ))}
      </div>

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
