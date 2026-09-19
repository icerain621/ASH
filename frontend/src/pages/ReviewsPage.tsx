import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { ClipboardList, RefreshCcw, Send } from "lucide-react";
import { useEffect, useState } from "react";
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
import { projectHooksFromBodyJson } from "@/modules/registry/hooksPolicy";
import { RegistryAssetsPanel } from "@/modules/registry/components/RegistryAssetsPanel";
import { getAuthMe } from "@/modules/platform/api/platform.api";
import { MemoryLinkPanel } from "@/modules/interactions/components/MemoryLinkPanel";
import { ThreadComparePanel } from "@/modules/interactions/components/ThreadComparePanel";
import { ThreadTimeline } from "@/modules/interactions/components/ThreadTimeline";
import { ObservabilityPage } from "@/pages/ObservabilityPage";
import { getCurrentSpaceId } from "@/services/http/client";

type RubricForm = {
  correctness: number;
  safety: number;
  citable: number;
  efficiency: number;
};

type ReviewPillar = "queue" | "assets" | "observe" | "orchestrate";

const defaultRubric: RubricForm = { correctness: 4, safety: 4, citable: 4, efficiency: 4 };

const PILLAR_TABS: { id: ReviewPillar; label: string; testId: string }[] = [
  { id: "queue", label: "队列评审", testId: "review-nav-queue" },
  { id: "assets", label: "启停登记", testId: "review-nav-assets" },
  { id: "observe", label: "监控观测", testId: "review-nav-observe" },
  { id: "orchestrate", label: "编排流程", testId: "review-nav-orchestrate" },
];

function isPendingSecond(status: string | undefined) {
  return status === "pending_second";
}

