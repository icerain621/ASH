import type { SessionEventEnvelope } from "../api/session.api";
import { ThreadTimeline } from "@/modules/interactions/components/ThreadTimeline";
import { resolveConversationNode } from "./conversationNodes";
import type { ChatBubbleSelection } from "./ChatTranscript";

type Props = {
  events: SessionEventEnvelope[];
  runId?: string;
  highlightSeq?: number | null;
  onSelectSeq?: (seq: number | null) => void;
  onSelectEvent?: (selection: ChatBubbleSelection) => void;
};

function selectionFromEvent(item: SessionEventEnvelope): ChatBubbleSelection {
  const node = resolveConversationNode(item);
  return {
    id: item.id || `${item.seq}-${item.type}`,
    type: item.type || "",
    kind: node.kind,
    title: node.title,
    summary: node.summary,
    payload: item.payload,
    seq: item.seq > 0 ? item.seq : undefined,
  };
}

/** Trajectory tab: ThreadTimeline when runId present, else flat session events. */
export function TrajectoryPane({
  events,
  runId,
  highlightSeq,
  onSelectSeq,
  onSelectEvent,
}: Props) {
  return (
    <div className="agent-chat-trajectory" data-testid="agent-chat-trajectory">
      {runId ? (
        <ThreadTimeline
          runId={runId}
          highlightSeq={highlightSeq}
          onSelectSeq={(seq) => {
            onSelectSeq?.(seq);
            if (seq != null && seq > 0) {
              const match = events.find((e) => e.seq === seq);
              if (match) onSelectEvent?.(selectionFromEvent(match));
            }
          }}
        />
      ) : (
        <ul className="agent-trajectory-flat" data-testid="agent-trajectory-flat">
          {events.length === 0 ? (
            <li className="muted-line">暂无事件（空白会话）</li>
          ) : (
            events.map((item) => {
              const node = resolveConversationNode(item);
              const active = highlightSeq != null && item.seq === highlightSeq;
              return (
                <li key={item.id || `${item.seq}-${item.type}`}>
                  <button
                    type="button"
                    className={`event-line agent-trajectory-row${active ? " active" : ""}`}
                    data-testid={`agent-trajectory-row-${item.seq || item.id}`}
                    data-active={active ? "1" : "0"}
                    onClick={() => {
                      if (item.seq > 0) onSelectSeq?.(item.seq);
                      onSelectEvent?.(selectionFromEvent(item));
                    }}
                  >
                    <strong className="type">{node.title}</strong>
                    <span className="muted" style={{ marginLeft: "0.5rem" }}>
                      {item.type}
                      {item.seq != null ? ` · #${item.seq}` : ""}
                    </span>
                    <div className="muted-line">{node.summary}</div>
                  </button>
                </li>
              );
            })
          )}
        </ul>
      )}
    </div>
  );
}
