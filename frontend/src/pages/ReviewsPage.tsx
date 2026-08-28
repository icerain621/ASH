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
import { getCurrentSpaceId } from "@/services/http/client";

export function ReviewsPage() {
  const qc = useQueryClient();
  const spaceId = getCurrentSpaceId();
  const [queue, setQueue] = useState<"all" | "orchestration" | "memory">("orchestration");
  const [reason, setReason] = useState("reviewed from UI");
  const [message, setMessage] = useState("");

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
      decideReview(id, { decision, reason, policyProfile: "default" }),
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
          <h1>编排与记忆评审</h1>
          <p>统一队列：Harness Profile / Scenario patch / Memory candidate。批准后才可升格。</p>
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
          <label className="scenario-picker">
            决定原因
            <input value={reason} onChange={(e) => setReason(e.target.value)} data-testid="reviews-reason" />
          </label>
          <button type="button" className="btn icon-btn" onClick={() => queueQuery.refetch()} disabled={queueQuery.isFetching}>
            <RefreshCcw size={16} strokeWidth={1.8} />
            刷新
          </button>
        </div>
      </div>
      {message ? <p className="muted-line">{message}</p> : null}

      <div className="split ops-split">
        <div className="pane">
          <div className="pane-title">
            <h2>待审项</h2>
            <span>{items.length} 项</span>
          </div>
          {queueQuery.isLoading ? <p className="muted-line">加载中…</p> : null}
          <table className="table" data-testid="reviews-queue-list">
            <thead>
              <tr>
                <th>Item</th>
                <th>Type</th>
                <th>Action</th>
              </tr>
            </thead>
            <tbody>
              {items.map((item) => (
                <ReviewRow key={item.id} item={item} busy={decideMut.isPending} onDecide={(d) => decideMut.mutate({ id: item.id, decision: d })} />
              ))}
              {items.length === 0 && !queueQuery.isLoading && (
                <tr className="empty-row">
                  <td colSpan={3}>队列为空</td>
                </tr>
              )}
            </tbody>
          </table>
        </div>

        <div className="pane">
          <div className="pane-title">
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
            <label>
              from
              <input name="fromVersion" placeholder="1.0.0" />
            </label>
            <label>
              to
              <input name="toVersion" placeholder="1.1.0" />
            </label>
            <label className="wide-field">
              Diff / 说明
              <textarea name="diffText" required rows={6} placeholder="- gate: require citations&#10;+ gate: require citations + human" data-testid="patch-diff" />
            </label>
            <button type="submit" className="btn primary icon-btn" data-testid="patch-create" disabled={createPatchMut.isPending}>
              <Send size={16} strokeWidth={1.8} />
              创建草稿
            </button>
          </form>
          <table className="table" data-testid="patch-draft-list">
            <thead>
              <tr>
                <th>Title</th>
                <th>Scenario</th>
                <th>Action</th>
              </tr>
            </thead>
            <tbody>
              {(draftsQuery.data?.items ?? []).map((p) => (
                <tr key={p.id}>
                  <td>{p.title}</td>
                  <td>{p.scenarioName}</td>
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
              {!(draftsQuery.data?.items ?? []).length && (
                <tr className="empty-row">
                  <td colSpan={3}>暂无草稿。</td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>
    </section>
  );
}

function ReviewRow({
  item,
  busy,
  onDecide,
}: {
  item: ReviewItem;
  busy: boolean;
  onDecide: (d: "approve" | "reject") => void;
}) {
  return (
    <tr data-testid={`review-item-${item.targetType}`}>
      <td>
        <strong>{item.title}</strong>
        {item.summary ? <div className="muted-line">{item.summary}</div> : null}
        {item.diff ? <pre className="code-block compact">{item.diff}</pre> : null}
      </td>
      <td>
        {item.targetType}
        <div className="muted-line">{item.queue}</div>
      </td>
      <td>
        <div className="row-actions">
          <button type="button" className="btn mini ok" disabled={busy} onClick={() => onDecide("approve")} data-testid="review-approve">
            批准
          </button>
          <button type="button" className="btn mini err" disabled={busy} onClick={() => onDecide("reject")} data-testid="review-reject">
            拒绝
          </button>
        </div>
      </td>
    </tr>
  );
}
