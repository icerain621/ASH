import { useEffect, useRef, useState } from "react";

export type StreamLine = { id: string; type: string; payload: string };

export function useRunStream(runId: string | null) {
  const [lines, setLines] = useState<StreamLine[]>([]);
  const sourceRef = useRef<EventSource | null>(null);

  useEffect(() => {
    if (!runId) {
      setLines([]);
      return;
    }
    setLines([]);

    const url = `/api/v1/runs/${runId}/stream`;
    const es = new EventSource(url);
    sourceRef.current = es;

    const append = (type: string, raw: string) => {
      setLines((prev) => [...prev, { id: `${prev.length}-${type}`, type, payload: raw }]);
    };

    const onAny = (ev: MessageEvent) => append(ev.type || "message", ev.data);
    es.onmessage = onAny;
    for (const t of [
      "run.started", "run.resumed", "run.finished", "run.failed",
      "step.started", "step.finished", "tool.called", "tool.result",
      "run.checkpoint_saved", "policy.denied",
      "memory.candidate_created", "memory.review_requested", "memory.reviewed", "memory.deprecated",
    ]) {
      es.addEventListener(t, onAny);
    }
    es.onerror = () => append("sse", "connection closed or error");

    return () => {
      es.close();
      sourceRef.current = null;
    };
  }, [runId]);

  return lines;
}
