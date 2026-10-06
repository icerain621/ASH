import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { RunsPage } from "./RunsPage";
import { renderPage } from "@/test/renderPage";
import { useRunStream } from "@/services/sse/runStream";
import { ApiError } from "@/services/http/client";
import {
  approveRun,
  cancelRun,
  createRun,
  createRunFromGoal,
  approveGoalPlan,
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
  replayRun,
  resumeRun,
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

  it("shows artifact load failure instead of an empty list", async () => {
    const runId = "run_artifacts_err";
    const summary = {
      runId,
      traceId: "trc_art",
      scenario: { name: "hotfix", scenarioVersion: "1.1.0" },
      policyProfile: "hotfix",
      status: "failed",
      spaceId: "local",
      actorRole: "operator",
      startedAt: Date.now(),
    };
    vi.mocked(listRuns).mockResolvedValue({ items: [summary] });
    vi.mocked(getRun).mockResolvedValue(summary);
    vi.mocked(getRunArtifacts).mockRejectedValue(new ApiError("ARTIFACTS_NOT_FOUND", "manifest unreadable"));
    renderPage(<RunsPage />);
    fireEvent.click(await screen.findByText("失败"));
    expect(await screen.findByText("产物加载失败。")).toBeInTheDocument();
    expect(screen.queryByText("暂无产物。")).not.toBeInTheDocument();
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

  it("renders duplicate provenance links without crashing", async () => {
    const runId = "run_prov_dup";
    const summary = {
      runId,
      traceId: "trc_dup",
      scenario: { name: "feature_delivery", scenarioVersion: "1.0.0" },
      policyProfile: "default",
      status: "failed",
      spaceId: "local",
      actorRole: "maintainer",
      startedAt: Date.now(),
    };
    vi.mocked(listRuns).mockResolvedValue({ items: [summary] });
    vi.mocked(getRun).mockResolvedValue(summary);
    vi.mocked(getRunProvenance).mockResolvedValue({
      ...emptyProvenance,
      runId,
      traceId: "trc_dup",
      links: [
        { kind: "model", ref: "primary/not-configured-not_configured" },
        { kind: "model", ref: "primary/not-configured-not_configured" },
      ],
    });
    vi.mocked(getRunWaterfall).mockResolvedValue({
      ...emptyWaterfall,
      runId,
      failures: [
        { type: "model", ref: "primary/not-configured", code: "not_configured" },
        { type: "model", ref: "primary/not-configured", code: "not_configured" },
      ],
    });
    renderPage(<RunsPage />);
    await waitFor(() => {
      expect(screen.getByText(runId.slice(0, 12), { exact: false })).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText(/失败/));
    await waitFor(() => {
      expect(screen.getAllByTestId("provenance-link-row")).toHaveLength(2);
      expect(screen.getAllByTestId("waterfall-failure-item")).toHaveLength(2);
    });
  });

  it("selects run from ?runId= deep link", async () => {
    const runId = "run_deep_link_01";
    const summary = {
      runId,
      traceId: "trc_deep",
      scenario: { name: "hotfix", scenarioVersion: "1.1.0" },
      policyProfile: "hotfix",
      status: "failed",
      spaceId: "local",
      actorRole: "operator",
      startedAt: Date.now(),
    };
    window.history.replaceState({}, "", `/ui/runs?runId=${runId}`);
    vi.mocked(listRuns).mockResolvedValue({ items: [summary] });
    vi.mocked(getRun).mockResolvedValue(summary);
    renderPage(<RunsPage />);
    await waitFor(() => {
      expect(getRun).toHaveBeenCalledWith(runId);
    });
    expect(screen.getByText("运行详情")).toBeInTheDocument();
    expect(screen.getByText(runId.slice(0, 12), { exact: false })).toBeInTheDocument();
  });

  it("surfaces missing run deep-link instead of a blank select prompt", async () => {
    window.history.replaceState({}, "", "/ui/runs?runId=run_missing_zzz");
    vi.mocked(listRuns).mockResolvedValue({ items: [] });
    vi.mocked(getRun).mockRejectedValue(new ApiError("NOT_FOUND", "run not found"));
    renderPage(<RunsPage />);
    expect(await screen.findByTestId("runs-detail-error")).toHaveTextContent(
      "未找到该 Run，请检查 Run ID",
    );
    expect(screen.queryByText("选择一条运行记录")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "重放" })).not.toBeInTheDocument();
    expect(screen.getByText(/无法操作：运行详情加载失败/)).toBeInTheDocument();
    // page banner must not repeat the same detail error
    expect(screen.getAllByText("未找到该 Run，请检查 Run ID")).toHaveLength(1);
    await waitFor(() => {
      expect(useRunStream).toHaveBeenCalledWith(null);
    });
  });

  it("maps invalid token shape to a login hint on deep-link", async () => {
    window.history.replaceState({}, "", "/ui/runs?runId=run_any");
    vi.mocked(listRuns).mockResolvedValue({ items: [] });
    vi.mocked(getRun).mockRejectedValue(new ApiError("UNAUTHORIZED", "invalid token shape"));
    renderPage(<RunsPage />);
    expect(await screen.findByTestId("runs-detail-error")).toHaveTextContent(
      "登录已失效或令牌无效，请重新登录后再查看运行",
    );
  });

  it("requires confirm before replay", async () => {
    const runId = "run_confirm_replay_01";
    const summary = {
      runId,
      traceId: "trc_r",
      scenario: { name: "hotfix", scenarioVersion: "1.1.0" },
      policyProfile: "hotfix",
      status: "failed",
      spaceId: "local",
      actorRole: "operator",
      startedAt: Date.now(),
    };
    window.history.replaceState({}, "", `/ui/runs?runId=${runId}`);
    vi.mocked(listRuns).mockResolvedValue({ items: [summary] });
    vi.mocked(getRun).mockResolvedValue(summary);
    vi.mocked(replayRun).mockResolvedValue({ runId: "run_replayed" } as never);

    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    renderPage(<RunsPage />);
    const replay = await screen.findByTestId("runs-replay");
    expect(replay).toHaveAttribute("title", "重放此运行（需确认）");
    fireEvent.click(replay);
    expect(replayRun).not.toHaveBeenCalled();

    confirmSpy.mockReturnValue(true);
    fireEvent.click(replay);
    await waitFor(() => {
      expect(replayRun).toHaveBeenCalledWith(runId, { mode: "exact" });
    });
    confirmSpy.mockRestore();
  });

  it("requires confirm before cancel", async () => {
    const runId = "run_confirm_cancel_01";
    const summary = {
      runId,
      traceId: "trc_c",
      scenario: { name: "hotfix", scenarioVersion: "1.1.0" },
      policyProfile: "hotfix",
      status: "running",
      spaceId: "local",
      actorRole: "operator",
      startedAt: Date.now(),
    };
    window.history.replaceState({}, "", `/ui/runs?runId=${runId}`);
    vi.mocked(listRuns).mockResolvedValue({ items: [summary] });
    vi.mocked(getRun).mockResolvedValue(summary);
    vi.mocked(cancelRun).mockResolvedValue({ runId, status: "canceled" } as never);

    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    renderPage(<RunsPage />);
    const cancel = await screen.findByTestId("runs-cancel");
    expect(cancel).toHaveAttribute("title", "取消运行（需确认）");
    fireEvent.click(cancel);
    expect(cancelRun).not.toHaveBeenCalled();

    confirmSpy.mockReturnValue(true);
    fireEvent.click(cancel);
    await waitFor(() => {
      expect(cancelRun).toHaveBeenCalledWith(runId);
    });
    confirmSpy.mockRestore();
  });

  it("requires confirm before resume", async () => {
    const runId = "run_confirm_resume_01";
    const summary = {
      runId,
      traceId: "trc_res",
      scenario: { name: "hotfix", scenarioVersion: "1.1.0" },
      policyProfile: "hotfix",
      status: "failed",
      spaceId: "local",
      actorRole: "operator",
      startedAt: Date.now(),
    };
    window.history.replaceState({}, "", `/ui/runs?runId=${runId}`);
    vi.mocked(listRuns).mockResolvedValue({ items: [summary] });
    vi.mocked(getRun).mockResolvedValue(summary);
    vi.mocked(resumeRun).mockResolvedValue({ runId, status: "running" } as never);

    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    renderPage(<RunsPage />);
    const resume = await screen.findByTestId("runs-resume");
    expect(resume).toHaveAttribute("title", "从失败点继续（需确认）");
    fireEvent.click(resume);
    expect(resumeRun).not.toHaveBeenCalled();

    confirmSpy.mockReturnValue(true);
    fireEvent.click(resume);
    await waitFor(() => {
      expect(resumeRun).toHaveBeenCalledWith(runId);
    });
    confirmSpy.mockRestore();
  });

  it("requires confirm before approve gate", async () => {
    const runId = "run_confirm_approve_01";
    const summary = {
      runId,
      traceId: "trc_ap",
      scenario: { name: "hotfix", scenarioVersion: "1.1.0" },
      policyProfile: "hotfix",
      status: "waiting_approval",
      spaceId: "local",
      actorRole: "operator",
      startedAt: Date.now(),
    };
    window.history.replaceState({}, "", `/ui/runs?runId=${runId}`);
    vi.mocked(listRuns).mockResolvedValue({ items: [summary] });
    vi.mocked(getRun).mockResolvedValue(summary);
    vi.mocked(approveRun).mockResolvedValue({ runId, ok: true } as never);

    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    renderPage(<RunsPage />);
    const approve = await screen.findByTestId("runs-approve");
    expect(approve).toHaveAttribute("title", "批准通过门禁（需确认）");
    fireEvent.click(approve);
    expect(approveRun).not.toHaveBeenCalled();

    confirmSpy.mockReturnValue(true);
    fireEvent.click(approve);
    await waitFor(() => {
      expect(approveRun).toHaveBeenCalledWith(
        runId,
        expect.objectContaining({ actorId: "console", reason: "Approved from ASH Console" }),
      );
    });
    confirmSpy.mockRestore();
  });

  it("requires confirm before creating a run", async () => {
    window.history.replaceState({}, "", "/ui/runs");
    vi.mocked(createRun).mockResolvedValue({ runId: "run_new" } as never);
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    renderPage(<RunsPage />);
    const create = await screen.findByTestId("runs-create");
    await waitFor(() => {
      expect(create).toHaveAttribute("title", "按所选场景新建运行（需确认）");
    });
    fireEvent.click(create);
    expect(confirmSpy).toHaveBeenCalled();
    expect(createRun).not.toHaveBeenCalled();

    confirmSpy.mockReturnValue(true);
    fireEvent.click(create);
    await waitFor(() => {
      expect(createRun).toHaveBeenCalled();
    });
    confirmSpy.mockRestore();
  });

  it("requires confirm before approving a quest plan", async () => {
    window.history.replaceState({}, "", "/ui/runs");
    vi.mocked(createRunFromGoal).mockResolvedValue({
      id: "plan_1",
      status: "draft",
      scenarioName: "feature_delivery",
      scenarioVersion: "1.0.0",
      routeReason: "test",
      steps: [],
      inputs: {},
    } as never);
    vi.mocked(approveGoalPlan).mockResolvedValue({ id: "plan_1", status: "approved", runId: "run_q" } as never);

    renderPage(<RunsPage />);
    fireEvent.change(screen.getByTestId("quest-goal-input"), { target: { value: "fix payment" } });
    const route = screen.getByTestId("quest-route");
    expect(route).toHaveAttribute("title", "根据 Goal 生成 Plan（需确认）");
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    fireEvent.click(route);
    expect(createRunFromGoal).not.toHaveBeenCalled();
    confirmSpy.mockReturnValue(true);
    fireEvent.click(route);
    const approve = await screen.findByTestId("quest-approve");
    expect(approve).toHaveAttribute("title", "批准 Plan 并启动 Run（需确认）");

    confirmSpy.mockReturnValue(false);
    fireEvent.click(approve);
    expect(approveGoalPlan).not.toHaveBeenCalled();

    confirmSpy.mockReturnValue(true);
    fireEvent.click(approve);
    await waitFor(() => {
      expect(approveGoalPlan).toHaveBeenCalledWith(
        "plan_1",
        expect.objectContaining({ actorId: "console" }),
      );
    });
    confirmSpy.mockRestore();
  });
});
