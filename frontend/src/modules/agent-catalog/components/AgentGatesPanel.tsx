import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import {
  decideAgentReview,
  getMemoryEvalCounts,
  listAgentReviews,
  runPassK,
} from "@/modules/agent-catalog/api/agentCatalog.api";

/** Review-side gate for templates/components + Pass^k (F08: not on Doctor). */
export function AgentGatesPanel() {
  const qc = useQueryClient();
  const [passTarget, setPassTarget] = useState("tpl.react");
  const [passK, setPassK] = useState(2);
  const [message, setMessage] = useState("");

  const reviewsQuery = useQuery({
    queryKey: ["agent-reviews"],
    queryFn: () => listAgentReviews(),
  });
  const evalQuery = useQuery({
    queryKey: ["memory-eval"],
    queryFn: getMemoryEvalCounts,
  });

  const decideMut = useMutation({
    mutationFn: ({ id, decision }: { id: string; decision: "approve" | "reject" }) =>
      decideAgentReview(id, decision, "console gate"),
    onSuccess: (item) => {
      setMessage(`已${item.status === "approved" ? "批准" : "拒绝"} ${item.targetId}`);
      void qc.invalidateQueries({ queryKey: ["agent-reviews"] });
      void qc.invalidateQueries({ queryKey: ["agent-templates"] });
      void qc.invalidateQueries({ queryKey: ["agent-components"] });
    },
    onError: (err) => setMessage((err as Error).message),
  });

  const passMut = useMutation({
    mutationFn: () => runPassK(passTarget, passK),
    onSuccess: (res) => {
      setMessage(`Pass^k k=${res.k} target=${res.target} allPassed=${res.allPassed}`);
    },
    onError: (err) => setMessage((err as Error).message),
  });

  const items = reviewsQuery.data?.items ?? [];

  return (
    <div data-testid="agent-gates-panel">
      <p className="muted-line">组件评审闸与 Pass^k 抽样（质量面；不在诊断页）。</p>
      {message ? <p className="muted-line" data-testid="agent-gates-message">{message}</p> : null}

      <div className="pane" data-testid="memory-eval-pane">
        <div className="pane-title">
          <h2>记忆三计数</h2>
          <span>F05</span>
        </div>
        <p className="muted-line" data-testid="memory-eval-counts">
          记得 {evalQuery.data?.remembered ?? "—"} · 用对 {evalQuery.data?.usedCorrectly ?? "—"} · 错误更新{" "}
          {evalQuery.data?.wrongUpdate ?? "—"}
        </p>
      </div>

      <div className="pane">
        <div className="pane-title">
          <h2>待决组件评审</h2>
          <span>{items.length}</span>
        </div>
        {items.length === 0 ? (
          <p className="muted-line" data-testid="agent-gates-empty">
            无待决项
          </p>
        ) : (
          <ul data-testid="agent-gates-list">
            {items.map((item) => (
              <li key={item.id} data-testid={`agent-review-${item.id}`}>
                <code>{item.kind}</code> · {item.targetId}
                <div className="toolbar">
                  <button
                    type="button"
                    className="btn mini primary"
                    data-testid={`agent-review-approve-${item.id}`}
                    onClick={() => decideMut.mutate({ id: item.id, decision: "approve" })}
                  >
                    批准
                  </button>
                  <button
                    type="button"
                    className="btn mini"
                    data-testid={`agent-review-reject-${item.id}`}
                    onClick={() => decideMut.mutate({ id: item.id, decision: "reject" })}
                  >
                    拒绝
                  </button>
                </div>
              </li>
            ))}
          </ul>
        )}
      </div>

      <div className="pane" data-testid="passk-pane">
        <div className="pane-title">
          <h2>Pass^k</h2>
        </div>
        <div className="toolbar">
          <label>
            目标
            <input
              value={passTarget}
              data-testid="passk-target"
              onChange={(e) => setPassTarget(e.target.value)}
            />
          </label>
          <label>
            k
            <input
              type="number"
              min={1}
              value={passK}
              data-testid="passk-k"
              onChange={(e) => setPassK(Number(e.target.value) || 1)}
            />
          </label>
          <button
            type="button"
            className="btn primary"
            data-testid="passk-run"
            disabled={passMut.isPending}
            onClick={() => passMut.mutate()}
          >
            运行抽样
          </button>
        </div>
      </div>
    </div>
  );
}
