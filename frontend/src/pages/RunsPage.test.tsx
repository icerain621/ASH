import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { RunsPage } from "./RunsPage";
import { renderPage } from "@/test/renderPage";
import { useRunStream } from "@/services/sse/runStream";
import { ApiError } from "@/services/http/client";
import {
  getRun,
  getRunAgentTasks,
  getRunArtifacts,
  getRunCheckpoints,
  getRunProvenance,
  getRunQualityMetrics,
  getRunTimeline,
  getRunToolCalls,
  getRunWaterfall,
  listRuns,
  type RunProvenance,
  type RunWaterfall,
} from "@/modules/runs/api/runs.api";

const emptyWaterfall: RunWaterfall = {
  runId: "",
  traceId: "",
  status: "",
  generatedAt: 0,
  spans: [],
};
const emptyProvenance: RunProvenance = {
  runId: "",
  traceId: "",
  scenario: { name: "", scenarioVersion: "" },
  status: "",
  toolCalls: 0,
  agentTasks: 0,
  artifacts: 0,
  events: 0,
  modelUsage: 0,
  links: [],
};

vi.mock("@/modules/runs/api/runs.api", () => ({
  listRuns: vi.fn().mockResolvedValue({ items: [] }),
  getRun: vi.fn(),
  getRunArtifacts: vi.fn().mockResolvedValue({ artifacts: [] }),
  getRunCheckpoints: vi.fn().mockResolvedValue({ items: [] }),
  getRunTimeline: vi.fn().mockResolvedValue({ items: [] }),
  getRunToolCalls: vi.fn().mockResolvedValue({ items: [] }),
  getRunAgentTasks: vi.fn().mockResolvedValue({ items: [] }),
  getRunQualityMetrics: vi.fn().mockResolvedValue({ items: [] }),
  getRunWaterfall: vi.fn().mockResolvedValue({
    runId: "",
    traceId: "",
    status: "",
    generatedAt: 0,
    spans: [],
  }),
  getRunProvenance: vi.fn().mockResolvedValue({
    runId: "",
    traceId: "",
    scenario: { name: "", scenarioVersion: "" },
    status: "",
    toolCalls: 0,
    agentTasks: 0,
    artifacts: 0,
    events: 0,
    modelUsage: 0,
    links: [],
  }),
  getRunArtifactAccess: vi.fn(),
  getRunCheckpointAccess: vi.fn(),
  createRun: vi.fn(),
  createRunFromGoal: vi.fn(),
  approveGoalPlan: vi.fn(),
  rejectGoalPlan: vi.fn(),
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
  beforeEach(() => {
    vi.mocked(useRunStream).mockReturnValue({ lines: [], status: "idle" });
    vi.mocked(listRuns).mockReset().mockResolvedValue({ items: [] });
    vi.mocked(getRun).mockReset();
    vi.mocked(getRunArtifacts).mockReset().mockResolvedValue({ artifacts: [] });
    vi.mocked(getRunCheckpoints).mockReset().mockResolvedValue({ items: [] });
    vi.mocked(getRunTimeline).mockReset().mockResolvedValue({ items: [] });
    vi.mocked(getRunToolCalls).mockReset().mockResolvedValue({ items: [] });
    vi.mocked(getRunAgentTasks).mockReset().mockResolvedValue({ items: [] });
    vi.mocked(getRunQualityMetrics).mockReset().mockResolvedValue({ items: [] });
    vi.mocked(getRunWaterfall).mockReset().mockResolvedValue(emptyWaterfall);
    vi.mocked(getRunProvenance).mockReset().mockResolvedValue(emptyProvenance);
  });

  it("renders runs console heading and create controls", async () => {
    renderPage(<RunsPage />);
    expect(screen.getByRole("heading", { name: "运行" })).toBeInTheDocument();
    await waitFor(() => {
      expect(screen.getByRole("button", { name: "新建运行" })).toBeInTheDocument();
    });
    expect(screen.getByTestId("quest-pane")).toBeInTheDocument();
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

  it("shows tool_risk gate summary when waiting approval", async () => {
    vi.mocked(useRunStream).mockReturnValue({ lines: [], status: "idle" });
    const runId = "run_gate_tool_risk_01";
    vi.mocked(listRuns).mockResolvedValue({
      items: [
        {
          runId,
          traceId: "trc_1",
          scenario: { name: "hotfix", scenarioVersion: "1.1.0" },
          policyProfile: "hotfix",
          status: "waiting_approval",
          spaceId: "local",
          actorRole: "operator",
          startedAt: Date.now(),
        },
      ],
    });
    vi.mocked(getRun).mockResolvedValue({
      runId,
      traceId: "trc_1",
      scenario: { name: "hotfix", scenarioVersion: "1.1.0" },
      policyProfile: "hotfix",
      status: "waiting_approval",
      spaceId: "local",
      actorRole: "operator",
      startedAt: Date.now(),
    });
    vi.mocked(getRunTimeline).mockResolvedValue({
      items: [
        {
          type: "gate.waiting_approval",
          payload: {
            gate: "tool_risk",
            tool: "runtime.command",
            risk: "danger",
            stepId: "sre.approve_ship",
          },
        },
      ],
    });
    vi.mocked(getRunArtifacts).mockResolvedValue({ artifacts: [] });
    vi.mocked(getRunCheckpoints).mockResolvedValue({ items: [] });
    vi.mocked(getRunToolCalls).mockResolvedValue({ items: [] });
    vi.mocked(getRunAgentTasks).mockResolvedValue({ items: [] });
    vi.mocked(getRunQualityMetrics).mockResolvedValue({ items: [] });
    vi.mocked(getRunWaterfall).mockResolvedValue(emptyWaterfall);
    vi.mocked(getRunProvenance).mockResolvedValue(emptyProvenance);

    renderPage(<RunsPage />);
    await waitFor(() => {
      expect(screen.getByText("待审批")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("待审批"));
    await waitFor(() => {
      const summary = screen.getByTestId("run-gate-summary");
      expect(summary).toHaveAttribute("data-gate", "tool_risk");
      expect(screen.getByText("危险工具审批 · runtime.command")).toBeInTheDocument();
      expect(screen.getByText(/工具风险级别 danger/)).toBeInTheDocument();
    });
  });
});
