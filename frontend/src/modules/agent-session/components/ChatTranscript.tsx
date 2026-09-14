import type { SessionEventEnvelope } from "../api/session.api";
import { isThreadVisibleEvent, resolveConversationNode } from "./conversationNodes";

export type ChatBubbleSelection = {
  id: string;
  type: string;
  kind: string;
  title: string;
  summary: string;
  payload?: unknown;
};

type Props = {
  events: SessionEventEnvelope[];
  selectedId?: string | null;
  onSelect?: (node: ChatBubbleSelection) => void;
};

function bubbleRole(type: string, kind: string): "user" | "gate" | "tool" | "other" {
  if (type === "session.turn" || kind === "session.turn") return "user";
  if (type === "gate.waiting_approval" || kind === "gate.waiting_approval") return "gate";
  if (
    kind === "tool.called" ||
    kind === "tool.result" ||
    kind === "tool" ||
    kind === "step" ||
    type.startsWith("tool.") ||
    type.startsWith("step.")
  ) {
    return "tool";
  }
  return "other";
}

/** Bubble transcript for Chat tab (DSH-aligned). */
export function ChatTranscript({ events, selectedId, onSelect }: Props) {
  const visible = events.filter(isThreadVisibleEvent);

  return (
    <div className="agent-chat-transcript" data-testid="agent-chat-transcript">
      {visible.length === 0 ? (
        <p className="muted-line" data-testid="agent-chat-transcript-empty">
          暂无消息。在下方输入意图开始对话。
        </p>
      ) : (
        <ul className="agent-chat-bubbles">
          {visible.map((item) => {
            const node = resolveConversationNode(item);
            const role = bubbleRole(item.type, node.kind);
            const id = item.id || `${item.seq}-${item.type}`;
            const active = selectedId === id;
            return (
              <li key={id} className={`agent-chat-bubble-row role-${role}`}>
                <button
                  type="button"
                  className={`agent-chat-bubble role-${role}${active ? " active" : ""}`}
                  data-testid={role === "tool" ? "agent-chat-bubble-tool" : "agent-chat-bubble"}
                  data-role={role}
                  data-node-kind={node.kind}
                  data-active={active ? "1" : "0"}
                  onClick={() =>
                    onSelect?.({
                      id,
                      type: item.type,
                      kind: node.kind,
                      title: node.title,
                      summary: node.summary,
                      payload: item.payload,
                    })
                  }
                >
                  <div className="agent-chat-bubble-meta">
                    <strong>{node.title}</strong>
                    <span className="muted">{item.type}</span>
                  </div>
                  <div className="agent-chat-bubble-body">{node.summary}</div>
                </button>
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}
