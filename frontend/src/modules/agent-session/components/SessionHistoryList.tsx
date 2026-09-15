import { useMemo, useState } from "react";
import type { AgentSessionView } from "../api/session.api";
import type { AgentWorkspaceView } from "../api/workspace.api";
import { shortId } from "@/shared/utils/format";
import { reorderSessionIds } from "./reorderSessionIds";
import { groupSessionsByWorkspace } from "./workspaceGroups";

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
  /** Reorder sessions inside a workspace (PATCH sessionIds). */
  onReorderSessions?: (workspaceId: string, sessionIds: string[]) => void;
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
}: Props) {
  const [editingId, setEditingId] = useState<string | null>(null);
  const [draft, setDraft] = useState("");
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({});
  const [dragId, setDragId] = useState<string | null>(null);

  const groups = useMemo(
    () => groupSessionsByWorkspace(items, workspaces),
    [items, workspaces],
  );

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

  function toggleGroup(key: string) {
    setCollapsed((prev) => ({ ...prev, [key]: !prev[key] }));
  }

  function renderSessionRow(item: AgentSessionView, workspaceId: string | null) {
    const active = item.id === selectedSessionId;
    const editing = editingId === item.id;
    const canDrag = Boolean(workspaceId && onReorderSessions);
    return (
      <li
        key={item.id}
        className={`agent-history-row${dragId === item.id ? " dragging" : ""}`}
        draggable={canDrag && !editing}
        data-testid={`agent-history-row-${item.id}`}
        onDragStart={(e) => {
          if (!canDrag) return;
          setDragId(item.id);
          e.dataTransfer.setData("text/plain", item.id);
          e.dataTransfer.effectAllowed = "move";
        }}
        onDragEnd={() => setDragId(null)}
        onDragOver={(e) => {
          if (!canDrag || !dragId || dragId === item.id) return;
          e.preventDefault();
          e.dataTransfer.dropEffect = "move";
        }}
        onDrop={(e) => {
          e.preventDefault();
          if (!workspaceId || !onReorderSessions) return;
          const fromId = e.dataTransfer.getData("text/plain") || dragId;
          setDragId(null);
          if (!fromId || fromId === item.id) return;
          const group = groups.find((g) => g.workspace?.id === workspaceId);
          if (!group) return;
          const ids = group.sessions.map((s) => s.id);
          const next = reorderSessionIds(ids, fromId, item.id);
          if (next.join(",") === ids.join(",")) return;
          onReorderSessions(workspaceId, next);
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
      <div className="agent-workspace-list" data-testid="agent-workspace-list">
        {groups.map((group) => {
          const key = group.workspace?.id ?? "unassigned";
          const isOpen = !collapsed[key];
          const selectedWs = group.workspace
            ? activeWorkspaceId === group.workspace.id
            : activeWorkspaceId === null;
          return (
            <section
              key={key}
              className={selectedWs ? "agent-workspace-section active" : "agent-workspace-section"}
              data-testid={
                group.workspace ? `agent-workspace-${group.workspace.id}` : "agent-workspace-unassigned"
              }
            >
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
              {isOpen ? (
                <ul className="agent-history-list">
                  {group.sessions.map((item) =>
                    renderSessionRow(item, group.workspace?.id ?? null),
                  )}
                  {!group.sessions.length ? (
                    <li className="muted-line">暂无会话</li>
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
