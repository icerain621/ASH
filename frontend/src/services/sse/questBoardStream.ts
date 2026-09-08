import { useEffect, useRef, useState } from "react";
import { getQuestBoard } from "@/modules/quest/api/quest.api";
import type { StreamStatus } from "@/services/sse/runStream";
import { DEFAULT_MAX_RECONNECT_ATTEMPTS, nextReconnectDelayMs } from "@/services/sse/runStream";

export const QUEST_BOARD_LIVE_TYPES = [
  "plan.created",
  "plan.approved",
  "plan.started",
  "plan.rejected",
  "run.started",
  "run.finished",
  "run.failed",
  "run.canceled",
  "gate.waiting_approval",
] as const;

export type UseQuestBoardStreamOptions = {
  enabled?: boolean;
  maxReconnectAttempts?: number;
  pollIntervalMs?: number;
  onBoardEvent?: (type: string) => void;
  pollBoard?: () => Promise<unknown>;
};

export type UseQuestBoardStreamResult = {
  status: StreamStatus;
};

const DEFAULT_POLL_MS = 3000;

/** Space-scoped Quest board SSE; invalidates / notifies on plan.* and key run.* events. */
export function useQuestBoardStream(
  spaceId: string | null,
  options: UseQuestBoardStreamOptions = {},
): UseQuestBoardStreamResult {
  const [status, setStatus] = useState<StreamStatus>("idle");
  const sourceRef = useRef<EventSource | null>(null);
  const lastEventIdRef = useRef("");
  const attemptRef = useRef(0);
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const pollTimerRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const closedRef = useRef(false);
  const optionsRef = useRef(options);
  optionsRef.current = options;

  const enabled = options.enabled !== false && !!spaceId;

  useEffect(() => {
    if (!enabled || !spaceId) {
      setStatus("idle");
      return;
    }

    closedRef.current = false;
    attemptRef.current = 0;
    lastEventIdRef.current = "";
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

    const notify = (type: string) => {
      optionsRef.current.onBoardEvent?.(type);
    };

    const startPolling = () => {
      if (closedRef.current || pollTimerRef.current != null) return;
      closeSource();
      clearReconnectTimer();
      setStatus("polling");
      const pollMs = optionsRef.current.pollIntervalMs ?? DEFAULT_POLL_MS;
      const pollBoard =
        optionsRef.current.pollBoard ?? (() => getQuestBoard(80));
      const tick = async () => {
        if (closedRef.current) return;
        try {
          await pollBoard();
          notify("poll.refresh");
        } catch {
          // keep polling
        }
      };
      void tick();
      pollTimerRef.current = setInterval(() => {
        void tick();
      }, pollMs);
    };

    const connect = () => {
      if (closedRef.current) return;
      closeSource();
      let url = "/api/v1/quest/stream";
      if (lastEventIdRef.current) {
        url += `?Last-Event-ID=${encodeURIComponent(lastEventIdRef.current)}`;
      }
      const es = new EventSource(url);
      sourceRef.current = es;

      const onAny = (ev: MessageEvent) => {
        if (ev.lastEventId) {
          lastEventIdRef.current = ev.lastEventId;
        }
        const type = ev.type || "message";
        if ((QUEST_BOARD_LIVE_TYPES as readonly string[]).includes(type)) {
          notify(type);
        }
      };

      es.onopen = () => {
        attemptRef.current = 0;
        setStatus("open");
      };
      es.onmessage = onAny;
      for (const t of QUEST_BOARD_LIVE_TYPES) {
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
  }, [enabled, spaceId]);

  return { status };
}
