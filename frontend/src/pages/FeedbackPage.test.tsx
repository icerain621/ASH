import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { FeedbackPage } from "./FeedbackPage";
import { renderPage } from "@/test/renderPage";

vi.mock("@/modules/closure/api/closure.api", () => ({
  listFeedback: vi.fn().mockResolvedValue({ items: [] }),
  createFeedback: vi.fn(),
  updateFeedback: vi.fn(),
}));

describe("FeedbackPage", () => {
  it("renders feedback heading and submit control", async () => {
    renderPage(<FeedbackPage />);
    expect(screen.getByRole("heading", { name: "反馈闭环" })).toBeInTheDocument();
    await waitFor(() => {
      expect(screen.getByRole("button", { name: "提交反馈" })).toBeInTheDocument();
    });
  });
});
