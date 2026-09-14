import type { AgentSessionView } from "../api/session.api";
import { shortId } from "@/shared/utils/format";

type Props = {
  items: AgentSessionView[];
  selectedSessionId: string | null;
  loading?: boolean;
  onSelect: (session: AgentSessionView) => void;
  onNew: () => void;
  creating?: boolean;
};

function sessionTitle(session: AgentSessionView): string {
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
}: Props) {
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
          return (
            <li key={item.id}>
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
