import type { InteractionMetricSummary } from "../parseInteractionMetrics";
import { parseInteractionMetrics } from "../parseInteractionMetrics";

type Props = {
  prometheusText?: string;
};

/** Observability summary for interaction seal / mismatch / MemoryLink derive counters. */
export function InteractionMetricsSummary({ prometheusText = "" }: Props) {
  const summary = parseInteractionMetrics(prometheusText);
  return <InteractionMetricsSummaryView summary={summary} />;
}

export function InteractionMetricsSummaryView({ summary }: { summary: InteractionMetricSummary }) {
  const types = Object.keys(summary.memoryLinksByType).sort();
  return (
    <div className="pane" data-testid="interaction-metrics-summary">
      <div className="pane-title">
        <h2>Interaction / MemoryLink</h2>
        <span>derive 摘要</span>
      </div>
      <table className="table">
        <tbody>
          <tr>
            <td>thread sealed</td>
            <td data-testid="ix-metric-sealed">{summary.sealedTotal}</td>
          </tr>
          <tr>
            <td>replay mismatch</td>
            <td data-testid="ix-metric-mismatch">{summary.replayMismatchTotal}</td>
          </tr>
          <tr>
            <td>memory links</td>
            <td data-testid="ix-metric-links">{summary.memoryLinkTotal}</td>
          </tr>
          {types.map((t) => (
            <tr key={t}>
              <td>link · {t}</td>
              <td data-testid={`ix-metric-link-${t}`}>{summary.memoryLinksByType[t]}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
