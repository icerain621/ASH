import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
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

vi.mock("@/services/http/client", () => ({
  getCurrentSpaceId: () => "local",
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
    vi.mocked(getAuthMe).mockResolvedValue({
      user: { id: "u1", displayName: "Reviewer" },
      space: { id: "local", name: "local" },
      role: "reviewer",
      permissions: ["reviews:assign", "memory:review"],
    });
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
        expect.objectContaining({ decision: "approve", reason: "reviewed from UI" }),
      );
    });
    const call = vi.mocked(decideReview).mock.calls.at(-1)?.[1] as { rubric?: unknown };
    expect(call.rubric).toBeUndefined();
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
