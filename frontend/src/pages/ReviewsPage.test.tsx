import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ReviewsPage } from "./ReviewsPage";
import { decideReview, listReviewsQueue } from "@/modules/reviews/api/reviews.api";
import { getSpacePolicy } from "@/modules/registry/api/registry.api";

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
    ],
  })),
  listScenarioPatches: vi.fn(async () => ({ items: [] })),
  decideReview: vi.fn().mockResolvedValue({ ok: true }),
  createScenarioPatch: vi.fn(),
  submitScenarioPatchReview: vi.fn(),
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

describe("ReviewsPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders workbench columns and submits rubric", async () => {
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={qc}>
        <ReviewsPage />
      </QueryClientProvider>,
    );
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
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={qc}>
        <ReviewsPage />
      </QueryClientProvider>,
    );
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
});
