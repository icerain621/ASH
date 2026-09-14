import { useState } from "react";
import type { AgentSessionView } from "../api/session.api";
import { shortId } from "@/shared/utils/format";

type Props = {
  items: AgentSessionView[];
  selectedSessionId: string | null;
  loading?: boolean;
  onSelect: (session: AgentSessionView) => void;
  onNew: () => void;
  creating?: boolean;
  renamingId?: string | null;
  closingId?: string | null;
  onRename?: (sessionId: string, title: string) => void;
  onClose?: (sessionId: string) => void;
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

/** Left sidebar: real agent session history from listAgentSessions. */
export function SessionHistoryList({
  items,
  selectedSessionId,
  loading,
  onSelect,
  onNew,
  creating,
  renamingId,
  closingId,
  onRename,
  onClose,
}: Props) {
  const [editingId, setEditingId] = useState<string | null>(null);
  const [draft, setDraft] = useState("");

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

  return (
    <aside className="agent-session-history" data-testid="agent-session-history">
      <div className="agent-history-header">
        <h2>历史会话</h2>
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
      <ul className="agent-history-list">
        {items.map((item) => {
          const active = item.id === selectedSessionId;
          const editing = editingId === item.id;
          return (
            <li key={item.id} className="agent-history-row">
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
                  <button
                    type="button"
                    className="btn mini"
                    onClick={() => setEditingId(null)}
                  >
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
                      disabled={closingId === item.id || item.status === "closed"}
                      onClick={(e) => {
                        e.stopPropagation();
                        onClose?.(item.id);
                      }}
                    >
                      关闭
                    </button>
                  </div>
                </>
              )}
            </li>
          );
        })}
        {loading && !items.length ? <li className="muted-line">加载中…</li> : null}
        {!loading && !items.length ? (
          <li className="muted-line">暂无会话，点击新建开始对话</li>
        ) : null}
      </ul>
    </aside>
  );
}
