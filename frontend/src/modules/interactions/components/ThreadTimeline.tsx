import { useQuery } from "@tanstack/react-query";
import { getInteractionByRun, getInteractionThread } from "../api/interactions.api";

type Props = {
  runId: string;
  highlightSeq?: number | null;
  onSelectSeq?: (seq: number | null) => void;
};

/** Folded Interaction Thread timeline with seal badge (GV05–06 Task 8). */
export function ThreadTimeline({ runId, highlightSeq, onSelectSeq }: Props) {
  const byRun = useQuery({
    queryKey: ["interaction-by-run", runId],
    queryFn: () => getInteractionByRun(runId),
    enabled: Boolean(runId),
  });
  const threadId = byRun.data?.thread.id ?? "";
  const foldQuery = useQuery({
    queryKey: ["interaction-fold", threadId],
    queryFn: () => getInteractionThread(threadId),
    enabled: Boolean(threadId),
  });

  const thread = byRun.data?.thread;
  const sealed = thread?.status === "sealed";
  const nodes = foldQuery.data?.nodes ?? [];
  const linkSeqs = new Set((foldQuery.data?.links ?? []).map((l) => l.eventSeq));

  return (
    <div className="pane" data-testid="thread-timeline" style={{ marginTop: "0.75rem" }}>
      <div className="pane-title">
        <h2>Thread 时间线</h2>
        <span data-testid="thread-timeline-badge" data-sealed={sealed ? "1" : "0"}>
          {sealed ? "sealed" : "open"}
          {threadId ? ` · ${threadId.slice(0, 10)}` : ""}
          {foldQuery.data?.digest ? ` · ${foldQuery.data.digest.slice(0, 12)}` : ""}
        </span>
      </div>
      {!runId ? (
        <p className="muted-line">选择 Run 后加载 Thread 折叠视图</p>
      ) : byRun.isLoading || foldQuery.isLoading ? (
        <p className="muted-line">加载中…</p>
      ) : nodes.length === 0 ? (
        <p className="muted-line" data-testid="thread-timeline-empty">
          暂无折叠节点
        </p>
      ) : (
        <ul style={{ listStyle: "none", padding: 0, margin: 0 }} data-testid="thread-timeline-list">
          {nodes.map((n) => {
            const active = highlightSeq === n.seq;
            const hasLink = linkSeqs.has(n.seq);
            return (
              <li key={n.id || `${n.seq}-${n.type}`}>
                <button
                  type="button"
                  className="btn mini"
                  data-testid="thread-timeline-node"
                  data-seq={String(n.seq)}
                  data-active={active ? "1" : "0"}
                  data-linked={hasLink ? "1" : "0"}
                  style={{
                    display: "block",
                    width: "100%",
                    textAlign: "left",
                    marginBottom: 4,
                    opacity: active ? 1 : hasLink ? 0.95 : 0.75,
                    outline: active ? "1px solid currentColor" : undefined,
                  }}
                  onClick={() => onSelectSeq?.(active ? null : n.seq)}
                >
                  <span style={{ opacity: 0.55 }}>#{n.seq}</span> {n.type}
                  <span style={{ opacity: 0.55 }}> · {n.visibility}</span>
                  {hasLink ? <span> · link</span> : null}
                </button>
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}
