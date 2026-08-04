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
    summary: [{ id: "KPI-01", label: "任务成功率", value: 1, unit: "ratio", status: "ok", denominator: 1 }],
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
});
