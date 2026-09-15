import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo, useState } from "react";
import {
  closeAgentSession,
  createAgentSession,
  getAgentSession,
  listAgentSessions,
  listSessionEvents,
  patchAgentSession,
  submitSessionIntent,
  type AgentSessionView,
  type SessionEventEnvelope,
} from "../api/session.api";
import {
  createAgentWorkspace,
  listAgentWorkspaces,
} from "../api/workspace.api";
import { useRunStream, type StreamLine } from "@/services/sse/runStream";
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

function parseStreamPayload(raw: string): unknown {
  if (!raw) return undefined;
  try {
    return JSON.parse(raw) as unknown;
  } catch {
    return raw;
  }
}

function streamLinesToEvents(lines: StreamLine[], runId: string): SessionEventEnvelope[] {
  const out: SessionEventEnvelope[] = [];
  for (const line of lines) {
    if (!line.type || line.type === "sse" || line.type === "message") continue;
    const payload = parseStreamPayload(line.payload);
    let seq = 0;
    if (payload && typeof payload === "object" && !Array.isArray(payload)) {
      const s = (payload as { seq?: unknown }).seq;
      if (typeof s === "number") seq = s;
    }
    out.push({
      id: line.id,
      runId,
      seq,
      ts: Date.now(),
      type: line.type,
      severity: "info",
      visibility: line.type.startsWith("tool.") || line.type.startsWith("step.") ? "ui_only" : undefined,
      payload,
    });
  }
  return out;
}

