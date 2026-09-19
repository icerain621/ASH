import type { SessionEventEnvelope } from "../api/session.api";
import { ThreadTimeline } from "@/modules/interactions/components/ThreadTimeline";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  getRunTree,
  spawnSubRun,
  type RunSummary,
  type RunTreeNode,
} from "@/modules/runs/api/runs.api";
import { resolveConversationNode } from "./conversationNodes";
import type { ChatBubbleSelection } from "./ChatTranscript";
import { useState } from "react";

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

function flattenTree(node: RunTreeNode | undefined, out: RunSummary[] = []): RunSummary[] {
  if (!node?.summary?.runId) return out;
  out.push(node.summary);
  for (const child of node.children ?? []) flattenTree(child, out);
  return out;
}

/** Trajectory tab: ThreadTimeline when runId present, else flat session events. */
export function TrajectoryPane({
  events,
  runId,
  highlightSeq,
  onSelectSeq,
  onSelectEvent,
}: Props) {
  const qc = useQueryClient();
  const [focusRunId, setFocusRunId] = useState<string | null>(null);
  const treeQ = useQuery({
    queryKey: ["run-tree", runId],
    queryFn: () => getRunTree(runId!),
    enabled: Boolean(runId),
  });
  const lineage = flattenTree(treeQ.data?.tree);
  const rootRunId = treeQ.data?.rootRunId || lineage[0]?.runId || runId;
  const parent = lineage.find((n) => n.runId === runId) ?? lineage[0];
  const scenario = parent?.scenario;
  const spawnM = useMutation({
    mutationFn: () =>
      spawnSubRun(runId!, {
        scenario: scenario!,
        inputs: { issue: "sub-run from chat" },
        reason: "chat-spawn",
      }),
    onSuccess: (res) => {
      setFocusRunId(res.runId);
      void qc.invalidateQueries({ queryKey: ["run-tree", runId] });
    },
  });

  return (
    <div className="agent-chat-trajectory" data-testid="agent-chat-trajectory">
      {runId ? (
        <>
          <div data-testid="subrun-lineage">
            <p className="muted-line">子代理谱系</p>
            <div style={{ display: "flex", gap: "0.5rem", marginBottom: "0.35rem" }}>
              <button
                type="button"
                data-testid="subrun-spawn"
                disabled={!scenario?.name || spawnM.isPending}
                onClick={() => spawnM.mutate()}
              >
                {spawnM.isPending ? "派生中…" : "派生子 Run"}
              </button>
              <button
                type="button"
                data-testid="subrun-focus-root"
                disabled={!rootRunId}
                onClick={() => setFocusRunId(rootRunId || null)}
              >
                回主
              </button>
            </div>
            {spawnM.isError ? (
              <p className="muted-line" data-testid="subrun-spawn-error">
                {(spawnM.error as Error)?.message || "派生失败"}
              </p>
            ) : null}
            {treeQ.isLoading ? (
              <p className="muted-line">加载谱系…</p>
            ) : lineage.length === 0 ? (
              <p className="muted-line">无子 Run</p>
            ) : (
              <ul style={{ listStyle: "none", padding: 0, margin: "0 0 0.5rem" }}>
                {lineage.map((node) => {
                  const active = (focusRunId || runId) === node.runId;
                  return (
                    <li
                      key={node.runId}
                      data-testid="subrun-lineage-node"
                      data-depth={node.depth ?? 0}
                      data-active={active ? "1" : "0"}
                    >
                      <button
                        type="button"
                        className="event-line"
                        data-testid={`subrun-lineage-select-${node.runId}`}
                        onClick={() => setFocusRunId(node.runId)}
                      >
                        {node.runId}
                        {node.parentRunId ? ` ← ${node.parentRunId}` : " · root"}
                        {` · ${node.status}`}
                      </button>
                    </li>
                  );
                })}
              </ul>
            )}
          </div>
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
        </>
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
