import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { FeedbackPage } from "./FeedbackPage";
import { createFeedback, listFeedback } from "@/modules/closure/api/closure.api";
import { renderPage } from "@/test/renderPage";
import { ApiError } from "@/services/http/client";

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

  it("exposes appendix K target types", async () => {
    renderPage(<FeedbackPage />);
    const select = await screen.findByTestId("feedback-target-type");
    expect(select).toBeInTheDocument();
    expect(select.innerHTML).toContain("harness_profile");
    expect(select.innerHTML).toContain("scenario_patch");
  });

  it("tolerates null feedback items", async () => {
    vi.mocked(listFeedback).mockResolvedValueOnce({ items: null as unknown as [] });
    renderPage(<FeedbackPage />);
    await waitFor(() => {
      expect(screen.getByRole("heading", { name: "反馈闭环" })).toBeInTheDocument();
      expect(screen.getByText("0 条低分")).toBeInTheDocument();
    });
  });

  it("defaults status filter to all and lists triaged feedback", async () => {
    vi.mocked(listFeedback).mockResolvedValueOnce({
      items: [
        {
          id: "fb_1",
          spaceId: "local",
          targetType: "run",
          targetId: "run_1",
          rating: 3,
          status: "triaged",
          category: "quality",
          severity: "normal",
          source: "ui",
          comment: "ok",
          actorId: "tester",
        },
      ],
    });
    renderPage(<FeedbackPage />);
    await waitFor(() => {
      expect(listFeedback).toHaveBeenCalledWith(
        expect.objectContaining({ status: "", category: "", targetType: "", limit: 80 }),
      );
      expect(screen.getByText("run_1")).toBeInTheDocument();
      expect(screen.getByText("1 条")).toBeInTheDocument();
    });
    const triage = screen.getByRole("button", { name: "triage" });
    expect(triage).toBeDisabled();
    expect(triage).toHaveAttribute("title", "已 triage");
  });

  it("disables submit until target id is filled", async () => {
    renderPage(<FeedbackPage />);
    const submit = await screen.findByRole("button", { name: "提交反馈" });
    expect(submit).toBeDisabled();
    expect(submit).toHaveAttribute("title", "需要填写 Target ID");
    fireEvent.change(screen.getByTestId("feedback-target-id"), { target: { value: "run_x" } });
    expect(submit).not.toBeDisabled();
    expect(submit).toHaveAttribute("title", "提交反馈（需确认）");
  });

  it("disables submit when rating is outside 1–5", async () => {
    renderPage(<FeedbackPage />);
    const submit = await screen.findByTestId("feedback-submit");
    fireEvent.change(screen.getByTestId("feedback-target-id"), { target: { value: "run_x" } });
    fireEvent.change(screen.getByTestId("feedback-rating"), { target: { value: "0" } });
    expect(submit).toBeDisabled();
    expect(submit).toHaveAttribute("title", "Rating 需为 1–5");
    fireEvent.change(screen.getByTestId("feedback-rating"), { target: { value: "6" } });
    expect(submit).toBeDisabled();
    expect(submit).toHaveAttribute("title", "Rating 需为 1–5");
    fireEvent.change(screen.getByTestId("feedback-rating"), { target: { value: "4" } });
    expect(submit).not.toBeDisabled();
    expect(submit).toHaveAttribute("title", "提交反馈（需确认）");
  });

  it("surfaces create feedback API errors", async () => {
    vi.mocked(createFeedback).mockRejectedValueOnce(
      new ApiError("INVALID_REQUEST", "targetId is required"),
    );
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(true);
    renderPage(<FeedbackPage />);
    fireEvent.change(screen.getByPlaceholderText("run_..."), { target: { value: "run_x" } });
    fireEvent.click(screen.getByRole("button", { name: "提交反馈" }));
    expect(await screen.findByTestId("feedback-submit-error")).toHaveTextContent("targetId is required");
    confirmSpy.mockRestore();
  });

  it("requires confirm before submitting feedback or changing status", async () => {
    const { updateFeedback } = await import("@/modules/closure/api/closure.api");
    vi.mocked(listFeedback).mockResolvedValue({
      items: [
        {
          id: "fb_open",
          targetType: "run",
          targetId: "run_open",
          rating: 2,
          status: "open",
          category: "quality",
          severity: "normal",
          source: "ui",
          comment: "bad",
        },
      ],
    } as never);
    vi.mocked(createFeedback).mockClear();
    vi.mocked(updateFeedback).mockClear();
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    renderPage(<FeedbackPage />);

    fireEvent.change(await screen.findByTestId("feedback-target-id"), { target: { value: "run_x" } });
    fireEvent.click(screen.getByTestId("feedback-submit"));
    expect(confirmSpy).toHaveBeenCalled();
    expect(createFeedback).not.toHaveBeenCalled();

    const resolve = await screen.findByTestId("feedback-resolve-fb_open");
    expect(resolve).toHaveAttribute("title", "标记为 resolved（需确认）");
    fireEvent.click(resolve);
    expect(updateFeedback).not.toHaveBeenCalled();
    confirmSpy.mockRestore();
  });
});
