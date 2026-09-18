import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo, useRef, useState } from "react";
import {
  closeAgentSession,
  createAgentSession,
  getAgentSession,
  listAgentSessions,
  listSessionEvents,
  patchAgentSession,
  purgeAgentSession,
  submitSessionIntent,
  type AgentSessionView,
  type SessionEventEnvelope,
} from "../api/session.api";
import {
  attachAgentWorkspaceSession,
  closeAgentWorkspace,
  createAgentWorkspace,
  detachAgentWorkspaceSession,
  listAgentWorkspaces,
  patchAgentWorkspace,
} from "../api/workspace.api";
import { useSessionStream, type StreamLine } from "@/services/sse/runStream";
import { ChatComposer } from "./ChatComposer";
import { ChatTranscript, type ChatBubbleSelection } from "./ChatTranscript";
import { DetailsPane } from "./DetailsPane";
import { type IntentPayload } from "./IntentBar";
import { SessionHistoryList, type SessionMoveRequest } from "./SessionHistoryList";
import { TrajectoryPane } from "./TrajectoryPane";
import { deriveGateFromEvents } from "./deriveGateFromEvents";
import { useChatColumnWidths } from "./useChatColumnWidths";
import { shortId } from "@/shared/utils/format";
import {
  EMPTY_HERO_DISMISS_KEY,
  resolveAgentMode,
  type AgentMode,
} from "../agentModeLabels";
import { AgentEmptyHero } from "./AgentEmptyHero";
import { reorderSessionIds } from "./reorderSessionIds";

