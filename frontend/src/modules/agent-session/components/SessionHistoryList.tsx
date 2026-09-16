import { useMemo, useState } from "react";
import type { AgentSessionView } from "../api/session.api";
import type { AgentWorkspaceView } from "../api/workspace.api";
import { shortId } from "@/shared/utils/format";
import { reorderSessionIds } from "./reorderSessionIds";
import { groupSessionsByWorkspace } from "./workspaceGroups";

export type SessionMoveRequest = {
  sessionId: string;
  fromWorkspaceId: string | null;
  toWorkspaceId: string | null;
  /** Insert before this session within the target workspace; omit to append. */
  beforeSessionId?: string | null;
};

type Props = {
  items: AgentSessionView[];
  workspaces: AgentWorkspaceView[];
  selectedSessionId: string | null;
  activeWorkspaceId: string | null;
  loading?: boolean;
  onSelect: (session: AgentSessionView) => void;
  onSelectWorkspace: (workspaceId: string | null) => void;
  onNew: () => void;
  onNewWorkspace: () => void;
  creating?: boolean;
  creatingWorkspace?: boolean;
  renamingId?: string | null;
  closingId?: string | null;
  purgingId?: string | null;
  onRename?: (sessionId: string, title: string) => void;
  onClose?: (sessionId: string) => void;
  onPurge?: (sessionId: string) => void;
  /** Same-workspace reorder. */
  onReorderSessions?: (workspaceId: string, sessionIds: string[]) => void;
  /** Cross-workspace (or unassigned) move. */
  onMoveSession?: (req: SessionMoveRequest) => void;
  renamingWorkspaceId?: string | null;
  closingWorkspaceId?: string | null;
  onRenameWorkspace?: (workspaceId: string, title: string) => void;
  onCloseWorkspace?: (workspaceId: string) => void;
  includeClosed?: boolean;
  onIncludeClosedChange?: (include: boolean) => void;
};

function sessionTitle(session: AgentSessionView): string {
  const title = (session.title || "").trim();
  if (title) return title.length > 48 ? `${title.slice(0, 48)}…` : title;
  const goal = (session.goal || "").trim();
  if (goal) return goal.length > 48 ? `${goal.slice(0, 48)}…` : goal;
  const turn = session.turns?.[session.turns.length - 1]?.prompt?.trim();
  if (turn) return turn.length > 48 ? `${turn.slice(0, 48)}…` : turn;
  return shortId(session.id);
}

