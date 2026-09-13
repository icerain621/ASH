import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  getInteractionByRun,
  getInteractionThread,
  listInteractionMemoryLinks,
  replayInteractionThread,
  sealInteractionThread,
} from "../api/interactions.api";

type Props = {
  runId: string;
  highlightSeq?: number | null;
  onSelectLinkSeq?: (seq: number | null) => void;
};

/** Compact MemoryLink + seal/replay controls for Quest Diff side (GV05). */
export function MemoryLinkPanel({ runId, highlightSeq, onSelectLinkSeq }: Props) {
  const qc = useQueryClient();
  const byRun = useQuery({
    queryKey: ["interaction-by-run", runId],
    queryFn: () => getInteractionByRun(runId),
    enabled: Boolean(runId),
  });
  const threadId = byRun.data?.thread.id ?? "";
  const linksQuery = useQuery({
    queryKey: ["interaction-links", threadId],
    queryFn: () => listInteractionMemoryLinks(threadId),
    enabled: Boolean(threadId),
  });
  const foldQuery = useQuery({
    queryKey: ["interaction-fold", threadId],
    queryFn: () => getInteractionThread(threadId),
    enabled: Boolean(threadId),
  });

  const sealMut = useMutation({
    mutationFn: () => sealInteractionThread(threadId),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["interaction-by-run", runId] });
      void qc.invalidateQueries({ queryKey: ["interaction-fold", threadId] });
    },
  });
  const replayMut = useMutation({
    mutationFn: () => replayInteractionThread(threadId),
  });

  const sealed = byRun.data?.thread.status === "sealed";
  const replayOk = replayMut.data?.ok;
  const links = linksQuery.data?.items ?? [];

  return (
    <div className="pane" data-testid="memory-link-panel" style={{ marginTop: "0.75rem" }}>
      <div className="pane-title">
        <h2>MemoryLink</h2>
        <span>
          {sealed ? "sealed" : "open"}
          {foldQuery.data?.digest ? ` · ${foldQuery.data.digest.slice(0, 12)}` : ""}
        </span>
      </div>
      <div className="row-actions" style={{ marginBottom: "0.5rem" }}>
        <button
          type="button"
          className="btn mini"
          disabled={!threadId || sealMut.isPending || sealed}
          data-testid="memory-link-seal"
          onClick={() => sealMut.mutate()}
        >
          封印
        </button>
        <button
          type="button"
          className="btn mini"
          disabled={!threadId || replayMut.isPending}
          data-testid="memory-link-replay"
          onClick={() => replayMut.mutate()}
        >
          复现校验
        </button>
        {replayMut.isSuccess ? (
          <span
            className="muted-line"
            data-testid="memory-link-replay-status"
            data-ok={replayOk ? "1" : "0"}
          >
            {replayOk ? "digest 一致" : "digest 不一致"}
          </span>
        ) : null}
        {replayMut.isError ? (
          <span className="error-text" data-testid="memory-link-replay-error">
            {(replayMut.error as Error).message}
          </span>
        ) : null}
      </div>
      {links.length === 0 ? (
        <p className="muted-line" data-testid="memory-link-empty">
          本 Thread 暂无记忆关联边
        </p>
      ) : (
        <ul style={{ listStyle: "none", padding: 0, margin: 0 }} data-testid="memory-link-list">
          {links.map((l) => {
            const active = highlightSeq === l.eventSeq;
            return (
              <li key={l.id}>
                <button
                  type="button"
                  className="btn mini"
                  data-testid="memory-link-item"
                  data-active={active ? "1" : "0"}
                  style={{
                    display: "block",
                    width: "100%",
                    textAlign: "left",
                    marginBottom: 4,
                    outline: active ? "1px solid currentColor" : undefined,
                  }}
                  onClick={() => onSelectLinkSeq?.(active ? null : l.eventSeq)}
                >
                  <strong>{l.linkType}</strong> · {l.memoryId}
                  <span style={{ opacity: 0.6 }}> · seq {l.eventSeq}</span>
                </button>
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}