export type AgentChatShellProps = {
  selectedSessionId: string | null;
  onSelectSession: (session: AgentSessionView | null) => void;
  runStatus?: string;
  gateReason?: string;
  onIntentSuccess?: () => void;
  onOpenTools?: () => void;
  onOpenMcp?: () => void;
  onOpenSkills?: () => void;
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
  onOpenTools,
  onOpenMcp,
  onOpenSkills,
}: AgentChatShellProps) {
  const qc = useQueryClient();
  const [viewTab, setViewTab] = useState<CenterTab>("chat");
  const [detailsOpen, setDetailsOpen] = useState(true);
  const [selection, setSelection] = useState<ChatBubbleSelection | null>(null);
  const [highlightSeq, setHighlightSeq] = useState<number | null>(null);
  const [error, setError] = useState("");
  const [activeWorkspaceId, setActiveWorkspaceId] = useState<string | null>(null);
  const [includeClosed, setIncludeClosed] = useState(false);
  const [editingTitle, setEditingTitle] = useState(false);
  const [titleDraft, setTitleDraft] = useState("");
  const [showJumpBottom, setShowJumpBottom] = useState(false);
  const [localAgentMode, setLocalAgentMode] = useState<AgentMode>("coding");
  const [heroDismissed, setHeroDismissed] = useState(
    () => localStorage.getItem(EMPTY_HERO_DISMISS_KEY) === "1",
  );
  const { shellStyle, startResize } = useChatColumnWidths();

  const listQuery = useQuery({
    queryKey: ["agent-sessions", includeClosed],
    queryFn: () => listAgentSessions({ limit: 50, includeClosed }),
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
  const streamUrl =
    activeSession?.streamUrl ||
    (selectedSessionId ? `/api/v1/agents/sessions/${selectedSessionId}/stream` : "");
  // Prefer session-level SSE (covers blank + bound); keep light events poll as cold-start only.
  const pollEvents = Boolean(selectedSessionId) && !streamUrl;

  const eventsQuery = useQuery({
    queryKey: ["agent-session-events", selectedSessionId],
    queryFn: () => listSessionEvents(selectedSessionId!, { limit: 100 }),
    enabled: Boolean(selectedSessionId),
    refetchInterval: pollEvents ? 4000 : false,
  });

  const { lines: streamLines } = useSessionStream(
    selectedSessionId,
    activeSession?.streamUrl || streamUrl || null,
  );

  useEffect(() => {
    if (!selectedSessionId || streamLines.length === 0) return;
    void qc.invalidateQueries({ queryKey: ["agent-session-events", selectedSessionId] });
  }, [streamLines.length, selectedSessionId, qc]);

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

  const agentModeMut = useMutation({
    mutationFn: ({ sessionId, agentMode }: { sessionId: string; agentMode: AgentMode }) =>
      patchAgentSession(sessionId, { agentMode }),
    onSuccess: (session) => {
      setError("");
      void qc.invalidateQueries({ queryKey: ["agent-sessions"] });
      void qc.invalidateQueries({ queryKey: ["agent-session", session.id] });
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

  const purgeMut = useMutation({
    mutationFn: (sessionId: string) => purgeAgentSession(sessionId),
    onSuccess: (result) => {
      setError("");
      void qc.invalidateQueries({ queryKey: ["agent-sessions"] });
      void qc.removeQueries({ queryKey: ["agent-session", result.id] });
      if (selectedSessionId === result.id) {
        onSelectSession(null);
      }
    },
    onError: (e: Error) => setError(e.message),
  });

  const reorderMut = useMutation({
    mutationFn: ({ workspaceId, sessionIds }: { workspaceId: string; sessionIds: string[] }) =>
      patchAgentWorkspace(workspaceId, { sessionIds }),
    onSuccess: () => {
      setError("");
      void qc.invalidateQueries({ queryKey: ["agent-workspaces"] });
    },
    onError: (e: Error) => setError(e.message),
  });

  const moveMut = useMutation({
    mutationFn: async (req: SessionMoveRequest) => {
      const { sessionId, fromWorkspaceId, toWorkspaceId, beforeSessionId } = req;
      if (fromWorkspaceId && fromWorkspaceId !== toWorkspaceId) {
        await detachAgentWorkspaceSession(fromWorkspaceId, sessionId);
      }
      if (toWorkspaceId) {
        await attachAgentWorkspaceSession(toWorkspaceId, sessionId);
        const wsList = workspaceQuery.data?.items ?? [];
        const target = wsList.find((w) => w.id === toWorkspaceId);
        const base = (target?.sessionIds ?? []).filter((id) => id !== sessionId);
        let next = [...base, sessionId];
        if (beforeSessionId) {
          next = reorderSessionIds([...base, sessionId], sessionId, beforeSessionId);
        }
        await patchAgentWorkspace(toWorkspaceId, { sessionIds: next });
      }
    },
    onSuccess: () => {
      setError("");
      void qc.invalidateQueries({ queryKey: ["agent-workspaces"] });
      void qc.invalidateQueries({ queryKey: ["agent-sessions"] });
    },
    onError: (e: Error) => setError(e.message),
  });

  const renameWorkspaceMut = useMutation({
    mutationFn: ({ workspaceId, title }: { workspaceId: string; title: string }) =>
      patchAgentWorkspace(workspaceId, { title }),
    onSuccess: () => {
      setError("");
      void qc.invalidateQueries({ queryKey: ["agent-workspaces"] });
    },
    onError: (e: Error) => setError(e.message),
  });

  const closeWorkspaceMut = useMutation({
    mutationFn: (workspaceId: string) => closeAgentWorkspace(workspaceId),
    onSuccess: (ws) => {
      setError("");
      void qc.invalidateQueries({ queryKey: ["agent-workspaces"] });
      if (activeWorkspaceId === ws.id) setActiveWorkspaceId(null);
    },
    onError: (e: Error) => setError(e.message),
  });

  const baseEvents: SessionEventEnvelope[] = useMemo(
    () => eventsQuery.data?.items ?? [],
    [eventsQuery.data?.items],
  );
  const streamedEvents = useMemo(
    () =>
      streamLines.length
        ? streamLinesToEvents(streamLines, runId || selectedSessionId || "")
        : [],
    [streamLines, runId, selectedSessionId],
  );
  const events = useMemo(
    () => mergeEvents(baseEvents, streamedEvents),
    [baseEvents, streamedEvents],
  );

  const eventGate = useMemo(() => deriveGateFromEvents(events), [events]);
  const mode =
    runStatus === "waiting_approval" || eventGate.waiting ? "gate" : "prompt";
  const effectiveGateReason =
    (gateReason && gateReason.trim()) || eventGate.reason || "Run 等待审批";
  const runBusy =
    runStatus === "running" ||
    runStatus === "waiting_approval" ||
    eventGate.waiting ||
    intentMut.isPending;
  const headerTitle = activeSession
    ? (activeSession.title || "").trim() ||
      (activeSession.goal || "").trim() ||
      shortId(activeSession.id)
    : "选择或新建会话";

  const sessionAgentMode = resolveAgentMode(
    activeSession?.agentMode ??
      (typeof activeSession?.meta?.agentMode === "string"
        ? activeSession.meta.agentMode
        : undefined),
  );
  const effectiveAgentMode = selectedSessionId ? sessionAgentMode : localAgentMode;

  const transcriptHasContent = useMemo(() => {
    if (events.length > 0) return true;
    const turns = activeSession?.turns?.length ?? 0;
    const replies = activeSession?.replies?.length ?? 0;
    return turns + replies > 0;
  }, [events.length, activeSession?.turns, activeSession?.replies]);

  const showBrandHero = !heroDismissed && !transcriptHasContent;

  const handleAgentModeChange = (mode: AgentMode) => {
    if (selectedSessionId) {
      agentModeMut.mutate({ sessionId: selectedSessionId, agentMode: mode });
    } else {
      setLocalAgentMode(mode);
    }
  };

  const dismissBrandHero = () => {
    setHeroDismissed(true);
    localStorage.setItem(EMPTY_HERO_DISMISS_KEY, "1");
  };

  const scrollRef = useRef<HTMLDivElement>(null);
  const stickToBottomRef = useRef(true);

  useEffect(() => {
    setSelection(null);
    setHighlightSeq(null);
    setViewTab("chat");
    setEditingTitle(false);
    stickToBottomRef.current = true;
    setShowJumpBottom(false);
  }, [selectedSessionId]);

  useEffect(() => {
    const el = scrollRef.current;
    if (!el || !stickToBottomRef.current) return;
    el.scrollTop = el.scrollHeight;
  }, [events, viewTab]);

  const jumpToBottom = () => {
    stickToBottomRef.current = true;
    setShowJumpBottom(false);
    const el = scrollRef.current;
    if (el) el.scrollTop = el.scrollHeight;
  };

  const commitHeaderTitle = () => {
    if (!selectedSessionId) {
      setEditingTitle(false);
      return;
    }
    const next = titleDraft.trim();
    setEditingTitle(false);
    if (!next || next === headerTitle) return;
    renameMut.mutate({ sessionId: selectedSessionId, title: next });
  };

  return (
    <div
      className={`agent-chat-shell${detailsOpen ? "" : " details-collapsed"}`}
      data-testid="agent-chat-shell"
      style={shellStyle}
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
        purgingId={purgeMut.isPending ? purgeMut.variables ?? null : null}
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
        onPurge={(sessionId) => purgeMut.mutate(sessionId)}
        onReorderSessions={(workspaceId, sessionIds) =>
          reorderMut.mutate({ workspaceId, sessionIds })
        }
        onMoveSession={(req) => moveMut.mutate(req)}
        renamingWorkspaceId={
          renameWorkspaceMut.isPending ? renameWorkspaceMut.variables?.workspaceId : null
        }
        closingWorkspaceId={
          closeWorkspaceMut.isPending ? closeWorkspaceMut.variables ?? null : null
        }
        onRenameWorkspace={(workspaceId, title) =>
          renameWorkspaceMut.mutate({ workspaceId, title })
        }
        onCloseWorkspace={(workspaceId) => closeWorkspaceMut.mutate(workspaceId)}
        includeClosed={includeClosed}
        onIncludeClosedChange={setIncludeClosed}
        agentMode={effectiveAgentMode}
        agentModeBusy={agentModeMut.isPending}
        onAgentModeChange={handleAgentModeChange}
      />

      <div
        className="agent-chat-col-resizer"
        data-testid="agent-chat-resize-sidebar"
        role="separator"
        aria-orientation="vertical"
        aria-label="调整侧栏宽度"
        onMouseDown={(e) => {
          e.preventDefault();
          startResize("sidebar", e.clientX);
        }}
      />

      <div className="agent-chat-main">
        <header className="agent-chat-header">
          <div className="agent-chat-header-title">
            {editingTitle && selectedSessionId ? (
              <div className="agent-chat-header-rename" data-testid="agent-chat-header-rename">
                <input
                  value={titleDraft}
                  data-testid="agent-chat-header-rename-input"
                  autoFocus
                  onChange={(e) => setTitleDraft(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === "Enter") commitHeaderTitle();
                    if (e.key === "Escape") setEditingTitle(false);
                  }}
                  onBlur={commitHeaderTitle}
                />
              </div>
            ) : (
              <h2
                className={selectedSessionId ? "agent-chat-header-title-editable" : undefined}
                data-testid="agent-chat-header-title"
                title={selectedSessionId ? "点击改名" : undefined}
                onClick={() => {
                  if (!selectedSessionId) return;
                  setTitleDraft(headerTitle);
                  setEditingTitle(true);
                }}
              >
                {headerTitle}
              </h2>
            )}
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
          <div className="agent-chat-empty hero" data-testid="agent-chat-empty">
            {showBrandHero ? (
              <AgentEmptyHero agentMode={effectiveAgentMode} onDismiss={dismissBrandHero} />
            ) : (
              <>
                <p>选择左侧会话，或新建空白会话直接对话</p>
                <button
                  type="button"
                  className="btn mini"
                  disabled={createMut.isPending}
                  onClick={() => createMut.mutate()}
                >
                  新建会话
                </button>
              </>
            )}
          </div>
        ) : (
          <>
            <div className="agent-chat-scroll-wrap">
              <div
                className="agent-chat-scroll"
                data-testid="agent-chat-scroll"
                ref={scrollRef}
                onScroll={() => {
                  const el = scrollRef.current;
                  if (!el) return;
                  const dist = el.scrollHeight - el.scrollTop - el.clientHeight;
                  const pinned = dist < 80;
                  stickToBottomRef.current = pinned;
                  setShowJumpBottom(!pinned);
                }}
              >
                {viewTab === "chat" ? (
                  <ChatTranscript
                    events={events}
                    selectedId={selection?.id}
                    agentMode={effectiveAgentMode}
                    showBrandHero={showBrandHero}
                    onDismissBrandHero={dismissBrandHero}
                    onSelect={(node) => {
                      setSelection(node);
                      if (node.seq && node.seq > 0) setHighlightSeq(node.seq);
                    }}
                  />
                ) : (
                  <TrajectoryPane
                    events={events}
                    runId={runId || undefined}
                    highlightSeq={highlightSeq}
                    onSelectSeq={(seq) => {
                      setHighlightSeq(seq);
                    }}
                    onSelectEvent={(node) => {
                      setSelection(node);
                      if (node.seq && node.seq > 0) setHighlightSeq(node.seq);
                      setDetailsOpen(true);
                    }}
                  />
                )}
              </div>
              {showJumpBottom ? (
                <button
                  type="button"
                  className="agent-chat-jump-bottom"
                  data-testid="agent-chat-jump-bottom"
                  onClick={jumpToBottom}
                >
                  ↓ 回到底部
                </button>
              ) : null}
            </div>
            <ChatComposer
              mode={mode}
              busy={intentMut.isPending || !selectedSessionId}
              canStop={runBusy}
              gateReason={effectiveGateReason}
              session={activeSession}
              onIntent={(payload) => intentMut.mutate(payload)}
              onOpenTools={onOpenTools}
              onOpenMcp={onOpenMcp}
              onOpenSkills={onOpenSkills}
            />
          </>
        )}
      </div>

      {detailsOpen ? (
        <div
          className="agent-chat-col-resizer"
          data-testid="agent-chat-resize-details"
          role="separator"
          aria-orientation="vertical"
          aria-label="调整详情栏宽度"
          onMouseDown={(e) => {
            e.preventDefault();
            startResize("details", e.clientX);
          }}
        />
      ) : null}

      <DetailsPane open={detailsOpen} selection={selection} />
    </div>
  );
}