function mergeEvents(
  base: SessionEventEnvelope[],
  streamed: SessionEventEnvelope[],
): SessionEventEnvelope[] {
  const byKey = new Map<string, SessionEventEnvelope>();
  for (const ev of base) {
    const key = ev.id || `${ev.seq}:${ev.type}`;
    byKey.set(key, ev);
  }
  for (const ev of streamed) {
    const key = ev.seq > 0 ? `seq:${ev.seq}` : ev.id || `${ev.type}:${JSON.stringify(ev.payload)}`;
    // Prefer seq-keyed merge when available
    if (ev.seq > 0) {
      let replaced = false;
      for (const [k, existing] of byKey) {
        if (existing.seq === ev.seq) {
          byKey.delete(k);
          byKey.set(`seq:${ev.seq}`, { ...existing, ...ev, id: existing.id || ev.id });
          replaced = true;
          break;
        }
      }
      if (!replaced) byKey.set(key, ev);
    } else if (!byKey.has(key)) {
      byKey.set(key, ev);
    }
  }
  return Array.from(byKey.values()).sort((a, b) => {
    if (a.seq && b.seq && a.seq !== b.seq) return a.seq - b.seq;
    return (a.ts || 0) - (b.ts || 0);
  });
}

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
  const [activeWorkspaceId, setActiveWorkspaceId] = useState<string | null>(null);

  const listQuery = useQuery({
    queryKey: ["agent-sessions"],
    queryFn: () => listAgentSessions({ limit: 50 }),
  });

  const workspaceQuery = useQuery({
    queryKey: ["agent-workspaces"],
    queryFn: () => listAgentWorkspaces({ limit: 50 }),
  });

  const sessionQuery = useQuery({
    queryKey: ["agent-session", selectedSessionId],
    queryFn: () => getAgentSession(selectedSessionId!),
    enabled: Boolean(selectedSessionId),
  });

  const activeSession: AgentSessionView | null =
    sessionQuery.data ??
    listQuery.data?.items?.find((s) => s.id === selectedSessionId) ??
    null;

  const runId = activeSession?.runId || "";
  const pollEvents = !runId;

  const eventsQuery = useQuery({
    queryKey: ["agent-session-events", selectedSessionId],
    queryFn: () => listSessionEvents(selectedSessionId!, { limit: 100 }),
    enabled: Boolean(selectedSessionId),
    refetchInterval: pollEvents ? 4000 : false,
  });

  const { lines: streamLines } = useRunStream(runId || null);

  useEffect(() => {
    if (!selectedSessionId || !runId || streamLines.length === 0) return;
    void qc.invalidateQueries({ queryKey: ["agent-session-events", selectedSessionId] });
  }, [streamLines.length, selectedSessionId, runId, qc]);

  useEffect(() => {
    const ws = activeSession?.workspaceId;
    if (ws) setActiveWorkspaceId(ws);
  }, [activeSession?.workspaceId]);

  const createMut = useMutation({
    mutationFn: () =>
      createAgentSession(activeWorkspaceId ? { workspaceId: activeWorkspaceId } : {}),
    onSuccess: (session) => {
      setError("");
      setSelection(null);
      setViewTab("chat");
      if (session.workspaceId) setActiveWorkspaceId(session.workspaceId);
      void qc.invalidateQueries({ queryKey: ["agent-sessions"] });
      void qc.invalidateQueries({ queryKey: ["agent-workspaces"] });
      onSelectSession(session);
    },
    onError: (e: Error) => setError(e.message),
  });

  const createWorkspaceMut = useMutation({
    mutationFn: () => createAgentWorkspace({ title: "新工作区" }),
    onSuccess: (ws) => {
      setError("");
      setActiveWorkspaceId(ws.id);
      void qc.invalidateQueries({ queryKey: ["agent-workspaces"] });
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
        command: payload.command,
        args: payload.args,
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

  const renameMut = useMutation({
    mutationFn: ({ sessionId, title }: { sessionId: string; title: string }) =>
      patchAgentSession(sessionId, { title }),
    onSuccess: (session) => {
      setError("");
      void qc.invalidateQueries({ queryKey: ["agent-sessions"] });
      void qc.invalidateQueries({ queryKey: ["agent-session", session.id] });
    },
    onError: (e: Error) => setError(e.message),
  });

  const closeMut = useMutation({
    mutationFn: (sessionId: string) => closeAgentSession(sessionId),
    onSuccess: (session) => {
      setError("");
      void qc.invalidateQueries({ queryKey: ["agent-sessions"] });
      if (selectedSessionId === session.id) {
        onSelectSession(null);
      }
    },
    onError: (e: Error) => setError(e.message),
  });

  const baseEvents: SessionEventEnvelope[] = useMemo(
    () => eventsQuery.data?.items ?? [],
    [eventsQuery.data?.items],
  );
  const streamedEvents = useMemo(
    () => (runId ? streamLinesToEvents(streamLines, runId) : []),
    [streamLines, runId],
  );
  const events = useMemo(
    () => mergeEvents(baseEvents, streamedEvents),
    [baseEvents, streamedEvents],
  );

  const mode = runStatus === "waiting_approval" ? "gate" : "prompt";
  const runBusy =
    runStatus === "running" || runStatus === "waiting_approval" || intentMut.isPending;
  const headerTitle = activeSession
    ? (activeSession.title || "").trim() ||
      (activeSession.goal || "").trim() ||
      shortId(activeSession.id)
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
        workspaces={workspaceQuery.data?.items ?? []}
        selectedSessionId={selectedSessionId}
        activeWorkspaceId={activeWorkspaceId}
        loading={listQuery.isLoading || workspaceQuery.isLoading}
        creating={createMut.isPending}
        creatingWorkspace={createWorkspaceMut.isPending}
        renamingId={renameMut.isPending ? renameMut.variables?.sessionId : null}
        closingId={closeMut.isPending ? closeMut.variables ?? null : null}
        onSelect={(session) => {
          setError("");
          if (session.workspaceId) setActiveWorkspaceId(session.workspaceId);
          onSelectSession(session);
        }}
        onSelectWorkspace={setActiveWorkspaceId}
        onNew={() => createMut.mutate()}
        onNewWorkspace={() => createWorkspaceMut.mutate()}
        onRename={(sessionId, title) => renameMut.mutate({ sessionId, title })}
        onClose={(sessionId) => closeMut.mutate(sessionId)}
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
              canStop={runBusy}
              gateReason={gateReason}
              session={activeSession}
              onIntent={(payload) => intentMut.mutate(payload)}
            />
          </>
        )}
      </div>

      <DetailsPane open={detailsOpen} selection={selection} />
    </div>
  );
}
