import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ClipboardList, RefreshCcw, Send } from "lucide-react";
import { useState } from "react";
import {
  createScenarioPatch,
  decideReview,
  listReviewsQueue,
  listScenarioPatches,
  submitScenarioPatchReview,
  type ReviewItem,
} from "@/modules/reviews/api/reviews.api";
import { ThreadComparePanel } from "@/modules/interactions/components/ThreadComparePanel";
import { getCurrentSpaceId } from "@/services/http/client";

type RubricForm = {
  correctness: number;
  safety: number;
  citable: number;
  efficiency: number;
};

const defaultRubric: RubricForm = { correctness: 4, safety: 4, citable: 4, efficiency: 4 };

export function ReviewsPage() {
  const qc = useQueryClient();
  const spaceId = getCurrentSpaceId();
  const [queue, setQueue] = useState<"all" | "orchestration" | "memory">("orchestration");
  const [reason, setReason] = useState("reviewed from UI");
  const [message, setMessage] = useState("");
  const [selected, setSelected] = useState<ReviewItem | null>(null);
  const [rubric, setRubric] = useState<RubricForm>(defaultRubric);

  const queueQuery = useQuery({
    queryKey: ["reviews-queue", queue, spaceId],
    queryFn: () => listReviewsQueue(queue, 80),
  });
  const draftsQuery = useQuery({
    queryKey: ["scenario-patches", "draft", spaceId],
    queryFn: () => listScenarioPatches("draft"),
  });

  const decideMut = useMutation({
    mutationFn: ({ id, decision }: { id: string; decision: "approve" | "reject" }) =>
      decideReview(id, { decision, reason, policyProfile: "default", rubric }),
    onSuccess: () => {
      setMessage("评审已提交");
      qc.invalidateQueries({ queryKey: ["reviews-queue"] });
    },
    onError: (e: Error) => setMessage(e.message),
  });

  const createPatchMut = useMutation({
    mutationFn: createScenarioPatch,
    onSuccess: () => {
      setMessage("Scenario patch 草稿已创建");
      qc.invalidateQueries({ queryKey: ["scenario-patches"] });
    },
    onError: (e: Error) => setMessage(e.message),
  });

  const submitPatchMut = useMutation({
    mutationFn: submitScenarioPatchReview,
    onSuccess: () => {
      setMessage("已提交编排评审");
      qc.invalidateQueries({ queryKey: ["scenario-patches"] });
      qc.invalidateQueries({ queryKey: ["reviews-queue"] });
    },
    onError: (e: Error) => setMessage(e.message),
  });

  function onCreatePatch(fd: FormData) {
    createPatchMut.mutate({
      scenarioName: String(fd.get("scenarioName") || ""),
      fromVersion: String(fd.get("fromVersion") || ""),
      toVersion: String(fd.get("toVersion") || ""),
      title: String(fd.get("title") || ""),
      diffText: String(fd.get("diffText") || ""),
    });
  }

  const items = queueQuery.data?.items ?? [];

  return (
    <section className="panel active" data-testid="reviews-page">
      <div className="page-kicker">
        <ClipboardList size={17} strokeWidth={1.8} />
        Reviews
      </div>
      <div className="page-heading">
        <div>
          <h1>评审管控</h1>
          <p>
            Workbench：左队列 / 中时间线占位 / 右打分决定。{" "}
            <a href="/ui/m/reviews" data-testid="reviews-mobile-link">
              移动审阅
            </a>
          </p>
          <span className="scope-badge">Space: {spaceId}</span>
        </div>
        <div className="toolbar metrics-toolbar">
          <label className="scenario-picker">
            队列
            <select value={queue} onChange={(e) => setQueue(e.target.value as typeof queue)} data-testid="reviews-queue-filter">
              <option value="orchestration">编排</option>
              <option value="memory">记忆</option>
              <option value="all">全部</option>
            </select>
          </label>
          <button type="button" className="btn icon-btn" onClick={() => queueQuery.refetch()} disabled={queueQuery.isFetching}>
            <RefreshCcw size={16} strokeWidth={1.8} />
            刷新
          </button>
        </div>
      </div>
      {message ? <p className="muted-line">{message}</p> : null}

      <div className="split ops-split" data-testid="reviews-workbench" style={{ gridTemplateColumns: "1fr 1fr 1fr" }}>
        <div className="pane">
          <div className="pane-title">
            <h2>待审队列</h2>
            <span>{items.length} 项</span>
          </div>
          {queueQuery.isLoading ? <p className="muted-line">加载中…</p> : null}
          <table className="table" data-testid="reviews-queue-list">
            <thead>
              <tr>
                <th>Item</th>
                <th>Type</th>
              </tr>
            </thead>
            <tbody>
              {items.map((item) => (
                <tr
                  key={item.id}
                  data-testid={`review-item-${item.targetType}`}
                  className={selected?.id === item.id ? "selected" : undefined}
                  onClick={() => setSelected(item)}
                  style={{ cursor: "pointer" }}
                >
                  <td>
                    <strong>{item.title}</strong>
                    {item.summary ? <div className="muted-line">{item.summary}</div> : null}
                  </td>
                  <td>
                    {item.targetType}
                    <div className="muted-line">{item.status}</div>
                  </td>
                </tr>
              ))}
              {items.length === 0 && !queueQuery.isLoading && (
                <tr className="empty-row">
                  <td colSpan={2}>队列为空</td>
                </tr>
              )}
            </tbody>
          </table>

          <div className="pane-title" style={{ marginTop: 16 }}>
            <h2>Scenario patch 草稿</h2>
            <span>{draftsQuery.data?.items.length ?? 0} 草稿</span>
          </div>
          <form
            className="form-grid"
            onSubmit={(e) => {
              e.preventDefault();
              onCreatePatch(new FormData(e.currentTarget));
              e.currentTarget.reset();
            }}
          >
            <label>
              场景名
              <input name="scenarioName" required placeholder="feature_delivery" data-testid="patch-scenario" />
            </label>
            <label>
              标题
              <input name="title" required placeholder="收紧 citation gate" data-testid="patch-title" />
            </label>
            <label className="wide-field">
              Diff / 说明
              <textarea name="diffText" required rows={4} data-testid="patch-diff" />
            </label>
            <input type="hidden" name="fromVersion" />
            <input type="hidden" name="toVersion" />
            <button type="submit" className="btn primary icon-btn" data-testid="patch-create" disabled={createPatchMut.isPending}>
              <Send size={16} strokeWidth={1.8} />
              创建草稿
            </button>
          </form>
          <table className="table" data-testid="patch-draft-list">
            <tbody>
              {(draftsQuery.data?.items ?? []).map((p) => (
                <tr key={p.id}>
                  <td>{p.title}</td>
                  <td>
                    <button
                      type="button"
                      className="btn mini"
                      onClick={() => submitPatchMut.mutate(p.id)}
                      disabled={submitPatchMut.isPending}
                      data-testid={`patch-submit-${p.id}`}
                    >
                      提交评审
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        <div className="pane" data-testid="reviews-timeline-placeholder">
          <div className="pane-title">
            <h2>时间线 / 事件</h2>
          </div>
          {selected ? (
            <>
              <p>
                <strong>{selected.title}</strong> ({selected.targetType})
              </p>
              <p className="muted-line">中栏占位：后续接入 Thread Timeline / MemoryLink。</p>
              {selected.diff ? <pre className="code-block compact">{selected.diff}</pre> : null}
              <ThreadComparePanel />
            </>
          ) : (
            <p className="muted-line">选择左侧队列项以查看详情。</p>
          )}
        </div>

        <div className="pane" data-testid="reviews-decide-form">
          <div className="pane-title">
            <h2>决定 + Rubric</h2>
          </div>
          <label className="scenario-picker">
            原因
            <input value={reason} onChange={(e) => setReason(e.target.value)} data-testid="reviews-reason" />
          </label>
          {(
            [
              ["correctness", "正确性"],
              ["safety", "安全性"],
              ["citable", "可引用"],
              ["efficiency", "效率"],
            ] as const
          ).map(([key, label]) => (
            <label key={key} className="scenario-picker">
              {label}
              <input
                type="number"
                min={1}
                max={5}
                value={rubric[key]}
                onChange={(e) => setRubric((r) => ({ ...r, [key]: Number(e.target.value) }))}
                data-testid={`rubric-${key}`}
              />
            </label>
          ))}
          <div className="row-actions">
            <button
              type="button"
              className="btn mini ok"
              disabled={decideMut.isPending || !selected}
              onClick={() => selected && decideMut.mutate({ id: selected.id, decision: "approve" })}
              data-testid="review-approve"
            >
              批准
            </button>
            <button
              type="button"
              className="btn mini err"
              disabled={decideMut.isPending || !selected}
              onClick={() => selected && decideMut.mutate({ id: selected.id, decision: "reject" })}
              data-testid="review-reject"
            >
              拒绝
            </button>
          </div>
        </div>
      </div>
    </section>
  );
}
