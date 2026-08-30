import { screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { MobileReviewsPage } from "./MobileReviewsPage";
import { renderPage } from "@/test/renderPage";

vi.mock("@tanstack/react-router", async () => {
  const actual = await vi.importActual<typeof import("@tanstack/react-router")>("@tanstack/react-router");
  return {
    ...actual,
    Link: ({ children, to }: { children: React.ReactNode; to: string }) => <a href={to}>{children}</a>,
  };
});

vi.mock("@/modules/reviews/api/reviews.api", () => ({
  listReviewsQueue: vi.fn().mockResolvedValue({
    items: [
      {
        id: "harness_profile:hp1",
        queue: "orchestration",
        targetType: "harness_profile",
        targetId: "hp1",
        title: "default v3",
        summary: "sandbox isolated",
        diff: "+ defaultMode: isolated",
        status: "in_review",
        spaceId: "local",
        createdAt: 1,
      },
    ],
  }),
  decideReview: vi.fn().mockResolvedValue({ ok: true }),
}));

describe("MobileReviewsPage", () => {
  it("renders compact queue with approve/reject", async () => {
    renderPage(<MobileReviewsPage />);
    expect(await screen.findByText("default v3")).toBeInTheDocument();
    expect(screen.getByTestId("mobile-review-approve")).toBeInTheDocument();
    expect(screen.getByTestId("mobile-review-reject")).toBeInTheDocument();
  });
});
