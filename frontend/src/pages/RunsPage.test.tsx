import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { RunsPage } from "./RunsPage";
import { renderPage } from "@/test/renderPage";

vi.mock("@/modules/runs/api/runs.api", () => ({
  listRuns: vi.fn().mockResolvedValue({ items: [] }),
  getRun: vi.fn(),
  getRunArtifacts: vi.fn(),
  getRunCheckpoints: vi.fn(),
  getRunTimeline: vi.fn(),
  getRunToolCalls: vi.fn(),
  getRunAgentTasks: vi.fn(),
  getRunQualityMetrics: vi.fn(),
  getRunWaterfall: vi.fn(),
  getRunProvenance: vi.fn(),
  getRunArtifactAccess: vi.fn(),
  getRunCheckpointAccess: vi.fn(),
  createRun: vi.fn(),
  cancelRun: vi.fn(),
  approveRun: vi.fn(),
  resumeRun: vi.fn(),
  replayRun: vi.fn(),
}));

vi.mock("@/modules/scenarios/api/scenarios.api", () => ({
  listScenarios: vi.fn().mockResolvedValue({
    items: [{ name: "feature_delivery", scenarioVersion: "1.0.0", title: "Feature delivery" }],
  }),
}));

vi.mock("@/services/sse/runStream", () => ({
  useRunStream: () => [],
}));

describe("RunsPage", () => {
  it("renders runs console heading and create controls", async () => {
    renderPage(<RunsPage />);
    expect(screen.getByRole("heading", { name: "运行" })).toBeInTheDocument();
    await waitFor(() => {
      expect(screen.getByRole("button", { name: "新建运行" })).toBeInTheDocument();
    });
  });
});
