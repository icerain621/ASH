import { useEffect, useRef, useState } from "react";
import { getRunTimeline, type TimelineItem } from "@/modules/runs/api/runs.api";

export type StreamLine = { id: string; type: string; payload: string };

export type StreamStatus = "idle" | "open" | "reconnecting" | "polling" | "closed";

export type UseRunStreamResult = {
  lines: StreamLine[];
  status: StreamStatus;
};

export type TimelinePollFn = (runId: string) => Promise<TimelineItem[]>;

export type UseRunStreamOptions = {
  /** After this many SSE failures, fall back to timeline polling. */
  maxReconnectAttempts?: number;
  pollIntervalMs?: number;
  pollTimeline?: TimelinePollFn;
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
export const DEFAULT_MAX_RECONNECT_ATTEMPTS = 5;
export const DEFAULT_POLL_INTERVAL_MS = 3000;

async function defaultPollTimeline(runId: string): Promise<TimelineItem[]> {
  const res = await getRunTimeline(runId);
  return res.items ?? [];
}

/** Exported for tests: exponential backoff with cap. */
export function nextReconnectDelayMs(attempt: number): number {
  return Math.min(MAX_BACKOFF_MS, BASE_BACKOFF_MS * 2 ** Math.max(0, attempt));
}

function timelineToLines(items: TimelineItem[], afterSeq: number): { lines: StreamLine[]; maxSeq: number } {
  let maxSeq = afterSeq;
  const lines: StreamLine[] = [];
  for (const item of items) {
    const seq = typeof item.seq === "number" ? item.seq : 0;
    if (seq <= afterSeq) continue;
    if (seq > maxSeq) maxSeq = seq;
    lines.push({
      id: `poll-${seq}-${item.type}`,
      type: item.type,
      payload: item.payload != null ? JSON.stringify(item.payload) : "",
    });
  }
  return { lines, maxSeq };
}

export function useRunStream(
  runId: string | null,
  options: UseRunStreamOptions = {},
): UseRunStreamResult {
  const [lines, setLines] = useState<StreamLine[]>([]);
  const [status, setStatus] = useState<StreamStatus>("idle");
  const sourceRef = useRef<EventSource | null>(null);
  const lastEventIdRef = useRef<string>("");
  const lastSeqRef = useRef(0);
  const attemptRef = useRef(0);
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const pollTimerRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const closedRef = useRef(false);
  const optionsRef = useRef(options);
  optionsRef.current = options;

  useEffect(() => {
    if (!runId) {
      setLines([]);
      setStatus("idle");
      return;
    }

    closedRef.current = false;
    attemptRef.current = 0;
    lastEventIdRef.current = "";
    lastSeqRef.current = 0;
    setLines([]);
    setStatus("open");

    const clearReconnectTimer = () => {
      if (reconnectTimerRef.current != null) {
        clearTimeout(reconnectTimerRef.current);
        reconnectTimerRef.current = null;
      }
    };

    const clearPollTimer = () => {
      if (pollTimerRef.current != null) {
        clearInterval(pollTimerRef.current);
        pollTimerRef.current = null;
      }
    };

    const closeSource = () => {
      if (sourceRef.current) {
        sourceRef.current.close();
        sourceRef.current = null;
      }
    };

    const append = (type: string, raw: string, eventId?: string) => {
      if (eventId) {
        lastEventIdRef.current = eventId;
      }
      setLines((prev) => [...prev, { id: `${prev.length}-${type}`, type, payload: raw }]);
    };

    const startPolling = () => {
      if (closedRef.current || pollTimerRef.current != null) return;
      closeSource();
      clearReconnectTimer();
      setStatus("polling");
      append("sse", "SSE unavailable; falling back to timeline polling");

      const opts = optionsRef.current;
      const pollIntervalMs = opts.pollIntervalMs ?? DEFAULT_POLL_INTERVAL_MS;
      const pollTimeline = opts.pollTimeline ?? defaultPollTimeline;

      const tick = async () => {
        if (closedRef.current) return;
        try {
          const items = await pollTimeline(runId);
          const { lines: next, maxSeq } = timelineToLines(items, lastSeqRef.current);
          if (next.length) {
            lastSeqRef.current = maxSeq;
            setLines((prev) => [...prev, ...next]);
          }
        } catch {
          // keep polling; transient API errors are expected during outages
        }
      };

      void tick();
      pollTimerRef.current = setInterval(() => {
        void tick();
      }, pollIntervalMs);
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

      const onAny = (ev: MessageEvent) => {
        if (ev.lastEventId) {
          lastEventIdRef.current = ev.lastEventId;
        }
        try {
          const parsed = JSON.parse(ev.data) as { seq?: number };
          if (typeof parsed?.seq === "number" && parsed.seq > lastSeqRef.current) {
            lastSeqRef.current = parsed.seq;
          }
        } catch {
          // non-JSON payloads are still shown
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
        const maxAttempts = optionsRef.current.maxReconnectAttempts ?? DEFAULT_MAX_RECONNECT_ATTEMPTS;
        if (attemptRef.current >= maxAttempts) {
          startPolling();
          return;
        }
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
      clearPollTimer();
      closeSource();
      setStatus("closed");
    };
  }, [runId]);

  return { lines, status };
}
