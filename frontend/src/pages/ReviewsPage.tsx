import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ClipboardList, RefreshCcw, Send } from "lucide-react";
import { useState } from "react";
import {
  assignReview,
  createScenarioPatch,
  createScoreAppeal,
  decideReview,
  listReviewsQueue,
  listScenarioPatches,
  submitScenarioPatchReview,
  type ReviewItem,
} from "@/modules/reviews/api/reviews.api";
import { getSpacePolicy } from "@/modules/registry/api/registry.api";
import { getAuthMe } from "@/modules/platform/api/platform.api";
import { MemoryLinkPanel } from "@/modules/interactions/components/MemoryLinkPanel";
import { ThreadComparePanel } from "@/modules/interactions/components/ThreadComparePanel";
import { ThreadTimeline } from "@/modules/interactions/components/ThreadTimeline";
import { getCurrentSpaceId } from "@/services/http/client";

type RubricForm = {
  correctness: number;
  safety: number;
  citable: number;
  efficiency: number;
};

const defaultRubric: RubricForm = { correctness: 4, safety: 4, citable: 4, efficiency: 4 };

function isPendingSecond(status: string | undefined) {
  return status === "pending_second";
}

export function ReviewsPage() {
  const qc = useQueryClient();
  const spaceId = getCurrentSpaceId();
  const [queue, setQueue] = useState<"all" | "orchestration" | "memory" | "appeal">("orchestration");
  const [reason, setReason] = useState("reviewed from UI");
  const [message, setMessage] = useState("");
  const [selected, setSelected] = useState<ReviewItem | null>(null);
  const [rubric, setRubric] = useState<RubricForm>(defaultRubric);
  const [inspectRunId, setInspectRunId] = useState("");
  const [highlightSeq, setHighlightSeq] = useState<number | null>(null);
  const [assigneeInput, setAssigneeInput] = useState("");
  const [overdueOnly, setOverdueOnly] = useState(false);
  const [appealScoreEventId, setAppealScoreEventId] = useState("");
  const [appealReason, setAppealReason] = useState("申诉评分");

  const queueQuery = useQuery({
    queryKey: ["reviews-queue", queue, spaceId],
    queryFn: () => listReviewsQueue(queue, 80),
  });
  const draftsQuery = useQuery({
    queryKey: ["scenario-patches", "draft", spaceId],
    queryFn: () => listScenarioPatches("draft"),
  });
  const policyQuery = useQuery({
    queryKey: ["space-policy", spaceId],
    queryFn: () => getSpacePolicy(spaceId),
  });
  const meQuery = useQuery({
    queryKey: ["auth-me", spaceId],
    queryFn: getAuthMe,
  });

  const isAppealItem = selected?.targetType === "score_appeal";

  const decideMut = useMutation({
    mutationFn: ({ id, decision }: { id: string; decision: "approve" | "reject" }) => {
      const body: {
        decision: string;
        reason: string;
        policyProfile?: string;
        rubric?: RubricForm;
      } = { decision, reason, policyProfile: "default" };
      if (!isAppealItem) {
        body.rubric = rubric;
      }
      return decideReview(id, body);
    },
    onSuccess: () => {
      setMessage("评审已提交");
      qc.invalidateQueries({ queryKey: ["reviews-queue"] });
    },
    onError: (e: Error) => setMessage(e.message),
  });

  const appealMut = useMutation({
    mutationFn: () => createScoreAppeal(appealScoreEventId.trim(), { reason: appealReason.trim() }),
    onSuccess: () => {
      setMessage("评分申诉已创建");
      setAppealScoreEventId("");
      qc.invalidateQueries({ queryKey: ["reviews-queue"] });
    },
    onError: (e: Error) => setMessage(e.message),
  });

  const assignMut = useMutation({
    mutationFn: ({ id, assigneeId }: { id: string; assigneeId: string }) =>
      assignReview(id, { assigneeId }),
    onSuccess: (_data, vars) => {
      setMessage(`已分配给 ${vars.assigneeId}`);
      setAssigneeInput("");
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

  const allItems = queueQuery.data?.items ?? [];
  const items = overdueOnly ? allItems.filter((it) => it.slaBreach) : allItems;
  const reviewSlaHours = policyQuery.data?.effective?.reviewSlaHours;
  const secondSign = isPendingSecond(selected?.status);
  const canAssign = (meQuery.data?.permissions ?? []).includes("reviews:assign");

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
          {typeof reviewSlaHours === "number" ? (
            <span className="scope-badge" data-testid="reviews-sla-hint" style={{ marginLeft: 8 }}>
              评审 SLA: {reviewSlaHours}h
            </span>
          ) : null}
        </div>
        <div className="toolbar metrics-toolbar">
          <label className="scenario-picker">
            队列
            <select value={queue} onChange={(e) => setQueue(e.target.value as typeof queue)} data-testid="reviews-queue-filter">
              <option value="orchestration">编排</option>
              <option value="memory">记忆</option>
              <option value="appeal">申诉</option>
              <option value="all">全部</option>
            </select>
          </label>
          <button
            type="button"
            className={`btn mini ${overdueOnly ? "ok" : ""}`}
            data-testid="reviews-filter-overdue"
            aria-pressed={overdueOnly}
            onClick={() => setOverdueOnly((v) => !v)}
          >
            仅逾期
          </button>
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
                    {isPendingSecond(item.status) ? (
                      <span className="scope-badge" data-testid="review-pending-second-badge" style={{ marginLeft: 6 }}>
                        待第二签
                      </span>
                    ) : null}
                    {item.slaBreach ? (
                      <span className="scope-badge" data-testid="review-sla-breach-badge" style={{ marginLeft: 6 }}>
                        SLA 逾期
                      </span>
                    ) : null}
                    {item.assigneeId ? (
                      <span className="scope-badge" data-testid="review-assignee-badge" style={{ marginLeft: 6 }}>
                        负责人: {item.assigneeId}
                      </span>
                    ) : null}
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
            <h2>时间线 / MemoryLink</h2>
          </div>
          {selected ? (
            <>
              <p>
                <strong>{selected.title}</strong> ({selected.targetType})
                {isPendingSecond(selected.status) ? (
                  <span className="scope-badge" data-testid="review-detail-pending-second" style={{ marginLeft: 6 }}>
                    待第二签
                  </span>
                ) : null}
                {selected.assigneeId ? (
                  <span className="scope-badge" data-testid="review-detail-assignee" style={{ marginLeft: 6 }}>
                    负责人: {selected.assigneeId}
                  </span>
                ) : null}
              </p>
              {selected.diff ? <pre className="code-block compact">{selected.diff}</pre> : null}
              <label className="scenario-picker" style={{ display: "block", marginBottom: 8 }}>
                关联 Run（探查 Thread）
                <input
                  data-testid="reviews-inspect-run"
                  value={inspectRunId}
                  placeholder="run_…"
                  onChange={(e) => {
                    setInspectRunId(e.target.value);
                    setHighlightSeq(null);
                  }}
                />
              </label>
              {inspectRunId.trim() ? (
                <>
                  <ThreadTimeline
                    runId={inspectRunId.trim()}
                    highlightSeq={highlightSeq}
                    onSelectSeq={setHighlightSeq}
                  />
                  <MemoryLinkPanel
                    runId={inspectRunId.trim()}
                    highlightSeq={highlightSeq}
                    onSelectLinkSeq={setHighlightSeq}
                  />
                </>
              ) : (
                <p className="muted-line">输入 Run ID 后加载 Thread 时间线与 MemoryLink。</p>
              )}
              <ThreadComparePanel defaultLeftRunId={inspectRunId.trim()} />
            </>
          ) : (
            <p className="muted-line">选择左侧队列项以查看详情。</p>
          )}
        </div>

        <div className="pane" data-testid="reviews-decide-form">
          <div className="pane-title">
            <h2>
              {isAppealItem
                ? "申诉决定（保持 / 作废）"
                : secondSign
                  ? "第二签决定 + Rubric"
                  : "决定 + Rubric"}
            </h2>
          </div>
          <label className="scenario-picker">
            原因
            <input value={reason} onChange={(e) => setReason(e.target.value)} data-testid="reviews-reason" />
          </label>
          {canAssign ? (
            <>
              <label className="scenario-picker">
                分配给
                <input
                  value={assigneeInput}
                  onChange={(e) => setAssigneeInput(e.target.value)}
                  placeholder="assigneeId"
                  data-testid="reviews-assignee-input"
                />
              </label>
              <div className="row-actions" style={{ marginBottom: 8 }}>
                <button
                  type="button"
                  className="btn mini"
                  disabled={assignMut.isPending || !selected || !assigneeInput.trim()}
                  onClick={() =>
                    selected && assignMut.mutate({ id: selected.id, assigneeId: assigneeInput.trim() })
                  }
                  data-testid="review-assign"
                >
                  分配
                </button>
              </div>
            </>
          ) : (
            <p className="muted-line" data-testid="reviews-assign-denied">
              需要 reviews:assign 权限才能分配责任人
            </p>
          )}
          {!isAppealItem
            ? (
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
              ))
            : (
              <p className="muted-line" data-testid="reviews-appeal-no-rubric">
                评分申诉决定无需 Rubric：批准=保持，拒绝=作废评分
              </p>
            )}
          <div className="row-actions">
            <button
              type="button"
              className="btn mini ok"
              disabled={decideMut.isPending || !selected}
              onClick={() => selected && decideMut.mutate({ id: selected.id, decision: "approve" })}
              data-testid="review-approve"
            >
              {isAppealItem ? "保持评分" : secondSign ? "第二签批准" : "批准"}
            </button>
            <button
              type="button"
              className="btn mini err"
              disabled={decideMut.isPending || !selected}
              onClick={() => selected && decideMut.mutate({ id: selected.id, decision: "reject" })}
              data-testid="review-reject"
            >
              {isAppealItem ? "作废评分" : secondSign ? "第二签拒绝" : "拒绝"}
            </button>
          </div>
        </div>
      </div>

      <div className="pane" data-testid="reviews-create-appeal" style={{ marginTop: 16 }}>
        <div className="pane-title">
          <h2>申诉评分</h2>
        </div>
        <div className="row-actions" style={{ gap: 8, flexWrap: "wrap" }}>
          <label className="scenario-picker">
            scoreEventId
            <input
              value={appealScoreEventId}
              onChange={(e) => setAppealScoreEventId(e.target.value)}
              placeholder="score_…"
              data-testid="appeal-score-event-id"
            />
          </label>
          <label className="scenario-picker">
            原因
            <input
              value={appealReason}
              onChange={(e) => setAppealReason(e.target.value)}
              data-testid="appeal-reason"
            />
          </label>
          <button
            type="button"
            className="btn mini"
            disabled={appealMut.isPending || !appealScoreEventId.trim() || !appealReason.trim()}
            onClick={() => appealMut.mutate()}
            data-testid="appeal-create"
          >
            提交申诉
          </button>
        </div>
      </div>
    </section>
  );
}
