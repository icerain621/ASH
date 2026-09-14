import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { MobileReviewsPage } from "./MobileReviewsPage";
import { renderPage } from "@/test/renderPage";
import { assignReview, decideReview } from "@/modules/reviews/api/reviews.api";
import { getAuthMe } from "@/modules/platform/api/platform.api";

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
        status: "pending_second",
        spaceId: "local",
        createdAt: 1,
        assigneeId: "u1",
        slaBreach: true,
      },
      {
        id: "harness_profile:hp2",
        queue: "orchestration",
        targetType: "memory_candidate",
        targetId: "mc1",
        title: "other v1",
        status: "in_review",
        spaceId: "local",
        createdAt: 2,
      },
    ],
  }),
  decideReview: vi.fn().mockResolvedValue({ ok: true }),
  assignReview: vi.fn().mockResolvedValue({ ok: true, reviewId: "harness_profile:hp1", assigneeId: "op_x" }),
}));

vi.mock("@/modules/platform/api/platform.api", () => ({
  getAuthMe: vi.fn().mockResolvedValue({
    user: { id: "u1", displayName: "Reviewer" },
    space: { id: "local", name: "local" },
    role: "reviewer",
    permissions: ["reviews:assign", "memory:review"],
  }),
}));

describe("MobileReviewsPage", () => {
  it("renders compact queue with approve/reject", async () => {
    renderPage(<MobileReviewsPage />);
    expect(await screen.findByText("default v3")).toBeInTheDocument();
    expect(screen.getAllByTestId("mobile-review-approve")[0]).toBeInTheDocument();
    expect(screen.getAllByTestId("mobile-review-reject")[0]).toBeInTheDocument();
  });

  it("shows pending_second, sla, and assignee badges plus overdue filter", async () => {
    renderPage(<MobileReviewsPage />);
    expect(await screen.findByTestId("mobile-review-pending-second")).toHaveTextContent("第二签");
    expect(screen.getByTestId("mobile-review-sla-breach")).toHaveTextContent("逾期");
    expect(screen.getByTestId("mobile-review-assignee")).toHaveTextContent("负责人: u1");
    expect(screen.getByText("other v1")).toBeInTheDocument();

    fireEvent.click(screen.getByTestId("mobile-reviews-filter-overdue"));
    expect(screen.getByText("default v3")).toBeInTheDocument();
    expect(screen.queryByText("other v1")).toBeNull();
  });

  it("shows compact rubric and assign on expanded card", async () => {
    renderPage(<MobileReviewsPage />);
    expect(await screen.findByText("default v3")).toBeInTheDocument();
    expect(getAuthMe).toHaveBeenCalled();

    expect(screen.queryByTestId("mobile-rubric-correctness")).toBeNull();
    expect(screen.queryByTestId("mobile-review-assign")).toBeNull();

    fireEvent.click(screen.getByText("default v3"));
    expect(screen.getByTestId("mobile-rubric-correctness")).toBeInTheDocument();
    expect(screen.getByTestId("mobile-rubric-safety")).toBeInTheDocument();
    expect(screen.getByTestId("mobile-rubric-citable")).toBeInTheDocument();
    expect(screen.getByTestId("mobile-rubric-efficiency")).toBeInTheDocument();
    expect(screen.getByTestId("mobile-review-assign")).toBeInTheDocument();

    fireEvent.change(screen.getByTestId("mobile-rubric-correctness"), { target: { value: "3" } });
    fireEvent.click(screen.getAllByTestId("mobile-review-approve")[0]);
    await waitFor(() => {
      expect(decideReview).toHaveBeenCalledWith(
        "harness_profile:hp1",
        expect.objectContaining({
          decision: "approve",
          reason: "mobile review",
          rubric: expect.objectContaining({ correctness: 3, safety: 4, citable: 4, efficiency: 4 }),
        }),
      );
    });

    fireEvent.change(screen.getByPlaceholderText("assigneeId"), { target: { value: "op_x" } });
    fireEvent.click(screen.getByTestId("mobile-review-assign"));
    await waitFor(() => {
      expect(assignReview).toHaveBeenCalledWith("harness_profile:hp1", expect.objectContaining({ assigneeId: "op_x" }));
    });
  });
});
