import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo, useState } from "react";
import {
  createAgentSession,
  getAgentSession,
  listAgentSessions,
  listSessionEvents,
  submitSessionIntent,
  type AgentSessionView,
  type SessionEventEnvelope,
} from "../api/session.api";
import { ChatComposer } from "./ChatComposer";
import { ChatTranscript, type ChatBubbleSelection } from "./ChatTranscript";
import { DetailsPane } from "./DetailsPane";
import { type IntentPayload } from "./IntentBar";
import { SessionHistoryList } from "./SessionHistoryList";
import { TrajectoryPane } from "./TrajectoryPane";
import { shortId } from "@/shared/utils/format";

export type AgentChatShellProps = {
  selectedSessionId: string | null;
  onSelectSession: (session: AgentSessionView | null) => void;
  runStatus?: string;
  gateReason?: string;
  onIntentSuccess?: () => void;
};

type CenterTab = "chat" | "trajectory";

/** DSH-aligned three-column Agent Chat shell. */
export function AgentChatShell({
  selectedSessionId,
  onSelectSession,
  runStatus,
  gateReason,
  onIntentSuccess,
}: AgentChatShellProps) {
  const qc = useQueryClient();
  const [viewTab, setViewTab] = useState<CenterTab>("chat");
  const [detailsOpen, setDetailsOpen] = useState(true);
  const [selection, setSelection] = useState<ChatBubbleSelection | null>(null);
  const [highlightSeq, setHighlightSeq] = useState<number | null>(null);
  const [error, setError] = useState("");

  const listQuery = useQuery({
    queryKey: ["agent-sessions"],
    queryFn: () => listAgentSessions({ limit: 50 }),
  });

  const sessionQuery = useQuery({
    queryKey: ["agent-session", selectedSessionId],
    queryFn: () => getAgentSession(selectedSessionId!),
    enabled: Boolean(selectedSessionId),
  });

  const eventsQuery = useQuery({
    queryKey: ["agent-session-events", selectedSessionId],
    queryFn: () => listSessionEvents(selectedSessionId!, { limit: 100 }),
    enabled: Boolean(selectedSessionId),
    refetchInterval: 4000,
  });

  const createMut = useMutation({
    mutationFn: () => createAgentSession({}),
    onSuccess: (session) => {
      setError("");
      setSelection(null);
      setViewTab("chat");
      void qc.invalidateQueries({ queryKey: ["agent-sessions"] });
      onSelectSession(session);
    },
    onError: (e: Error) => setError(e.message),
  });

  const intentMut = useMutation({
    mutationFn: (payload: IntentPayload) => {
      if (!selectedSessionId) throw new Error("session not ready");
      return submitSessionIntent(selectedSessionId, {
        action: payload.action,
        prompt: payload.prompt,
        reason: payload.reason,
        actorId: "console",
      });
    },
    onSuccess: () => {
      setError("");
      void qc.invalidateQueries({ queryKey: ["agent-session-events", selectedSessionId] });
      void qc.invalidateQueries({ queryKey: ["agent-session", selectedSessionId] });
      void qc.invalidateQueries({ queryKey: ["agent-sessions"] });
      onIntentSuccess?.();
    },
    onError: (e: Error) => setError(e.message),
  });

  const events: SessionEventEnvelope[] = useMemo(
    () => eventsQuery.data?.items ?? [],
    [eventsQuery.data?.items],
  );

  const activeSession: AgentSessionView | null =
    sessionQuery.data ??
    listQuery.data?.items?.find((s) => s.id === selectedSessionId) ??
    null;

  const runId = activeSession?.runId || eventsQuery.data?.runId || "";
  const mode = runStatus === "waiting_approval" ? "gate" : "prompt";
  const headerTitle = activeSession
    ? (activeSession.goal || "").trim() || shortId(activeSession.id)
    : "选择或新建会话";

  useEffect(() => {
    setSelection(null);
    setHighlightSeq(null);
    setViewTab("chat");
  }, [selectedSessionId]);

  return (
    <div
      className={`agent-chat-shell${detailsOpen ? "" : " details-collapsed"}`}
      data-testid="agent-chat-shell"
    >
      <SessionHistoryList
        items={listQuery.data?.items ?? []}
        selectedSessionId={selectedSessionId}
        loading={listQuery.isLoading}
        creating={createMut.isPending}
        onSelect={(session) => {
          setError("");
          onSelectSession(session);
        }}
        onNew={() => createMut.mutate()}
      />

      <div className="agent-chat-main">
        <header className="agent-chat-header">
          <div className="agent-chat-header-title">
            <h2>{headerTitle}</h2>
            <span className="muted-line">
              {selectedSessionId ? shortId(selectedSessionId) : "—"}
              {runId ? ` · run ${shortId(runId)}` : ""}
              {activeSession?.status ? ` · ${activeSession.status}` : ""}
            </span>
          </div>
          <div className="agent-chat-header-actions">
            <div className="agent-chat-view-tabs" data-testid="agent-chat-view-tabs" role="tablist">
              <button
                type="button"
                role="tab"
                className={viewTab === "chat" ? "btn mini active" : "btn mini"}
                aria-selected={viewTab === "chat"}
                data-testid="agent-chat-tab-chat"
                onClick={() => setViewTab("chat")}
              >
                Chat
              </button>
              <button
                type="button"
                role="tab"
                className={viewTab === "trajectory" ? "btn mini active" : "btn mini"}
                aria-selected={viewTab === "trajectory"}
                data-testid="agent-chat-tab-trajectory"
                onClick={() => setViewTab("trajectory")}
              >
                Trajectory
              </button>
            </div>
            <button
              type="button"
              className="btn mini"
              data-testid="agent-chat-details-toggle"
              aria-pressed={detailsOpen}
              onClick={() => setDetailsOpen((v) => !v)}
            >
              {detailsOpen ? "隐藏 Details" : "显示 Details"}
            </button>
          </div>
        </header>

        {error ? (
          <p className="error-text" data-testid="agent-chat-error">
            {error}
          </p>
        ) : null}

        {!selectedSessionId ? (
          <div className="agent-chat-empty" data-testid="agent-chat-empty">
            <p>选择左侧会话，或新建空白会话直接对话</p>
            <button
              type="button"
              className="btn mini"
              disabled={createMut.isPending}
              onClick={() => createMut.mutate()}
            >
              新建会话
            </button>
          </div>
        ) : (
          <>
            <div className="agent-chat-scroll">
              {viewTab === "chat" ? (
                <ChatTranscript
                  events={events}
                  selectedId={selection?.id}
                  onSelect={setSelection}
                />
              ) : (
                <TrajectoryPane
                  events={events}
                  runId={runId || undefined}
                  highlightSeq={highlightSeq}
                  onSelectSeq={setHighlightSeq}
                />
              )}
            </div>
            <ChatComposer
              mode={mode}
              busy={intentMut.isPending || !selectedSessionId}
              gateReason={gateReason}
              onIntent={(payload) => intentMut.mutate(payload)}
            />
          </>
        )}
      </div>

      <DetailsPane open={detailsOpen} selection={selection} />
    </div>
  );
}
