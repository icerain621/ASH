import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ReviewsPage } from "./ReviewsPage";
import { decideReview } from "@/modules/reviews/api/reviews.api";

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
    ],
  })),
  listScenarioPatches: vi.fn(async () => ({ items: [] })),
  decideReview: vi.fn().mockResolvedValue({ ok: true }),
  createScenarioPatch: vi.fn(),
  submitScenarioPatchReview: vi.fn(),
}));

vi.mock("@/modules/interactions/api/interactions.api", () => ({
  getInteractionByRun: vi.fn(),
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
});
