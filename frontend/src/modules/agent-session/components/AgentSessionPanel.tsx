import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useMemo, useState } from "react";
import {
  createAgentSession,
  listSessionEvents,
  submitSessionIntent,
  type SessionEventEnvelope,
} from "../api/session.api";
import { ConversationThread } from "./ConversationThread";
import { IntentBar, type IntentPayload } from "./IntentBar";

type Props = {
  runId: string;
  runStatus?: string;
  gateReason?: string;
  onIntentSuccess?: () => void;
};

/** Thin Agent Session panel: event projection + intent composer (GV02). */
export function AgentSessionPanel({ runId, runStatus, gateReason, onIntentSuccess }: Props) {
  const qc = useQueryClient();
  const [error, setError] = useState("");

  const sessionQuery = useQuery({
    queryKey: ["agent-session-bind", runId],
    queryFn: () => createAgentSession({ runId }),
    enabled: Boolean(runId),
    staleTime: 60_000,
  });

  const sessionId = sessionQuery.data?.id ?? "";

  const eventsQuery = useQuery({
    queryKey: ["agent-session-events", sessionId],
    enabled: Boolean(sessionId),
    queryFn: () => listSessionEvents(sessionId, { limit: 100 }),
    refetchInterval: 4000,
  });

  const intentMut = useMutation({
    mutationFn: (payload: IntentPayload) => {
      if (!sessionId) throw new Error("session not ready");
      return submitSessionIntent(sessionId, {
        action: payload.action,
        prompt: payload.prompt,
        reason: payload.reason,
        actorId: "console",
      });
    },
    onSuccess: () => {
      setError("");
      void qc.invalidateQueries({ queryKey: ["agent-session-events", sessionId] });
      onIntentSuccess?.();
    },
    onError: (e: Error) => setError(e.message),
  });

  const events: SessionEventEnvelope[] = useMemo(
    () => eventsQuery.data?.items ?? [],
    [eventsQuery.data?.items],
  );

  const mode = runStatus === "waiting_approval" ? "gate" : "prompt";
  const bindError = sessionQuery.error instanceof Error ? sessionQuery.error.message : "";

  return (
    <div className="pane" data-testid="agent-session-panel" style={{ marginBottom: "1rem" }}>
      <div className="pane-title">
        <h2>薄会话</h2>
        <span>{sessionId ? sessionId.slice(0, 12) : sessionQuery.isPending ? "绑定中…" : "—"}</span>
      </div>
      <p className="muted-line">仅渲染事件投影；底部只发意图（对齐 DSH composer）。</p>
      {bindError || error ? (
        <p className="error-text" data-testid="agent-session-error">
          {bindError || error}
        </p>
      ) : null}
      <ConversationThread events={events} />
      <div style={{ marginTop: "0.75rem" }}>
        <IntentBar
          mode={mode}
          busy={intentMut.isPending || !sessionId}
          gateReason={gateReason}
          onIntent={(payload) => intentMut.mutate(payload)}
        />
      </div>
    </div>
  );
}
