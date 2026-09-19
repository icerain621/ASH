import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ReviewsPage } from "./ReviewsPage";
import { decideReview, listReviewsQueue, assignReview, createScoreAppeal } from "@/modules/reviews/api/reviews.api";
import { getSpacePolicy } from "@/modules/registry/api/registry.api";
import { getAuthMe } from "@/modules/platform/api/platform.api";

vi.mock("@/modules/reviews/api/reviews.api", () => ({
  listReviewsQueue: vi.fn(async () => ({
    items: [
      {
        id: "harness_profile:hprof_1",
        queue: "orchestration",
        targetType: "harness_profile",
        targetId: "hprof_1",
        title: "default@v1",
        status: "pending",
        spaceId: "local",
        createdAt: 1,
        assigneeId: "rev_bob",
        slaBreach: true,
        ageHours: 96,
      },
      {
        id: "harness_profile:hprof_2",
        queue: "orchestration",
        targetType: "harness_profile",
        targetId: "hprof_2",
        title: "team@v2",
        status: "pending_second",
        spaceId: "local",
        createdAt: 2,
      },
      {
        id: "score_appeal:score_1",
        queue: "appeal",
        targetType: "score_appeal",
        targetId: "score_1",
        title: "Appeal score_1",
        summary: "please reconsider",
        status: "pending",
        spaceId: "local",
        createdAt: 3,
      },
    ],
  })),
  listScenarioPatches: vi.fn(async () => ({ items: [] })),
  decideReview: vi.fn().mockResolvedValue({ ok: true }),
  assignReview: vi.fn().mockResolvedValue({ ok: true, reviewId: "harness_profile:hprof_1", assigneeId: "op_x" }),
  createScenarioPatch: vi.fn(),
  submitScenarioPatchReview: vi.fn(),
  createScoreAppeal: vi.fn().mockResolvedValue({ id: "score_appeal:score_x" }),
}));

vi.mock("@/modules/registry/api/registry.api", () => ({
  listAgentAssets: vi.fn().mockResolvedValue({ items: [] }),
  listMemoryAssets: vi.fn().mockResolvedValue({ items: [] }),
  getSpacePolicy: vi.fn().mockResolvedValue({
    pack: { spaceId: "local", citationMode: "optional", multiSign: true, reviewSlaHours: 48 },
    effective: {
      spaceId: "local",
      spaceKind: "team",
      citationMode: "required",
      multiSign: true,
      reviewSlaHours: 48,
      sources: ["kind:team"],
    },
  }),
  getSpaceQuotas: vi.fn().mockResolvedValue({
    spaceId: "local",
    limits: { maxConcurrentRuns: 0, tokenBudgetProxy: 0 },
    usage: { activeConcurrentRuns: 0, tokenBudgetProxyUsed: 0 },
  }),
  putSpacePolicy: vi.fn(),
  createAgentAsset: vi.fn(),
  createMemoryAsset: vi.fn(),
  patchAgentAssetStatus: vi.fn(),
  patchMemoryAssetStatus: vi.fn(),
}));

vi.mock("@/modules/platform/api/platform.api", () => ({
  getAuthMe: vi.fn().mockResolvedValue({
    user: { id: "u1", displayName: "Reviewer" },
    space: { id: "local", name: "local" },
    role: "reviewer",
    permissions: ["reviews:assign", "memory:review"],
  }),
}));

vi.mock("@/modules/interactions/api/interactions.api", () => ({
  getInteractionByRun: vi.fn(async () => ({
    runId: "run_x",
    thread: { id: "th_x", spaceId: "local", runId: "run_x", kind: "main", status: "open" },
  })),
  getInteractionThread: vi.fn(async () => ({
    threadId: "th_x",
    runId: "run_x",
    spaceId: "local",
    nodes: [],
    links: [],
    digest: "thd_x",
    headSeq: 0,
  })),
  listInteractionMemoryLinks: vi.fn(async () => ({ threadId: "th_x", items: [] })),
  sealInteractionThread: vi.fn(),
  replayInteractionThread: vi.fn(),
  compareInteractionThreads: vi.fn(),
}));

vi.mock("@/pages/ObservabilityPage", () => ({
  ObservabilityPage: () => <div data-testid="observability-page-stub">Observability stub (review_sla)</div>,
}));

vi.mock("@/services/http/client", () => ({
  getCurrentSpaceId: () => "local",
}));

vi.mock("@tanstack/react-router", () => ({
  Link: ({ children, ...props }: { children: ReactNode; to?: string; className?: string }) => (
    <a href={props.to ?? "#"} className={props.className} data-to={props.to} data-testid={(props as { "data-testid"?: string })["data-testid"]}>
      {children}
    </a>
  ),
}));

function renderReviews() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <ReviewsPage />
    </QueryClientProvider>,
  );
}

