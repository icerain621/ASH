import type { SessionEventEnvelope } from "../api/session.api";
import { isThreadVisibleEvent, resolveConversationNode } from "./conversationNodes";

type Props = {
  events: SessionEventEnvelope[];
};

/** Renders thread as ConversationNode projections only. */
export function ConversationThread({ events }: Props) {
  const visible = events.filter(isThreadVisibleEvent);
  return (
    <ul className="event-log agent-session-thread" data-testid="agent-session-thread" style={{ listStyle: "none", padding: 0, margin: 0 }}>
      {visible.length === 0 ? (
        <li className="muted-line" data-testid="agent-session-thread-empty">
          暂无会话事件投影
        </li>
      ) : (
        visible.map((item) => {
          const node = resolveConversationNode(item);
          return (
            <li
              key={item.id || `${item.seq}-${item.type}`}
              className="event-line"
              data-testid="agent-conversation-node"
              data-node-kind={node.kind}
              data-visibility={node.visibility}
            >
              <div>
                <strong className="type">{node.title}</strong>
                <span className="muted" style={{ marginLeft: "0.5rem" }}>
                  {item.type}
                </span>
              </div>
              <div className="muted-line">{node.summary}</div>
            </li>
          );
        })
      )}
    </ul>
  );
}
