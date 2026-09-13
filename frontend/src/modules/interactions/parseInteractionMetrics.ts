/** Extract GV05–06 interaction / MemoryLink counters from Prometheus text. */

export type InteractionMetricSummary = {
  sealedTotal: number;
  replayMismatchTotal: number;
  memoryLinksByType: Record<string, number>;
  memoryLinkTotal: number;
};

const LINE_RE = /^(ash_interaction_thread_sealed_total|ash_interaction_replay_mismatch_total|ash_memory_link_total(?:\{[^}]*\})?)\s+([0-9.eE+-]+)\s*$/gm;

export function parseInteractionMetrics(text: string): InteractionMetricSummary {
  const out: InteractionMetricSummary = {
    sealedTotal: 0,
    replayMismatchTotal: 0,
    memoryLinksByType: {},
    memoryLinkTotal: 0,
  };
  if (!text) return out;
  let m: RegExpExecArray | null;
  LINE_RE.lastIndex = 0;
  while ((m = LINE_RE.exec(text)) !== null) {
    const name = m[1];
    const value = Number(m[2]);
    if (!Number.isFinite(value)) continue;
    if (name === "ash_interaction_thread_sealed_total") {
      out.sealedTotal += value;
      continue;
    }
    if (name === "ash_interaction_replay_mismatch_total") {
      out.replayMismatchTotal += value;
      continue;
    }
    if (name.startsWith("ash_memory_link_total")) {
      out.memoryLinkTotal += value;
      const typeMatch = /type="([^"]+)"/.exec(name);
      const linkType = typeMatch?.[1] ?? "unknown";
      out.memoryLinksByType[linkType] = (out.memoryLinksByType[linkType] ?? 0) + value;
    }
  }
  return out;
}