describe("ReviewsPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    window.history.replaceState({}, "", "/reviews");
    vi.mocked(getAuthMe).mockResolvedValue({
      user: { id: "u1", displayName: "Reviewer" },
      space: { id: "local", name: "local" },
      role: "reviewer",
      permissions: ["reviews:assign", "memory:review"],
    });
  });

  it("reads memoryId query into deeplink banner and memory queue", async () => {
    window.history.replaceState({}, "", "/reviews?queue=memory&memoryId=mem_deeplink");
    renderReviews();
    expect(await screen.findByTestId("reviews-memory-deeplink")).toHaveTextContent("mem_deeplink");
    expect(screen.getByTestId("reviews-queue-filter")).toHaveValue("memory");
    expect(listReviewsQueue).toHaveBeenCalledWith("memory", 80);
  });

  it("renders pillar sub-nav and defaults to queue workbench", async () => {
    renderReviews();
    expect(await screen.findByTestId("reviews-page")).toBeTruthy();
    expect(screen.getByTestId("review-pillar-nav")).toBeTruthy();
    expect(screen.getByTestId("review-nav-queue")).toHaveAttribute("aria-selected", "true");
    expect(screen.getByTestId("review-nav-assets")).toBeTruthy();
    expect(screen.getByTestId("review-nav-observe")).toBeTruthy();
    expect(screen.getByTestId("review-nav-orchestrate")).toBeTruthy();
    expect(screen.getByTestId("reviews-workbench")).toBeTruthy();
    expect(screen.queryByTestId("review-assets-panel")).toBeNull();
  });

  it("switches to assets, observe, and orchestrate panels", async () => {
    renderReviews();
    expect(await screen.findByTestId("review-pillar-nav")).toBeTruthy();

    fireEvent.click(screen.getByTestId("review-nav-assets"));
    expect(await screen.findByTestId("review-assets-panel")).toBeTruthy();
    expect(await screen.findByTestId("registry-assets-panel")).toBeTruthy();
    expect(await screen.findByTestId("effective-policy-summary")).toHaveTextContent("生效策略");
    expect(screen.getByTestId("review-assets-space-link")).toHaveAttribute("data-to", "/space");
    expect(screen.queryByTestId("reviews-workbench")).toBeNull();

    fireEvent.click(screen.getByTestId("review-nav-observe"));
    expect(await screen.findByTestId("review-observe-panel")).toBeTruthy();
    expect(screen.getByTestId("observability-page-stub")).toBeTruthy();
    expect(screen.getByTestId("review-open-metrics")).toHaveAttribute("data-to", "/metrics");

    fireEvent.click(screen.getByTestId("review-nav-orchestrate"));
    expect(await screen.findByTestId("review-orchestrate-panel")).toBeTruthy();
    expect(screen.getByTestId("review-link-runs")).toHaveAttribute("data-to", "/runs");
    expect(screen.getByTestId("review-link-automation")).toHaveAttribute("data-to", "/automation");

    fireEvent.click(screen.getByTestId("review-nav-queue"));
    expect(await screen.findByTestId("reviews-workbench")).toBeTruthy();
  });

  it("renders workbench columns and submits rubric", async () => {
    renderReviews();
    expect(await screen.findByTestId("reviews-page")).toBeTruthy();
    expect(await screen.findByText("default@v1")).toBeTruthy();
    expect(screen.getByTestId("reviews-workbench")).toBeTruthy();
    expect(screen.getByTestId("reviews-timeline-placeholder")).toBeTruthy();
    expect(screen.getByTestId("reviews-decide-form")).toBeTruthy();
    expect(await screen.findByTestId("reviews-sla-hint")).toHaveTextContent("评审 SLA: 48h");
    expect(getSpacePolicy).toHaveBeenCalledWith("local");

    fireEvent.click(screen.getByText("default@v1"));
    fireEvent.change(screen.getByTestId("rubric-correctness"), { target: { value: "3" } });
    fireEvent.change(screen.getByTestId("rubric-safety"), { target: { value: "2" } });
    fireEvent.change(screen.getByTestId("rubric-citable"), { target: { value: "4" } });
    fireEvent.change(screen.getByTestId("rubric-efficiency"), { target: { value: "5" } });
    fireEvent.click(screen.getByTestId("review-approve"));

    await waitFor(() => {
      expect(decideReview).toHaveBeenCalledWith(
        "harness_profile:hprof_1",
        expect.objectContaining({
          decision: "approve",
          rubric: { correctness: 3, safety: 2, citable: 4, efficiency: 5 },
        }),
      );
    });
  });

  it("shows pending_second badge and second-sign decide labels", async () => {
    renderReviews();
    expect(await screen.findByText("team@v2")).toBeTruthy();
    expect(screen.getByTestId("review-pending-second-badge")).toHaveTextContent("待第二签");

    fireEvent.click(screen.getByText("team@v2"));
    expect(screen.getByTestId("review-detail-pending-second")).toHaveTextContent("待第二签");
    expect(screen.getByTestId("reviews-inspect-run")).toBeTruthy();
    expect(screen.getByTestId("review-approve")).toHaveTextContent("第二签批准");
    expect(screen.getByTestId("review-reject")).toHaveTextContent("第二签拒绝");

    fireEvent.click(screen.getByTestId("review-approve"));
    await waitFor(() => {
      expect(decideReview).toHaveBeenCalledWith(
        "harness_profile:hprof_2",
        expect.objectContaining({ decision: "approve" }),
      );
    });
    expect(listReviewsQueue).toHaveBeenCalled();
  });

  it("shows assignee and submits assign when permitted", async () => {
    renderReviews();
    expect(await screen.findByTestId("review-assignee-badge")).toHaveTextContent("负责人: rev_bob");
    fireEvent.click(screen.getByText("default@v1"));
    expect(screen.getByTestId("review-detail-assignee")).toHaveTextContent("负责人: rev_bob");
    fireEvent.change(screen.getByTestId("reviews-assignee-input"), { target: { value: "op_x" } });
    fireEvent.click(screen.getByTestId("review-assign"));
    await waitFor(() => {
      expect(assignReview).toHaveBeenCalledWith(
        "harness_profile:hprof_1",
        expect.objectContaining({ assigneeId: "op_x" }),
      );
    });
  });

  it("shows SLA breach badge and filters overdue only", async () => {
    renderReviews();
    expect(await screen.findByTestId("review-sla-breach-badge")).toHaveTextContent("SLA 逾期");
    expect(screen.getByText("default@v1")).toBeTruthy();
    expect(screen.getByText("team@v2")).toBeTruthy();

    fireEvent.click(screen.getByTestId("reviews-filter-overdue"));
    expect(screen.getByText("default@v1")).toBeTruthy();
    expect(screen.queryByText("team@v2")).toBeNull();
  });

  it("hides assign controls without reviews:assign", async () => {
    vi.mocked(getAuthMe).mockResolvedValue({
      user: { id: "u2", displayName: "Viewer" },
      space: { id: "local", name: "local" },
      role: "viewer",
      permissions: ["artifact:read"],
    });
    renderReviews();
    expect(await screen.findByTestId("reviews-decide-hint")).toHaveTextContent("请先选择左侧队列项再决定");
    fireEvent.click(await screen.findByText("default@v1"));
    expect(await screen.findByTestId("reviews-assign-denied")).toHaveTextContent("reviews:assign");
    expect(screen.queryByTestId("review-assign")).toBeNull();
    expect(screen.queryByTestId("reviews-assignee-input")).toBeNull();
  });

  it("offers appeal queue filter and decides appeal without rubric", async () => {
    renderReviews();
    const filter = await screen.findByTestId("reviews-queue-filter");
    expect(filter.querySelector('option[value="appeal"]')).toBeTruthy();

    fireEvent.click(await screen.findByText("Appeal score_1"));
    expect(screen.getByTestId("reviews-appeal-no-rubric")).toBeTruthy();
    expect(screen.queryByTestId("rubric-correctness")).toBeNull();
    expect(screen.getByTestId("review-approve")).toHaveTextContent("保持评分");

    fireEvent.click(screen.getByTestId("review-approve"));
    await waitFor(() => {
      expect(decideReview).toHaveBeenCalledWith(
        "score_appeal:score_1",
        expect.objectContaining({ decision: "approve", reason: "控制台评审" }),
      );
    });
    const call = vi.mocked(decideReview).mock.calls.at(-1)?.[1] as { rubric?: unknown };
    expect(call.rubric).toBeUndefined();
  });

  it("shows empty queue copy for overdue filter", async () => {
    vi.mocked(listReviewsQueue).mockResolvedValueOnce({ items: [] });
    renderReviews();
    expect(await screen.findByTestId("reviews-empty")).toHaveTextContent("待审队列为空");
    fireEvent.click(screen.getByTestId("reviews-filter-overdue"));
    expect(screen.getByTestId("reviews-empty")).toHaveTextContent("当前无逾期评审");
  });

  it("submits compact create-appeal form", async () => {
    renderReviews();
    fireEvent.change(await screen.findByTestId("appeal-score-event-id"), {
      target: { value: "score_abc" },
    });
    fireEvent.change(screen.getByTestId("appeal-reason"), { target: { value: "too harsh" } });
    fireEvent.click(screen.getByTestId("appeal-create"));
    await waitFor(() => {
      expect(createScoreAppeal).toHaveBeenCalledWith("score_abc", { reason: "too harsh" });
    });
  });
});
