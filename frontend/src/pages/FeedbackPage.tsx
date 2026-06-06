import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { MessageSquarePlus, RefreshCcw, Send } from "lucide-react";
import { useState } from "react";
import { createFeedback, listFeedback, updateFeedback, type Feedback } from "@/modules/closure/api/closure.api";

export function FeedbackPage() {
  const qc = useQueryClient();
  const [status, setStatus] = useState("open");
  const [category, setCategory] = useState("");
  const [targetType, setTargetType] = useState("");
  const feedbackQuery = useQuery({
    queryKey: ["feedback", status, category, targetType],
    queryFn: () => listFeedback({ status, category, targetType, limit: 80 }),
  });
  const feedbackMut = useMutation({
    mutationFn: createFeedback,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["feedback"] });
      qc.invalidateQueries({ queryKey: ["observability-alerts"] });
      qc.invalidateQueries({ queryKey: ["metrics"] });
    },
  });
  const updateMut = useMutation({
    mutationFn: ({ item, next }: { item: Feedback; next: string }) =>
      updateFeedback(item.id, { status: next, severity: next === "resolved" ? "info" : item.severity }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["feedback"] }),
  });

  function submit(formData: FormData) {
    feedbackMut.mutate({
      targetType: String(formData.get("targetType") || "run"),
      targetId: String(formData.get("targetId") || ""),
      rating: Number(formData.get("rating") || 0),
      category: String(formData.get("category") || "general"),
      source: "ui",
      comment: String(formData.get("comment") || ""),
    });
  }

  return (
    <section className="panel active">
      <div className="page-kicker">
        <MessageSquarePlus size={17} strokeWidth={1.8} />
        Feedback
      </div>
      <div className="page-heading">
        <div>
          <h1>反馈闭环</h1>
          <p>记录运行、Memory、CI 诊断与产物反馈，并跟踪低分处理状态。</p>
        </div>
        <div className="toolbar metrics-toolbar">
          <label className="scenario-picker">
            状态
            <select value={status} onChange={(e) => setStatus(e.target.value)}>
              <option value="">全部</option>
              <option value="open">open</option>
              <option value="triaged">triaged</option>
              <option value="in_progress">in_progress</option>
              <option value="resolved">resolved</option>
              <option value="dismissed">dismissed</option>
            </select>
          </label>
          <label className="scenario-picker">
            分类
            <select value={category} onChange={(e) => setCategory(e.target.value)}>
              <option value="">全部</option>
              <option value="general">general</option>
              <option value="quality">quality</option>
              <option value="ci">ci</option>
              <option value="memory">memory</option>
              <option value="ux">ux</option>
            </select>
          </label>
          <input className="metric-filter" value={targetType} placeholder="target type" onChange={(e) => setTargetType(e.target.value)} />
          <button className="btn icon-btn" onClick={() => feedbackQuery.refetch()} disabled={feedbackQuery.isFetching}>
            <RefreshCcw size={16} strokeWidth={1.8} />
            刷新
          </button>
        </div>
      </div>

      {(feedbackQuery.error || feedbackMut.error || updateMut.error) && (
        <p className="error-text">{(feedbackQuery.error || feedbackMut.error || updateMut.error as Error)?.message}</p>
      )}

      <div className="split ops-split">
        <div className="pane">
          <div className="pane-title">
            <h2>提交反馈</h2>
            <span>{feedbackMut.isPending ? "saving" : "ready"}</span>
          </div>
          <form
            className="form-grid"
            onSubmit={(event) => {
              event.preventDefault();
              submit(new FormData(event.currentTarget));
            }}
          >
            <label>
              Target Type
              <select name="targetType" defaultValue="run">
                <option value="run">Run</option>
                <option value="artifact">Artifact</option>
                <option value="memory_hit">Memory Hit</option>
                <option value="ci_diagnosis">CI Diagnosis</option>
              </select>
            </label>
            <label>
              Category
              <select name="category" defaultValue="quality">
                <option value="quality">quality</option>
                <option value="ci">ci</option>
                <option value="memory">memory</option>
                <option value="ux">ux</option>
                <option value="general">general</option>
              </select>
            </label>
            <label>
              Target ID
              <input name="targetId" placeholder="run_..." required />
            </label>
            <label>
              Rating
              <input name="rating" type="number" min="1" max="5" defaultValue="3" />
            </label>
            <label className="wide-field">
              Comment
              <textarea name="comment" rows={4} placeholder="记录采纳、失败原因或改进建议。" />
            </label>
            <button className="btn primary icon-btn" type="submit" disabled={feedbackMut.isPending}>
              <Send size={16} strokeWidth={1.8} />
              提交反馈
            </button>
          </form>
        </div>

        <div className="pane">
          <div className="pane-title">
            <h2>低分与处理</h2>
            <span>{feedbackQuery.data?.items.filter((item) => item.rating > 0 && item.rating <= 2).length ?? 0} 条低分</span>
          </div>
          <table className="table">
            <thead>
              <tr>
                <th>target</th>
                <th>rating</th>
                <th>status</th>
                <th>actions</th>
              </tr>
            </thead>
            <tbody>
              {(feedbackQuery.data?.items ?? []).length === 0 && (
                <tr className="empty-row">
                  <td colSpan={4}>暂无反馈。</td>
                </tr>
              )}
              {(feedbackQuery.data?.items ?? []).map((item) => (
                <tr key={item.id}>
                  <td>
                    {item.targetType}
                    <div className="muted-line">{item.targetId}</div>
                  </td>
                  <td>{item.rating || "-"}</td>
                  <td>
                    <StatusPill value={item.status} />
                  </td>
                  <td>
                    <div className="row-actions">
                      <button className="btn mini" onClick={() => updateMut.mutate({ item, next: "triaged" })} disabled={item.status === "triaged"}>
                        triage
                      </button>
                      <button className="btn mini ok" onClick={() => updateMut.mutate({ item, next: "resolved" })} disabled={item.status === "resolved"}>
                        resolve
                      </button>
                      <button className="btn mini err" onClick={() => updateMut.mutate({ item, next: "dismissed" })} disabled={item.status === "dismissed"}>
                        dismiss
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      <div className="pane">
        <div className="pane-title">
          <h2>反馈详情</h2>
          <span>{feedbackQuery.data?.items.length ?? 0} 条</span>
        </div>
        <table className="table">
          <thead>
            <tr>
              <th>category</th>
              <th>severity</th>
              <th>source</th>
              <th>comment</th>
            </tr>
          </thead>
          <tbody>
            {(feedbackQuery.data?.items ?? []).map((item) => (
              <tr key={`detail-${item.id}`}>
                <td>{item.category}</td>
                <td>
                  <StatusPill value={item.severity} />
                </td>
                <td>{item.source}</td>
                <td>{item.comment || "-"}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}

function StatusPill({ value }: { value: string }) {
  const lowered = value.toLowerCase();
  const tone = ["resolved", "info", "ok"].includes(lowered) ? "ok" : ["dismissed", "warn", "critical"].includes(lowered) ? "err" : "idle";
  return (
    <span className={`status-pill ${tone}`}>
      <span className="status-dot" />
      {value || "unknown"}
    </span>
  );
}
