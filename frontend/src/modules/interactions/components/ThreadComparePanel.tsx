import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import {
  compareInteractionThreads,
  forkInteractionThread,
  getInteractionByRun,
  listInteractionSessionThreads,
  type InteractionCompareResult,
} from "../api/interactions.api";
import { formatInteractionError } from "../formatInteractionError";

type Props = {
  /** Optional prefilled left run id (Workbench / Reviews). */
  defaultLeftRunId?: string;
  defaultRightRunId?: string;
};

/** Dual Thread compare entry (GV05–06 Task 8). Resolves run → main thread then Compare API. */
export function ThreadComparePanel({ defaultLeftRunId = "", defaultRightRunId = "" }: Props) {
  const [leftRunId, setLeftRunId] = useState(defaultLeftRunId);
  const [rightRunId, setRightRunId] = useState(defaultRightRunId);
  const [siblingId, setSiblingId] = useState("");
  const [result, setResult] = useState<InteractionCompareResult | null>(null);
  const [error, setError] = useState("");
  const qc = useQueryClient();

  useEffect(() => {
    if (defaultLeftRunId.trim()) setLeftRunId(defaultLeftRunId.trim());
  }, [defaultLeftRunId]);

  useEffect(() => {
    if (defaultRightRunId.trim()) setRightRunId(defaultRightRunId.trim());
  }, [defaultRightRunId]);

  const leftQ = useQuery({
    queryKey: ["interaction-by-run", leftRunId.trim()],
    queryFn: () => getInteractionByRun(leftRunId.trim()),
    enabled: Boolean(leftRunId.trim()),
    retry: false,
  });
  const rightQ = useQuery({
    queryKey: ["interaction-by-run", rightRunId.trim()],
    queryFn: () => getInteractionByRun(rightRunId.trim()),
    enabled: Boolean(rightRunId.trim()),
    retry: false,
  });

  const sessionId = leftQ.data?.thread.sessionId || "";
  const siblingsQ = useQuery({
    queryKey: ["interaction-session-threads", sessionId],
    queryFn: () => listInteractionSessionThreads(sessionId),
    enabled: Boolean(sessionId),
    retry: false,
  });

  const compareMut = useMutation({
    mutationFn: async () => {
      const left = leftQ.data?.thread.id;
      const right = siblingId || rightQ.data?.thread.id;
      if (!left || !right || left === right) {
        throw new Error("左右两侧需能解析到不同的 Interaction Thread");
      }
      return compareInteractionThreads(left, right);
    },
    onSuccess: (data) => {
      setResult(data);
      setError("");
    },
    onError: (e: Error) => {
      setResult(null);
      setError(e.message);
    },
  });

  const rightId = siblingId || rightQ.data?.thread.id || "";
  const canCompare = Boolean(leftQ.data?.thread.id && rightId && leftQ.data.thread.id !== rightId);

  const sideResolveMsg = (side: "左侧" | "右侧", err: unknown) => {
    const raw = (err instanceof Error ? err.message : "").trim() || "无法解析到 Thread";
    if (/^(左侧|右侧)\s*Run/.test(raw)) return raw;
    return `${side} Run：${formatInteractionError(raw)}`;
  };

  const compareTitle = (() => {
    if (compareMut.isPending) return "比对进行中";
    if (!leftRunId.trim()) return "需要填写左侧 Run";
    if (leftQ.isFetching && !leftQ.isError) return "正在解析左侧 Run…";
    if (leftQ.isError) return sideResolveMsg("左侧", leftQ.error);
    if (!leftQ.data?.thread.id) return "左侧 Run 无法解析到 Thread";
    if (!siblingId && !rightRunId.trim()) return "需要填写右侧 Run / 选择 sibling Thread";
    if (!siblingId && rightQ.isFetching && !rightQ.isError) return "正在解析右侧 Run…";
    if (!siblingId && rightQ.isError) return sideResolveMsg("右侧", rightQ.error);
    if (!rightId) return "右侧 Run 无法解析到 Thread";
    if (leftQ.data.thread.id === rightId) return "左右两侧需为不同 Thread";
    return "比对两侧 Thread";
  })();

  const resolveError =
    (leftRunId.trim() && leftQ.isError ? sideResolveMsg("左侧", leftQ.error) : "") ||
    (!siblingId && rightRunId.trim() && rightQ.isError ? sideResolveMsg("右侧", rightQ.error) : "") ||
    "";

  const forkMut = useMutation({
    mutationFn: () => forkInteractionThread(leftQ.data!.thread.id),
    onSuccess: (child) => {
      setSiblingId(child.id);
      setError("");
      void qc.invalidateQueries({ queryKey: ["interaction-session-threads", sessionId] });
    },
    onError: (e: Error) => setError(e.message),
  });

  return (
    <div className="pane" data-testid="thread-compare-panel">
      <div className="pane-title">
        <h2>双 Thread 比对</h2>
        <span>digest / nodes / MemoryLink</span>
      </div>
      <div className="form-grid" style={{ marginBottom: "0.75rem" }}>
        <label>
          左侧 Run
          <input
            data-testid="thread-compare-left"
            value={leftRunId}
            onChange={(e) => setLeftRunId(e.target.value)}
            placeholder="run_…"
          />
        </label>
        <label>
          右侧 Run
          <input
            data-testid="thread-compare-right"
            value={rightRunId}
            onChange={(e) => setRightRunId(e.target.value)}
            placeholder="run_…"
          />
        </label>
      </div>
      {sessionId ? (
        <div className="row-actions" style={{ marginBottom: "0.75rem" }}>
          <label>
            同会话分支
            <select
              data-testid="thread-compare-sibling"
              value={siblingId}
              onChange={(e) => setSiblingId(e.target.value)}
            >
              <option value="">（用右侧 Run）</option>
              {(siblingsQ.data?.items ?? []).map((th) => (
                <option key={th.id} value={th.id}>
                  {th.kind}
                  {th.parentThreadId ? ` ← ${th.parentThreadId.slice(0, 8)}` : ""} {th.id.slice(0, 12)}
                </option>
              ))}
            </select>
          </label>
          <button
            type="button"
            className="btn mini"
            data-testid="thread-compare-fork"
            disabled={!leftQ.data?.thread.id || forkMut.isPending}
            title={
              !leftQ.data?.thread.id
                ? "需要先解析左侧 Thread"
                : forkMut.isPending
                  ? "Fork 中…"
                  : "Fork 出 sibling Thread（需确认）"
            }
            onClick={() => {
              if (!leftQ.data?.thread.id) return;
              const ok = window.confirm("确认 Fork 当前左侧 Thread？将创建 sibling Thread。");
              if (!ok) return;
              forkMut.mutate();
            }}
          >
            Fork
          </button>
        </div>
      ) : null}
      <div className="row-actions">
        <button
          type="button"
          className="btn primary mini"
          data-testid="thread-compare-submit"
          disabled={!canCompare || compareMut.isPending}
          title={compareTitle}
          onClick={() => compareMut.mutate()}
        >
          比对
        </button>
        {leftQ.data?.thread.id ? (
          <span className="muted-line" data-testid="thread-compare-left-th">
            L {leftQ.data.thread.id.slice(0, 12)}
          </span>
        ) : null}
        {rightId ? (
          <span className="muted-line" data-testid="thread-compare-right-th">
            R {rightId.slice(0, 12)}
          </span>
        ) : null}
      </div>
      {resolveError ? (
        <p className="error-text" data-testid="thread-compare-resolve-error">
          {resolveError}
        </p>
      ) : null}
      {error ? (
        <p className="error-text" data-testid="thread-compare-error">
          {error}
        </p>
      ) : null}
      {result ? (
        <div data-testid="thread-compare-result" style={{ marginTop: "0.75rem" }}>
          <p className="muted-line">
            digest {result.leftDigest.slice(0, 14)} ↔ {result.rightDigest.slice(0, 14)}
            {result.leftDigest === result.rightDigest ? " · 相同" : " · 不同"}
          </p>
          <table className="table compact">
            <thead>
              <tr>
                <th>维度</th>
                <th>数量</th>
              </tr>
            </thead>
            <tbody>
              <DiffRow label="nodes +" items={result.nodesAdded} />
              <DiffRow label="nodes −" items={result.nodesRemoved} />
              <DiffRow label="nodes ~" items={result.nodesChanged} />
              <DiffRow label="links +" items={result.linksAdded} />
              <DiffRow label="links −" items={result.linksRemoved} />
            </tbody>
          </table>
        </div>
      ) : null}
    </div>
  );
}

function DiffRow({ label, items }: { label: string; items?: string[] | null }) {
  const rows = items ?? [];
  return (
    <tr>
      <td>{label}</td>
      <td title={rows.join("\n")}>{rows.length}</td>
    </tr>
  );
}
