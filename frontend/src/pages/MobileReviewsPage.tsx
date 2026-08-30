import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Check, RefreshCcw, X } from "lucide-react";
import { useState } from "react";
import { Link } from "@tanstack/react-router";
import { decideReview, listReviewsQueue, type ReviewItem } from "@/modules/reviews/api/reviews.api";
import { getCurrentSpaceId } from "@/services/http/client";

/** Compact mobile review surface: Plan/Diff summary + approve/reject. */
export function MobileReviewsPage() {
  const qc = useQueryClient();
  const spaceId = getCurrentSpaceId();
  const [message, setMessage] = useState("");
  const [expanded, setExpanded] = useState<string | null>(null);

  const queueQuery = useQuery({
    queryKey: ["reviews-queue", "all", spaceId, "mobile"],
    queryFn: () => listReviewsQueue("all", 40),
  });

  const decideMut = useMutation({
    mutationFn: ({ id, decision }: { id: string; decision: "approve" | "reject" }) =>
      decideReview(id, { decision, reason: "mobile review", policyProfile: "default" }),
    onSuccess: () => {
      setMessage("已提交");
      qc.invalidateQueries({ queryKey: ["reviews-queue"] });
    },
    onError: (e: Error) => setMessage(e.message),
  });

  const items = queueQuery.data?.items ?? [];

  return (
    <section className="mobile-reviews" data-testid="mobile-reviews-page">
      <header className="mobile-reviews-header">
        <div>
          <p className="mobile-reviews-kicker">ASH · 移动审阅</p>
          <h1>待办评审</h1>
          <p className="muted-line">Space: {spaceId}</p>
        </div>
        <button
          type="button"
          className="btn icon-btn"
          onClick={() => queueQuery.refetch()}
          disabled={queueQuery.isFetching}
          aria-label="刷新"
        >
          <RefreshCcw size={18} strokeWidth={1.8} />
        </button>
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
            onToggle={() => setExpanded((cur) => (cur === item.id ? null : item.id))}
            onDecide={(d) => decideMut.mutate({ id: item.id, decision: d })}
          />
        ))}
      </ul>
      {items.length === 0 && !queueQuery.isLoading ? (
        <p className="muted-line" data-testid="mobile-reviews-empty">
          队列为空
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
  onToggle,
  onDecide,
}: {
  item: ReviewItem;
  open: boolean;
  busy: boolean;
  onToggle: () => void;
  onDecide: (d: "approve" | "reject") => void;
}) {
  return (
    <li className="mobile-review-card" data-testid={`mobile-review-${item.targetType}`}>
      <button type="button" className="mobile-review-main" onClick={onToggle}>
        <strong>{item.title}</strong>
        <span className="muted-line">
          {item.targetType} · {item.queue}
        </span>
        {item.summary ? <span className="muted-line">{item.summary}</span> : null}
      </button>
      {open && item.diff ? <pre className="mobile-review-diff">{item.diff}</pre> : null}
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
