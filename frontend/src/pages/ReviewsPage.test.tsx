import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ReviewsPage } from "./ReviewsPage";

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
  decideReview: vi.fn(),
  createScenarioPatch: vi.fn(),
  submitScenarioPatchReview: vi.fn(),
}));

vi.mock("@/services/http/client", () => ({
  getCurrentSpaceId: () => "local",
}));

describe("ReviewsPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders orchestration queue items", async () => {
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={qc}>
        <ReviewsPage />
      </QueryClientProvider>,
    );
    expect(await screen.findByTestId("reviews-page")).toBeTruthy();
    expect(await screen.findByText("default@v1")).toBeTruthy();
    expect(screen.getByTestId("review-approve")).toBeTruthy();
  });
});
