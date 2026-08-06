import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { RunsPage } from "./RunsPage";
import { renderPage } from "@/test/renderPage";
import { useRunStream } from "@/services/sse/runStream";
import { ApiError } from "@/services/http/client";
import { listRuns } from "@/modules/runs/api/runs.api";

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
  useRunStream: vi.fn(() => ({ lines: [], status: "idle" as const })),
}));

describe("RunsPage", () => {
  it("renders runs console heading and create controls", async () => {
    vi.mocked(useRunStream).mockReturnValue({ lines: [], status: "idle" });
    renderPage(<RunsPage />);
    expect(screen.getByRole("heading", { name: "运行" })).toBeInTheDocument();
    await waitFor(() => {
      expect(screen.getByRole("button", { name: "新建运行" })).toBeInTheDocument();
    });
  });

  it("shows SSE reconnecting status label", async () => {
    vi.mocked(useRunStream).mockReturnValue({ lines: [], status: "reconnecting" });
    renderPage(<RunsPage />);
    await waitFor(() => {
      expect(screen.getByText(/重连中/)).toBeInTheDocument();
    });
  });

  it("shows SSE polling fallback status label", async () => {
    vi.mocked(useRunStream).mockReturnValue({ lines: [], status: "polling" });
    renderPage(<RunsPage />);
    await waitFor(() => {
      expect(screen.getByText(/轮询回退/)).toBeInTheDocument();
    });
  });

  it("shows ApiError control codes in error banner", async () => {
    vi.mocked(useRunStream).mockReturnValue({ lines: [], status: "idle" });
    vi.mocked(listRuns).mockRejectedValueOnce(
      new ApiError("RUN_NOT_REPLAYABLE", "run is not in a replayable status"),
    );
    renderPage(<RunsPage />);
    await waitFor(() => {
      expect(
        screen.getByText(/RUN_NOT_REPLAYABLE: run is not in a replayable status/),
      ).toBeInTheDocument();
    });
  });
});
