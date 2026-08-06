import { screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ObservabilityPage } from "./ObservabilityPage";
import { renderPage } from "@/test/renderPage";
import { getPrometheusText, listAlertRules } from "@/modules/closure/api/closure.api";

vi.mock("@/modules/closure/api/closure.api", () => ({
  listAlerts: vi.fn().mockResolvedValue({ items: [] }),
  listAlertRules: vi.fn(),
  evaluateAlerts: vi.fn().mockResolvedValue({ evaluated: 0 }),
  putAlertRules: vi.fn(),
  getPrometheusText: vi.fn(),
  getTrace: vi.fn(),
}));

vi.mock("@/modules/platform/api/platform.api", () => ({
  getPluginHealth: vi.fn().mockResolvedValue({ items: [] }),
}));

vi.mock("@/modules/observability/api/observability.api", () => ({
  getOtelStatus: vi.fn().mockResolvedValue({ enabled: false, exporter: "none" }),
  getRagProfile: vi.fn().mockResolvedValue({ retrievalMode: "fts", documentCount: 0 }),
}));

vi.mock("@/modules/scale/api/scale.api", () => ({
  getScaleReadiness: vi.fn().mockResolvedValue({
    spaceId: "local",
    migrationReady: true,
    databaseDialect: "sqlite",
  }),
}));

describe("ObservabilityPage", () => {
  beforeEach(() => {
    vi.mocked(listAlertRules).mockResolvedValue({
      items: [
        {
          id: "rule_inflight",
          name: "运行 inflight 积压",
          metric: "run_inflight_count",
          condition: "gt",
          threshold: 20,
          windowMinutes: 60,
          severity: "warn",
          enabled: true,
          description: "status=running/waiting_approval 的运行数超过阈值（Scale backlog）",
        },
      ],
    });
    vi.mocked(getPrometheusText).mockResolvedValue(
      '# HELP ash_run_inflight_live\nash_run_inflight_live{space_id="local"} 3\n',
    );
  });

  it("renders observability heading and evaluate alerts control", async () => {
    renderPage(<ObservabilityPage />);
    expect(screen.getByRole("heading", { name: "可观测与告警" })).toBeInTheDocument();
    await waitFor(() => {
      expect(screen.getByRole("button", { name: "评估告警" })).toBeInTheDocument();
      expect(screen.getByText("Space: local")).toBeInTheDocument();
    });
  });

  it("surfaces run_inflight_count in governance alert rules", async () => {
    renderPage(<ObservabilityPage />);
    await waitFor(() => {
      expect(screen.getAllByText("run_inflight_count").length).toBeGreaterThanOrEqual(1);
      expect(screen.getByText("running / waiting_approval 运行数（Scale backlog）")).toBeInTheDocument();
      expect(screen.getByText("1 条")).toBeInTheDocument();
    });
    expect(screen.getByText(/ash_run_inflight_live/)).toBeInTheDocument();
  });
});
