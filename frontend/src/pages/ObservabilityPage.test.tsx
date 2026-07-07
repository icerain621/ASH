import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { ObservabilityPage } from "./ObservabilityPage";
import { renderPage } from "@/test/renderPage";

vi.mock("@/modules/closure/api/closure.api", () => ({
  listAlerts: vi.fn().mockResolvedValue({ items: [] }),
  listAlertRules: vi.fn().mockResolvedValue({ items: [] }),
  evaluateAlerts: vi.fn().mockResolvedValue({ evaluated: 0 }),
  putAlertRules: vi.fn(),
  getPrometheusText: vi.fn().mockResolvedValue("# ash metrics\n"),
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
  it("renders observability heading and evaluate alerts control", async () => {
    renderPage(<ObservabilityPage />);
    expect(screen.getByRole("heading", { name: "可观测与告警" })).toBeInTheDocument();
    await waitFor(() => {
      expect(screen.getByRole("button", { name: "评估告警" })).toBeInTheDocument();
      expect(screen.getByText("Space: local")).toBeInTheDocument();
    });
  });
});
