import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { MetricsPage } from "./MetricsPage";
import { renderPage } from "@/test/renderPage";

vi.mock("@/modules/metrics/api/metrics.api", () => ({
  getMetricsOverview: vi.fn().mockResolvedValue({
    spaceId: "local",
    from: "2026-07-01T00:00:00Z",
    to: "2026-07-07T00:00:00Z",
    period: "day",
    summary: [
      { id: "KPI-01", label: "任务成功率", value: 1, unit: "ratio", status: "ok", denominator: 1 },
      { id: "KPI-11", label: "场景稳定率", value: 0.5, unit: "ratio", status: "ok", numerator: 1, denominator: 2 },
    ],
    breakdowns: [
      {
        id: "scenarioStability",
        label: "场景可重复性 (R-02)",
        items: [
          { key: "feature_delivery@1.0.0", label: "feature_delivery@1.0.0 成功率", value: 0.67, unit: "ratio" },
          { key: "feature_delivery@1.0.0:n", label: "feature_delivery@1.0.0 样本", value: 3, unit: "count" },
        ],
      },
    ],
  }),
}));

describe("MetricsPage", () => {
  it("renders metrics heading and refresh control", async () => {
    renderPage(<MetricsPage />);
    expect(screen.getByRole("heading", { name: "指标看板" })).toBeInTheDocument();
    await waitFor(() => {
      expect(screen.getByRole("button", { name: "刷新" })).toBeInTheDocument();
      expect(screen.getByText("Space: local")).toBeInTheDocument();
    });
  });

  it("renders KPI-11 and scenarioStability breakdown (R-02)", async () => {
    renderPage(<MetricsPage />);
    await waitFor(() => {
      expect(screen.getByText("场景稳定率")).toBeInTheDocument();
      expect(screen.getByTestId("metrics-breakdown-scenarioStability")).toBeInTheDocument();
      expect(screen.getByText(/低于门槛/)).toBeInTheDocument();
    });
  });
});