export function ReviewsPage() {
  const qc = useQueryClient();
  const spaceId = getCurrentSpaceId();
  const [pillar, setPillar] = useState<ReviewPillar>("queue");
  const [queue, setQueue] = useState<"all" | "orchestration" | "memory" | "appeal">("orchestration");
  const [reason, setReason] = useState("控制台评审");
  const [message, setMessage] = useState("");
  const [selected, setSelected] = useState<ReviewItem | null>(null);
  const [rubric, setRubric] = useState<RubricForm>(defaultRubric);
  const [inspectRunId, setInspectRunId] = useState("");
  const [highlightSeq, setHighlightSeq] = useState<number | null>(null);
  const [assigneeInput, setAssigneeInput] = useState("");
  const [overdueOnly, setOverdueOnly] = useState(false);
  const [appealScoreEventId, setAppealScoreEventId] = useState("");
  const [appealReason, setAppealReason] = useState("申诉评分");
  const [memoryDeepLinkId, setMemoryDeepLinkId] = useState("");

  useEffect(() => {
    const q = new URLSearchParams(window.location.search);
    const queueParam = q.get("queue");
    if (queueParam === "memory" || queueParam === "orchestration" || queueParam === "appeal" || queueParam === "all") {
      setQueue(queueParam);
      setPillar("queue");
    }
    const mid = q.get("memoryId")?.trim();
    if (mid) setMemoryDeepLinkId(mid);
  }, []);

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

  const hooksProjection = projectHooksFromBodyJson(policyQuery.data?.pack?.bodyJson);

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
            {pillar === "queue"
              ? "Workbench：左队列 / 中时间线占位 / 右打分决定。"
              : pillar === "assets"
                ? "Agent / Memory 资产启停与生效策略摘要。"
                : pillar === "observe"
                  ? "监控观测（含 review_sla）与指标入口。"
                  : "编排与自动化流程的薄管控入口。"}{" "}
            {pillar === "queue" ? (
              <a href="/ui/m/reviews" data-testid="reviews-mobile-link">
                移动审阅
              </a>
            ) : null}
          </p>
          <span className="scope-badge">Space: {spaceId}</span>
          {pillar === "queue" && typeof reviewSlaHours === "number" ? (
            <span className="scope-badge" data-testid="reviews-sla-hint" style={{ marginLeft: 8 }}>
              评审 SLA: {reviewSlaHours}h
            </span>
          ) : null}
        </div>
        {pillar === "queue" ? (
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
        ) : null}
      </div>

      {memoryDeepLinkId ? (
        <div className="pane" data-testid="reviews-memory-deeplink" style={{ marginBottom: 12 }}>
          <p className="muted-line">
            记忆厚审入口 · 候选 <code>{memoryDeepLinkId}</code>
            {" · "}
            <Link to="/memory" className="inline-link" data-testid="reviews-memory-back">
              返回记忆薄批
            </Link>
            {" · "}
            <button
              type="button"
              className="btn mini"
              data-testid="reviews-memory-deeplink-clear"
              onClick={() => setMemoryDeepLinkId("")}
            >
              清除提示
            </button>
          </p>
        </div>
      ) : null}

      <nav className="work-mode" data-testid="review-pillar-nav" role="tablist" aria-label="评审管控板块" style={{ marginBottom: 12 }}>
        {PILLAR_TABS.map((tab) => (
          <button
            key={tab.id}
            type="button"
            role="tab"
            className={`work-mode-btn${pillar === tab.id ? " active" : ""}`}
            aria-selected={pillar === tab.id}
            data-testid={tab.testId}
            onClick={() => setPillar(tab.id)}
          >
            {tab.label}
          </button>
        ))}
      </nav>

      {message ? <p className="muted-line">{message}</p> : null}

      {pillar === "assets" ? (
        <div data-testid="review-assets-panel">
          <RegistryAssetsPanel spaceId={spaceId} />
          <p className="muted-line" style={{ marginTop: 8 }}>
            策略包编辑请前往{" "}
            <Link to="/space" className="inline-link" data-testid="review-assets-space-link">
              Space 设置
            </Link>
            。
          </p>
        </div>
      ) : null}

      {pillar === "observe" ? (
        <div data-testid="review-observe-panel">
          <div className="toolbar metrics-toolbar" style={{ marginBottom: 8 }}>
            <Link to="/metrics" className="btn mini" data-testid="review-open-metrics">
              打开完整指标
            </Link>
            <span className="muted-line">观测面板含 review_sla Waker / 告警可见性</span>
          </div>
          <ObservabilityPage />
        </div>
      ) : null}

      {pillar === "orchestrate" ? (
        <div className="pane" data-testid="review-orchestrate-panel">
          <div className="pane-title">
            <h2>编排流程</h2>
          </div>
          <p className="muted-line">
            运行队列与自动化/Scenario 编排仍使用既有页面；此处提供评审管控上下文中的快捷入口。
          </p>
          <div className="row-actions" style={{ gap: 8, flexWrap: "wrap" }}>
            <Link to="/runs" className="btn primary" data-testid="review-link-runs">
              打开运行
            </Link>
            <Link to="/automation" className="btn" data-testid="review-link-automation">
              打开自动化
            </Link>
          </div>
          {policyQuery.isLoading ? (
            <p className="muted-line" style={{ marginTop: 12 }}>
              加载空间策略…
            </p>
          ) : hooksProjection ? (
            <div className="pane" data-testid="reviews-hooks-projection" style={{ marginTop: 12 }}>
              <div className="pane-title">
                <h2>Hooks</h2>
                <span className="muted-line">{hooksProjection.version || "ash.hooks.v1"}</span>
              </div>
              {hooksProjection.parseError ? (
                <p className="muted-line">{hooksProjection.parseError}</p>
              ) : hooksProjection.rules.length === 0 ? (
                <p className="muted-line">已配置 hooks 容器，规则列表为空。</p>
              ) : (
                <table className="table compact">
                  <thead>
                    <tr>
                      <th>event</th>
                      <th>tool</th>
                      <th>action</th>
                      <th>reason</th>
                    </tr>
                  </thead>
                  <tbody>
                    {hooksProjection.rules.map((rule, idx) => (
                      <tr key={`hook-rule-${idx}`} data-testid="reviews-hooks-rule">
                        <td>{rule.event || "—"}</td>
                        <td>{rule.tool || "*"}</td>
                        <td>{rule.action || "—"}</td>
                        <td className="muted-line">{rule.reason || "—"}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
              <p className="muted-line" style={{ marginTop: 8 }}>
                Run 侧 PreToolUse 决策见 Agent Trajectory 的 <code>hook.*</code> 事件；附录{" "}
                <code>doc/appendices/ash-hooks-v1.md</code>。
              </p>
            </div>
          ) : (
            <p className="muted-line" style={{ marginTop: 12 }} data-testid="reviews-hooks-empty">
              当前空间 BodyJSON 未配置 <code>hooks</code>。
            </p>
          )}
        </div>
      ) : null}

      {pillar === "queue" ? (
        <>
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
                    <th>项</th>
                    <th>类型</th>
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
                      <td colSpan={2} data-testid="reviews-empty">
                        {overdueOnly ? "当前无逾期评审" : "待审队列为空"}
                      </td>
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
              {!selected ? (
                <p className="muted-line" data-testid="reviews-decide-hint">
                  请先选择左侧队列项再决定
                </p>
              ) : (
                <>
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
                          disabled={assignMut.isPending || !assigneeInput.trim()}
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
                      disabled={decideMut.isPending}
                      onClick={() => selected && decideMut.mutate({ id: selected.id, decision: "approve" })}
                      data-testid="review-approve"
                    >
                      {isAppealItem ? "保持评分" : secondSign ? "第二签批准" : "批准"}
                    </button>
                    <button
                      type="button"
                      className="btn mini err"
                      disabled={decideMut.isPending}
                      onClick={() => selected && decideMut.mutate({ id: selected.id, decision: "reject" })}
                      data-testid="review-reject"
                    >
                      {isAppealItem ? "作废评分" : secondSign ? "第二签拒绝" : "拒绝"}
                    </button>
                  </div>
                </>
              )}
            </div>
          </div>

          <div className="pane" data-testid="reviews-create-appeal" style={{ marginTop: 16 }}>
            <div className="pane-title">
              <h2>申诉评分</h2>
            </div>
            <div className="row-actions" style={{ gap: 8, flexWrap: "wrap" }}>
              <label className="scenario-picker">
                评分事件 ID
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
        </>
      ) : null}
    </section>
  );
}
