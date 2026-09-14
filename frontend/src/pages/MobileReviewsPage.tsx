import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Check, RefreshCcw, X } from "lucide-react";
import { useState } from "react";
import { Link } from "@tanstack/react-router";
import { assignReview, decideReview, listReviewsQueue, type ReviewItem } from "@/modules/reviews/api/reviews.api";
import { getAuthMe } from "@/modules/platform/api/platform.api";
import { getCurrentSpaceId } from "@/services/http/client";

type RubricForm = {
  correctness: number;
  safety: number;
  citable: number;
  efficiency: number;
};

const defaultRubric: RubricForm = { correctness: 4, safety: 4, citable: 4, efficiency: 4 };

const rubricDims = [
  ["correctness", "正确性"],
  ["safety", "安全性"],
  ["citable", "可引用"],
  ["efficiency", "效率"],
] as const;

function isPendingSecond(status: string | undefined) {
  return status === "pending_second";
}

/** Compact mobile review surface: Plan/Diff summary + approve/reject. */
export function MobileReviewsPage() {
  const qc = useQueryClient();
  const spaceId = getCurrentSpaceId();
  const [message, setMessage] = useState("");
  const [expanded, setExpanded] = useState<string | null>(null);
  const [overdueOnly, setOverdueOnly] = useState(false);
  const [reason, setReason] = useState("mobile review");
  const [rubric, setRubric] = useState<RubricForm>(defaultRubric);
  const [assigneeInput, setAssigneeInput] = useState("");

  const queueQuery = useQuery({
    queryKey: ["reviews-queue", "all", spaceId, "mobile"],
    queryFn: () => listReviewsQueue("all", 40),
  });
  const meQuery = useQuery({
    queryKey: ["auth-me", spaceId, "mobile"],
    queryFn: getAuthMe,
  });

  const decideMut = useMutation({
    mutationFn: ({ id, decision }: { id: string; decision: "approve" | "reject" }) =>
      decideReview(id, {
        decision,
        reason: reason.trim() || "mobile review",
        policyProfile: "default",
        rubric,
      }),
    onSuccess: () => {
      setMessage("已提交");
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

  const allItems = queueQuery.data?.items ?? [];
  const items = overdueOnly ? allItems.filter((it) => it.slaBreach) : allItems;
  const canAssign = (meQuery.data?.permissions ?? []).includes("reviews:assign");

  return (
    <section className="mobile-reviews" data-testid="mobile-reviews-page">
      <header className="mobile-reviews-header">
        <div>
          <p className="mobile-reviews-kicker">ASH · 移动审阅</p>
          <h1>待办评审</h1>
          <p className="muted-line">Space: {spaceId}</p>
        </div>
        <div style={{ display: "flex", gap: 8, alignItems: "center" }}>
          <button
            type="button"
            className={`btn mini ${overdueOnly ? "ok" : ""}`}
            data-testid="mobile-reviews-filter-overdue"
            aria-pressed={overdueOnly}
            onClick={() => setOverdueOnly((v) => !v)}
          >
            仅逾期
          </button>
          <button
            type="button"
            className="btn icon-btn"
            onClick={() => queueQuery.refetch()}
            disabled={queueQuery.isFetching}
            aria-label="刷新"
          >
            <RefreshCcw size={18} strokeWidth={1.8} />
          </button>
        </div>
      </header>

      {message ? <p className="mobile-reviews-toast">{message}</p> : null}
      {queueQuery.isLoading ? <p className="muted-line">加载中…</p> : null}

      <ul className="mobile-reviews-list">
        {items.map((item) => (
          <MobileReviewCard
            key={item.id}
            item={item}
            open={expanded === item.id}
            busy={decideMut.isPending}
            assignBusy={assignMut.isPending}
            canAssign={canAssign}
            reason={reason}
            rubric={rubric}
            assigneeInput={assigneeInput}
            onReason={setReason}
            onRubric={setRubric}
            onAssigneeInput={setAssigneeInput}
            onToggle={() => setExpanded((cur) => (cur === item.id ? null : item.id))}
            onDecide={(d) => decideMut.mutate({ id: item.id, decision: d })}
            onAssign={() => {
              const id = assigneeInput.trim();
              if (id) assignMut.mutate({ id: item.id, assigneeId: id });
            }}
          />
        ))}
      </ul>
      {items.length === 0 && !queueQuery.isLoading ? (
        <p className="muted-line" data-testid="mobile-reviews-empty">
          {overdueOnly ? "当前无逾期评审" : "待办评审为空"}
        </p>
      ) : null}

      <footer className="mobile-reviews-footer">
        <Link to="/reviews" className="muted-line">
          打开完整评审台 →
        </Link>
      </footer>
    </section>
  );
}

function MobileReviewCard({
  item,
  open,
  busy,
  assignBusy,
  canAssign,
  reason,
  rubric,
  assigneeInput,
  onReason,
  onRubric,
  onAssigneeInput,
  onToggle,
  onDecide,
  onAssign,
}: {
  item: ReviewItem;
  open: boolean;
  busy: boolean;
  assignBusy: boolean;
  canAssign: boolean;
  reason: string;
  rubric: RubricForm;
  assigneeInput: string;
  onReason: (v: string) => void;
  onRubric: (fn: (r: RubricForm) => RubricForm) => void;
  onAssigneeInput: (v: string) => void;
  onToggle: () => void;
  onDecide: (d: "approve" | "reject") => void;
  onAssign: () => void;
}) {
  return (
    <li className="mobile-review-card" data-testid={`mobile-review-${item.targetType}`}>
      <button type="button" className="mobile-review-main" onClick={onToggle}>
        <strong>{item.title}</strong>
        <span className="muted-line">
          {item.targetType} · {item.queue}
        </span>
        {item.summary ? <span className="muted-line">{item.summary}</span> : null}
        <span style={{ display: "flex", flexWrap: "wrap", gap: 6, marginTop: 4 }}>
          {isPendingSecond(item.status) ? (
            <span className="scope-badge" data-testid="mobile-review-pending-second" style={{ marginTop: 0 }}>
              第二签
            </span>
          ) : null}
          {item.slaBreach ? (
            <span className="scope-badge" data-testid="mobile-review-sla-breach" style={{ marginTop: 0 }}>
              逾期
            </span>
          ) : null}
          {item.assigneeId ? (
            <span className="scope-badge" data-testid="mobile-review-assignee" style={{ marginTop: 0 }}>
              负责人: {item.assigneeId}
            </span>
          ) : null}
        </span>
      </button>
      {open && item.diff ? <pre className="mobile-review-diff">{item.diff}</pre> : null}
      {open ? (
        <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
          <label className="muted-line" style={{ display: "flex", flexDirection: "column", gap: 4 }}>
            原因
            <input
              value={reason}
              onChange={(e) => onReason(e.target.value)}
              placeholder="mobile review"
              data-testid="mobile-review-reason"
            />
          </label>
          {canAssign ? (
            <div style={{ display: "flex", gap: 8 }}>
              <input
                value={assigneeInput}
                onChange={(e) => onAssigneeInput(e.target.value)}
                placeholder="assigneeId"
                aria-label="分配给"
                style={{ flex: 1 }}
              />
              <button
                type="button"
                className="btn mini"
                disabled={assignBusy || !assigneeInput.trim()}
                onClick={onAssign}
                data-testid="mobile-review-assign"
              >
                分配
              </button>
            </div>
          ) : (
            <p className="muted-line" data-testid="mobile-reviews-assign-denied">
              需要 reviews:assign 才能分配
            </p>
          )}
          <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: "0.4rem" }}>
            {rubricDims.map(([key, label]) => (
              <label key={key} className="muted-line" style={{ display: "flex", flexDirection: "column", gap: 4 }}>
                {label}
                <input
                  type="number"
                  min={1}
                  max={5}
                  value={rubric[key]}
                  onChange={(e) => onRubric((r) => ({ ...r, [key]: Number(e.target.value) }))}
                  data-testid={`mobile-rubric-${key}`}
                />
              </label>
            ))}
          </div>
        </div>
      ) : null}
      <div className="mobile-review-actions">
        <button type="button" className="btn ok icon-btn" disabled={busy} onClick={() => onDecide("approve")} data-testid="mobile-review-approve">
          <Check size={16} strokeWidth={1.8} />
          批准
        </button>
        <button type="button" className="btn err icon-btn" disabled={busy} onClick={() => onDecide("reject")} data-testid="mobile-review-reject">
          <X size={16} strokeWidth={1.8} />
          拒绝
        </button>
      </div>
    </li>
  );
}
