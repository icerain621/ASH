import { useEffect, useRef, useState } from "react";

export type StreamLine = { id: string; type: string; payload: string };

export type StreamStatus = "idle" | "open" | "reconnecting" | "closed";

export type UseRunStreamResult = {
  lines: StreamLine[];
  status: StreamStatus;
};

const STREAM_EVENT_TYPES = [
  "run.started",
  "run.resumed",
  "run.finished",
  "run.failed",
  "step.started",
  "step.finished",
  "tool.called",
  "tool.result",
  "run.checkpoint_saved",
  "policy.denied",
  "memory.candidate_created",
  "memory.review_requested",
  "memory.reviewed",
  "memory.deprecated",
] as const;

const BASE_BACKOFF_MS = 1000;
const MAX_BACKOFF_MS = 30000;

/** Exported for tests: exponential backoff with jitter cap. */
export function nextReconnectDelayMs(attempt: number): number {
  const exp = Math.min(MAX_BACKOFF_MS, BASE_BACKOFF_MS * 2 ** Math.max(0, attempt));
  return exp;
}

export function useRunStream(runId: string | null): UseRunStreamResult {
  const [lines, setLines] = useState<StreamLine[]>([]);
  const [status, setStatus] = useState<StreamStatus>("idle");
  const sourceRef = useRef<EventSource | null>(null);
  const lastEventIdRef = useRef<string>("");
  const attemptRef = useRef(0);
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const closedRef = useRef(false);

  useEffect(() => {
    if (!runId) {
      setLines([]);
      setStatus("idle");
      return;
    }

    closedRef.current = false;
    attemptRef.current = 0;
    lastEventIdRef.current = "";
    setLines([]);
    setStatus("open");

    const clearReconnectTimer = () => {
      if (reconnectTimerRef.current != null) {
        clearTimeout(reconnectTimerRef.current);
        reconnectTimerRef.current = null;
      }
    };

    const closeSource = () => {
      if (sourceRef.current) {
        sourceRef.current.close();
        sourceRef.current = null;
      }
    };

    const connect = () => {
      if (closedRef.current) return;
      closeSource();

      let url = `/api/v1/runs/${runId}/stream`;
      if (lastEventIdRef.current) {
        url += `?Last-Event-ID=${encodeURIComponent(lastEventIdRef.current)}`;
      }

      const es = new EventSource(url);
      sourceRef.current = es;

      const append = (type: string, raw: string, eventId?: string) => {
        if (eventId) {
          lastEventIdRef.current = eventId;
        }
        setLines((prev) => [...prev, { id: `${prev.length}-${type}`, type, payload: raw }]);
      };

      const onAny = (ev: MessageEvent) => {
        if (ev.lastEventId) {
          lastEventIdRef.current = ev.lastEventId;
        }
        append(ev.type || "message", ev.data, ev.lastEventId || undefined);
      };

      es.onopen = () => {
        attemptRef.current = 0;
        setStatus("open");
      };
      es.onmessage = onAny;
      for (const t of STREAM_EVENT_TYPES) {
        es.addEventListener(t, onAny as EventListener);
      }
      es.onerror = () => {
        closeSource();
        if (closedRef.current) return;
        const attempt = attemptRef.current;
        attemptRef.current = attempt + 1;
        setStatus("reconnecting");
        append("sse", `connection closed; reconnecting in ${nextReconnectDelayMs(attempt)}ms`);
        clearReconnectTimer();
        reconnectTimerRef.current = setTimeout(connect, nextReconnectDelayMs(attempt));
      };
    };

    connect();

    return () => {
      closedRef.current = true;
      clearReconnectTimer();
      closeSource();
      setStatus("closed");
    };
  }, [runId]);

  return { lines, status };
}
