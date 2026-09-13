import { useMutation, useQuery } from "@tanstack/react-query";
import { useState } from "react";
import {
  compareInteractionThreads,
  getInteractionByRun,
  type InteractionCompareResult,
} from "../api/interactions.api";

type Props = {
  /** Optional prefilled left run id (Workbench / Reviews). */
  defaultLeftRunId?: string;
  defaultRightRunId?: string;
};

/** Dual Thread compare entry (GV05–06 Task 8). Resolves run → main thread then Compare API. */
export function ThreadComparePanel({ defaultLeftRunId = "", defaultRightRunId = "" }: Props) {
  const [leftRunId, setLeftRunId] = useState(defaultLeftRunId);
  const [rightRunId, setRightRunId] = useState(defaultRightRunId);
  const [result, setResult] = useState<InteractionCompareResult | null>(null);
  const [error, setError] = useState("");

  const leftQ = useQuery({
    queryKey: ["interaction-by-run", leftRunId.trim()],
    queryFn: () => getInteractionByRun(leftRunId.trim()),
    enabled: Boolean(leftRunId.trim()),
  });
  const rightQ = useQuery({
    queryKey: ["interaction-by-run", rightRunId.trim()],
    queryFn: () => getInteractionByRun(rightRunId.trim()),
    enabled: Boolean(rightRunId.trim()),
  });

  const compareMut = useMutation({
    mutationFn: async () => {
      const left = leftQ.data?.thread.id;
      const right = rightQ.data?.thread.id;
      if (!left || !right) {
        throw new Error("左右两侧需能解析到 Interaction Thread");
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

  const canCompare = Boolean(leftQ.data?.thread.id && rightQ.data?.thread.id);

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
      <div className="row-actions">
        <button
          type="button"
          className="btn primary mini"
          data-testid="thread-compare-submit"
          disabled={!canCompare || compareMut.isPending}
          onClick={() => compareMut.mutate()}
        >
          比对
        </button>
        {leftQ.data?.thread.id ? (
          <span className="muted-line" data-testid="thread-compare-left-th">
            L {leftQ.data.thread.id.slice(0, 12)}
          </span>
        ) : null}
        {rightQ.data?.thread.id ? (
          <span className="muted-line" data-testid="thread-compare-right-th">
            R {rightQ.data.thread.id.slice(0, 12)}
          </span>
        ) : null}
      </div>
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

function DiffRow({ label, items }: { label: string; items: string[] }) {
  return (
    <tr>
      <td>{label}</td>
      <td title={items.join("\n")}>{items.length}</td>
    </tr>
  );
}
