import type { SessionEventEnvelope } from "../api/session.api";
import { ThreadTimeline } from "@/modules/interactions/components/ThreadTimeline";
import { resolveConversationNode } from "./conversationNodes";

type Props = {
  events: SessionEventEnvelope[];
  runId?: string;
  highlightSeq?: number | null;
  onSelectSeq?: (seq: number | null) => void;
};

/** Trajectory tab: ThreadTimeline when runId present, else flat session events. */
export function TrajectoryPane({ events, runId, highlightSeq, onSelectSeq }: Props) {
  return (
    <div className="agent-chat-trajectory" data-testid="agent-chat-trajectory">
      {runId ? (
        <ThreadTimeline runId={runId} highlightSeq={highlightSeq} onSelectSeq={onSelectSeq} />
      ) : (
        <ul className="agent-trajectory-flat" data-testid="agent-trajectory-flat">
          {events.length === 0 ? (
            <li className="muted-line">暂无事件（空白会话）</li>
          ) : (
            events.map((item) => {
              const node = resolveConversationNode(item);
              return (
                <li key={item.id || `${item.seq}-${item.type}`} className="event-line">
                  <strong className="type">{node.title}</strong>
                  <span className="muted" style={{ marginLeft: "0.5rem" }}>
                    {item.type}
                    {item.seq != null ? ` · #${item.seq}` : ""}
                  </span>
                  <div className="muted-line">{node.summary}</div>
                </li>
              );
            })
          )}
        </ul>
      )}
    </div>
  );
}