/** Left sidebar: sessions grouped under agent workspaces. */
export function SessionHistoryList({
  items,
  workspaces,
  selectedSessionId,
  activeWorkspaceId,
  loading,
  onSelect,
  onSelectWorkspace,
  onNew,
  onNewWorkspace,
  creating,
  creatingWorkspace,
  renamingId,
  closingId,
  purgingId,
  onRename,
  onClose,
  onPurge,
  onReorderSessions,
  onMoveSession,
  renamingWorkspaceId,
  closingWorkspaceId,
  onRenameWorkspace,
  onCloseWorkspace,
  includeClosed = false,
  onIncludeClosedChange,
}: Props) {
  const [editingId, setEditingId] = useState<string | null>(null);
  const [draft, setDraft] = useState("");
  const [editingWsId, setEditingWsId] = useState<string | null>(null);
  const [wsDraft, setWsDraft] = useState("");
  const [query, setQuery] = useState("");
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({});
  const [dragId, setDragId] = useState<string | null>(null);
  const [dragFromWs, setDragFromWs] = useState<string | null>(null);

  const filteredItems = useMemo(() => {
    const needle = query.trim().toLowerCase();
    if (!needle) return items;
    return items.filter((s) => {
      const title = (s.title || s.goal || "").toLowerCase();
      return s.id.toLowerCase().includes(needle) || title.includes(needle);
    });
  }, [items, query]);

  const groups = useMemo(
    () => groupSessionsByWorkspace(filteredItems, workspaces),
    [filteredItems, workspaces],
  );

  const canDragAny = Boolean(onReorderSessions || onMoveSession);

  function startRename(session: AgentSessionView) {
    setEditingId(session.id);
    setDraft(sessionTitle(session));
  }

  function commitRename(sessionId: string) {
    const next = draft.trim();
    setEditingId(null);
    if (!next || !onRename) return;
    onRename(sessionId, next);
  }

  function commitWorkspaceRename(workspaceId: string) {
    const next = wsDraft.trim();
    setEditingWsId(null);
    if (!next || !onRenameWorkspace) return;
    onRenameWorkspace(workspaceId, next);
  }

  function toggleGroup(key: string) {
    setCollapsed((prev) => ({ ...prev, [key]: !prev[key] }));
  }

  function handleDropOnSession(targetWsId: string | null, targetSessionId: string) {
    const fromId = dragId;
    const fromWs = dragFromWs;
    setDragId(null);
    setDragFromWs(null);
    if (!fromId || fromId === targetSessionId) return;

    if (fromWs && targetWsId && fromWs === targetWsId && onReorderSessions) {
      const group = groups.find((g) => g.workspace?.id === targetWsId);
      if (!group) return;
      const ids = group.sessions.map((s) => s.id);
      const next = reorderSessionIds(ids, fromId, targetSessionId);
      if (next.join(",") === ids.join(",")) return;
      onReorderSessions(targetWsId, next);
      return;
    }

    onMoveSession?.({
      sessionId: fromId,
      fromWorkspaceId: fromWs,
      toWorkspaceId: targetWsId,
      beforeSessionId: targetSessionId,
    });
  }

  function handleDropOnGroup(targetWsId: string | null) {
    const fromId = dragId;
    const fromWs = dragFromWs;
    setDragId(null);
    setDragFromWs(null);
    if (!fromId) return;
    if (fromWs === targetWsId) return;
    onMoveSession?.({
      sessionId: fromId,
      fromWorkspaceId: fromWs,
      toWorkspaceId: targetWsId,
      beforeSessionId: null,
    });
  }

  function renderSessionRow(item: AgentSessionView, workspaceId: string | null) {
    const active = item.id === selectedSessionId;
    const editing = editingId === item.id;
    return (
      <li
        key={item.id}
        className={`agent-history-row${dragId === item.id ? " dragging" : ""}`}
        draggable={canDragAny && !editing}
        data-testid={`agent-history-row-${item.id}`}
        onDragStart={(e) => {
          if (!canDragAny) return;
          setDragId(item.id);
          setDragFromWs(workspaceId);
          e.dataTransfer.setData("text/plain", item.id);
          e.dataTransfer.effectAllowed = "move";
        }}
        onDragEnd={() => {
          setDragId(null);
          setDragFromWs(null);
        }}
        onDragOver={(e) => {
          if (!dragId || dragId === item.id) return;
          e.preventDefault();
          e.dataTransfer.dropEffect = "move";
        }}
        onDrop={(e) => {
          e.preventDefault();
          e.stopPropagation();
          handleDropOnSession(workspaceId, item.id);
        }}
      >
        {editing ? (
          <div className="agent-history-rename" data-testid={`agent-history-rename-${item.id}`}>
            <input
              value={draft}
              data-testid={`agent-history-rename-input-${item.id}`}
              onChange={(e) => setDraft(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") commitRename(item.id);
                if (e.key === "Escape") setEditingId(null);
              }}
              autoFocus
            />
            <button
              type="button"
              className="btn mini ok"
              disabled={renamingId === item.id || !draft.trim()}
              data-testid={`agent-history-rename-save-${item.id}`}
              onClick={() => commitRename(item.id)}
            >
              保存
            </button>
            <button type="button" className="btn mini" onClick={() => setEditingId(null)}>
              取消
            </button>
          </div>
        ) : (
          <>
            <button
              type="button"
              className={active ? "agent-history-item active" : "agent-history-item"}
              onClick={() => onSelect(item)}
              data-testid={`agent-history-item-${item.id}`}
            >
              <strong>{sessionTitle(item)}</strong>
              <span className="muted-line">
                {item.status}
                {item.runId ? ` · ${shortId(item.runId)}` : " · blank"}
              </span>
            </button>
            <div className="agent-history-actions">
              <button
                type="button"
                className="btn mini"
                data-testid={`agent-history-rename-btn-${item.id}`}
                disabled={renamingId === item.id}
                onClick={(e) => {
                  e.stopPropagation();
                  startRename(item);
                }}
              >
                改名
              </button>
              <button
                type="button"
                className="btn mini err"
                data-testid={`agent-history-close-${item.id}`}
                disabled={closingId === item.id || purgingId === item.id || item.status === "closed"}
                onClick={(e) => {
                  e.stopPropagation();
                  onClose?.(item.id);
                }}
              >
                关闭
              </button>
              <button
                type="button"
                className="btn mini err"
                data-testid={`agent-history-purge-${item.id}`}
                disabled={purgingId === item.id || closingId === item.id}
                title="彻底删除（不可恢复）"
                onClick={(e) => {
                  e.stopPropagation();
                  if (
                    !window.confirm(
                      "彻底删除该会话？此操作不可恢复（仅移除会话记录，不删 run 事件）。",
                    )
                  ) {
                    return;
                  }
                  onPurge?.(item.id);
                }}
              >
                删除
              </button>
            </div>
          </>
        )}
      </li>
    );
  }

  return (
    <aside className="agent-session-history" data-testid="agent-session-history">
      <div className="agent-history-header">
        <h2>历史会话</h2>
        <div className="agent-history-header-actions">
          <button
            type="button"
            className="btn mini"
            onClick={onNewWorkspace}
            disabled={creatingWorkspace}
            data-testid="agent-workspace-new"
          >
            新建工作区
          </button>
          <button
            type="button"
            className="btn mini"
            onClick={onNew}
            disabled={creating}
            data-testid="agent-history-new"
          >
            新建
          </button>
        </div>
      </div>
      <label className="agent-history-search wide-field">
        <span className="sr-only">搜索会话</span>
        <input
          value={query}
          placeholder="搜索会话标题 / ID"
          data-testid="agent-history-search"
          onChange={(e) => setQuery(e.target.value)}
        />
      </label>
      {onIncludeClosedChange ? (
        <label className="agent-history-closed-toggle" data-testid="agent-history-include-closed">
          <input
            type="checkbox"
            checked={includeClosed}
            onChange={(e) => onIncludeClosedChange(e.target.checked)}
          />
          显示已关闭
        </label>
      ) : null}
      <div className="agent-workspace-list" data-testid="agent-workspace-list">
        {groups.map((group) => {
          const key = group.workspace?.id ?? "unassigned";
          const wsId = group.workspace?.id ?? null;
          const isOpen = !collapsed[key];
          const selectedWs = group.workspace
            ? activeWorkspaceId === group.workspace.id
            : activeWorkspaceId === null;
          const editingWs = Boolean(group.workspace && editingWsId === group.workspace.id);
          return (
            <section
              key={key}
              className={selectedWs ? "agent-workspace-section active" : "agent-workspace-section"}
              data-testid={
                group.workspace ? `agent-workspace-${group.workspace.id}` : "agent-workspace-unassigned"
              }
              onDragOver={(e) => {
                if (!dragId || !onMoveSession) return;
                e.preventDefault();
                e.dataTransfer.dropEffect = "move";
              }}
              onDrop={(e) => {
                e.preventDefault();
                handleDropOnGroup(wsId);
              }}
            >
              <div className="agent-workspace-head">
                {editingWs && group.workspace ? (
                  <div
                    className="agent-history-rename"
                    data-testid={`agent-workspace-rename-${group.workspace.id}`}
                  >
                    <input
                      value={wsDraft}
                      data-testid={`agent-workspace-rename-input-${group.workspace.id}`}
                      onChange={(e) => setWsDraft(e.target.value)}
                      onKeyDown={(e) => {
                        if (e.key === "Enter") commitWorkspaceRename(group.workspace!.id);
                        if (e.key === "Escape") setEditingWsId(null);
                      }}
                      autoFocus
                    />
                    <button
                      type="button"
                      className="btn mini ok"
                      disabled={
                        renamingWorkspaceId === group.workspace.id || !wsDraft.trim()
                      }
                      data-testid={`agent-workspace-rename-save-${group.workspace.id}`}
                      onClick={() => commitWorkspaceRename(group.workspace!.id)}
                    >
                      保存
                    </button>
                    <button type="button" className="btn mini" onClick={() => setEditingWsId(null)}>
                      取消
                    </button>
                  </div>
                ) : (
                  <>
                    <button
                      type="button"
                      className="agent-workspace-toggle"
                      aria-expanded={isOpen}
                      onClick={() => {
                        toggleGroup(key);
                        onSelectWorkspace(group.workspace?.id ?? null);
                      }}
                    >
                      <span className="agent-workspace-chevron">{isOpen ? "▾" : "▸"}</span>
                      <strong>{group.workspace?.title ?? "未分组"}</strong>
                      <span className="muted-line">{group.sessions.length}</span>
                    </button>
                    {group.workspace ? (
                      <div className="agent-workspace-actions">
                        <button
                          type="button"
                          className="btn mini"
                          data-testid={`agent-workspace-rename-btn-${group.workspace.id}`}
                          disabled={renamingWorkspaceId === group.workspace.id}
                          onClick={(e) => {
                            e.stopPropagation();
                            setEditingWsId(group.workspace!.id);
                            setWsDraft(group.workspace!.title || "工作区");
                          }}
                        >
                          改名
                        </button>
                        <button
                          type="button"
                          className="btn mini err"
                          data-testid={`agent-workspace-close-${group.workspace.id}`}
                          disabled={
                            closingWorkspaceId === group.workspace.id ||
                            group.workspace.status === "closed"
                          }
                          onClick={(e) => {
                            e.stopPropagation();
                            if (
                              !window.confirm(
                                `关闭工作区「${group.workspace!.title}」？会话不会删除。`,
                              )
                            ) {
                              return;
                            }
                            onCloseWorkspace?.(group.workspace!.id);
                          }}
                        >
                          关闭
                        </button>
                      </div>
                    ) : null}
                  </>
                )}
              </div>
              {isOpen ? (
                <ul className="agent-history-list">
                  {group.sessions.map((item) => renderSessionRow(item, wsId))}
                  {!group.sessions.length ? (
                    <li className="muted-line">
                      {query.trim() ? "无匹配会话" : "暂无会话 · 可拖入"}
                    </li>
                  ) : null}
                </ul>
              ) : null}
            </section>
          );
        })}
        {loading && !items.length && !workspaces.length ? (
          <p className="muted-line">加载中…</p>
        ) : null}
        {!loading && !items.length && !workspaces.length ? (
          <p className="muted-line">暂无会话，点击新建开始对话</p>
        ) : null}
      </div>
    </aside>
  );
}
