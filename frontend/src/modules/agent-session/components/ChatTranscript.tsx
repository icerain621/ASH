import type { SessionEventEnvelope } from "../api/session.api";
import { BubbleMarkdown } from "./BubbleMarkdown";
import { ToolCallBubble } from "./ToolCallBubble";
import { mergeAssistantBubbles, type MergedChatBubble } from "./conversationNodes";

export type ChatBubbleSelection = {
  id: string;
  type: string;
  kind: string;
  title: string;
  summary: string;
  payload?: unknown;
  seq?: number;
  toolStatus?: MergedChatBubble["toolStatus"];
  toolName?: string;
  toolInput?: string;
  toolOutput?: string;
};

type Props = {
  events: SessionEventEnvelope[];
  selectedId?: string | null;
  onSelect?: (node: ChatBubbleSelection) => void;
};

function testIdForRole(role: MergedChatBubble["role"]): string {
  if (role === "tool") return "agent-chat-bubble-tool";
  if (role === "assistant") return "agent-chat-bubble-assistant";
  return "agent-chat-bubble";
}

/** Bubble transcript for Chat tab (DSH-aligned). Merges assistant.delta by turnId. */
export function ChatTranscript({ events, selectedId, onSelect }: Props) {
  const bubbles = mergeAssistantBubbles(events);

  const select = (item: MergedChatBubble) => {
    onSelect?.({
      id: item.id,
      type: item.type,
      kind: item.kind,
      title: item.title,
      summary: item.summary,
      payload: item.payload,
      seq: item.seq,
      toolStatus: item.toolStatus,
      toolName: item.toolName,
      toolInput: item.toolInput,
      toolOutput: item.toolOutput,
    });
  };

  return (
    <div className="agent-chat-transcript" data-testid="agent-chat-transcript">
      {bubbles.length === 0 ? (
        <p className="muted-line" data-testid="agent-chat-transcript-empty">
          暂无消息。在下方输入意图开始对话。
        </p>
      ) : (
        <ul className="agent-chat-bubbles">
          {bubbles.map((item) => {
            const active = selectedId === item.id;
            if (item.role === "tool") {
              return (
                <li key={item.id} className="agent-chat-bubble-row role-tool">
                  <ToolCallBubble item={item} active={active} onSelect={() => select(item)} />
                </li>
              );
            }
            return (
              <li key={item.id} className={`agent-chat-bubble-row role-${item.role}`}>
                <button
                  type="button"
                  className={`agent-chat-bubble role-${item.role}${active ? " active" : ""}${
                    item.streaming ? " streaming" : ""
                  }${item.stopped ? " stopped" : ""}`}
                  data-testid={testIdForRole(item.role)}
                  data-role={item.role}
                  data-node-kind={item.kind}
                  data-stopped={item.stopped ? "1" : "0"}
                  data-streaming={item.streaming ? "1" : "0"}
                  data-active={active ? "1" : "0"}
                  onClick={() => select(item)}
                >
                  <div className="agent-chat-bubble-meta">
                    <strong>{item.title}</strong>
                    {item.stopped ? (
                      <span className="agent-stopped-badge" data-testid="agent-chat-stopped-badge">
                        已停止
                      </span>
                    ) : null}
                    {item.role === "gate" ? (
                      <span className="muted agent-chat-bubble-type">{item.type}</span>
                    ) : null}
                  </div>
                  <div className="agent-chat-bubble-body">
                    {item.role === "assistant" || item.role === "user" ? (
                      <BubbleMarkdown text={item.summary} />
                    ) : (
                      item.summary
                    )}
                  </div>
                </button>
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}
