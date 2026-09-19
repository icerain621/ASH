import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { getInteractionByRun, getInteractionThread } from "../api/interactions.api";

type Props = {
  runId: string;
  highlightSeq?: number | null;
  onSelectSeq?: (seq: number | null) => void;
};

type TimelineFilter = "all" | "model_visible" | "ui_only" | "tool" | "gate" | "hook" | "compact";

const FILTERS: { id: TimelineFilter; label: string }[] = [
  { id: "all", label: "全部" },
  { id: "model_visible", label: "模型可见" },
  { id: "ui_only", label: "界面" },
  { id: "tool", label: "工具" },
  { id: "gate", label: "门禁" },
  { id: "hook", label: "Hook" },
  { id: "compact", label: "Compact" },
];

function matchesFilter(type: string, visibility: string, filter: TimelineFilter) {
  if (filter === "all") return true;
  if (filter === "model_visible" || filter === "ui_only") return visibility === filter;
  if (filter === "tool") return type.startsWith("tool.");
  if (filter === "gate") return type.startsWith("gate.");
  if (filter === "hook") return type.startsWith("hook.");
  return type === "harness.compaction";
}

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
  const [filter, setFilter] = useState<TimelineFilter>("all");
  const nodes = (foldQuery.data?.nodes ?? []).filter((n) => matchesFilter(n.type, n.visibility, filter));
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
      <div className="toolbar" data-testid="thread-timeline-filters" style={{ marginBottom: "0.5rem", gap: "0.35rem" }}>
        {FILTERS.map((item) => (
          <button
            key={item.id}
            type="button"
            className={filter === item.id ? "btn mini primary" : "btn mini"}
            data-testid={`thread-timeline-filter-${item.id}`}
            aria-pressed={filter === item.id}
            onClick={() => setFilter(item.id)}
          >
            {item.label}
          </button>
        ))}
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
