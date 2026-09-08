import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { QuestPage } from "./QuestPage";
import {
  approveGoalPlan,
  approveRun,
  cancelRun,
  createRunFromGoal,
  getGoalPlan,
  getRunArtifactAccess,
  getRunArtifacts,
  rejectGoalPlan,
} from "@/modules/runs/api/runs.api";
import { rejectRunDiff } from "@/modules/quest/api/quest.api";

vi.mock("@/modules/quest/api/quest.api", () => ({
  getQuestBoard: vi.fn(async () => ({
    columns: {
      plans: [
        {
          id: "gplan_1",
          kind: "plan",
          title: "Add feature",
          status: "draft",
          column: "plans",
          planId: "gplan_1",
          spaceId: "local",
          updatedAt: 1,
        },
      ],
      running: [],
      waiting_approval: [],
      finished: [
        {
          id: "run_1",
          kind: "run",
          title: "feature_delivery@1.0.0",
          status: "finished",
          column: "finished",
          runId: "run_1",
          spaceId: "local",
          updatedAt: 2,
        },
      ],
    },
  })),
  getRunDiff: vi.fn(async () => ({
    runId: "run_1",
    raw: "diff --git a/a.go b/a.go\n",
    files: [{ path: "a.go", hunks: [{ header: "@@", oldStart: 1, newStart: 1, lines: [{ kind: "add", text: "+x", index: 0 }] }] }],
    contextRefs: [],
    rejectedPaths: [],
  })),
  listDiffComments: vi.fn(async () => ({ items: [] })),
  createDiffComment: vi.fn(),
  rateRunStep: vi.fn(),
  rejectRunDiff: vi.fn(),
}));

vi.mock("@/modules/runs/api/runs.api", () => ({
  getRunTimeline: vi.fn(async () => ({
    items: [
      {
        type: "gate.waiting_approval",
        payload: { gate: "human", reason: "need human review", stepId: "s_review" },
      },
    ],
  })),
  getRunTree: vi.fn(async () => ({
    rootRunId: "run_1",
    tree: {
      summary: {
        runId: "run_1",
        traceId: "trc_1",
        scenario: { name: "feature_delivery", scenarioVersion: "1.0.0" },
        policyProfile: "default",
        status: "finished",
        startedAt: 1,
        depth: 0,
      },
      children: [],
    },
  })),
  getRun: vi.fn(async () => ({
    runId: "run_1",
    traceId: "trc_1",
    scenario: { name: "feature_delivery", scenarioVersion: "1.0.0" },
    policyProfile: "default",
    status: "waiting_approval",
    startedAt: 1,
  })),
  getRunArtifacts: vi.fn(async () => ({
    artifacts: [{ type: "diff", name: "diff.patch", uri: "artifacts/diff.patch", digest: "sha256:abc" }],
  })),
  getRunArtifactAccess: vi.fn(async () => ({
    runId: "run_1",
    name: "diff.patch",
    uri: "artifacts/diff.patch",
    signedUrl: "https://example.test/diff.patch",
    expiresAt: 1,
    digest: "sha256:abc",
  })),
  createRunFromGoal: vi.fn(),
  getGoalPlan: vi.fn(),
  approveGoalPlan: vi.fn(),
  rejectGoalPlan: vi.fn(),
  approveRun: vi.fn(),
  cancelRun: vi.fn(),
}));

vi.mock("@/services/http/client", () => ({
  getCurrentSpaceId: () => "local",
}));

class MockEventSource {
  static instances: MockEventSource[] = [];
  url: string;
  onopen: (() => void) | null = null;
  onmessage: ((ev: MessageEvent) => void) | null = null;
  onerror: (() => void) | null = null;
  constructor(url: string) {
    this.url = url;
    MockEventSource.instances.push(this);
  }
  addEventListener() {}
  close() {}
}

function renderQuest() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <QuestPage />
    </QueryClientProvider>,
  );
}

describe("QuestPage", () => {
  beforeEach(() => {
    MockEventSource.instances = [];
    vi.stubGlobal("EventSource", MockEventSource as unknown as typeof EventSource);
    vi.mocked(createRunFromGoal).mockReset();
    vi.mocked(getGoalPlan).mockReset();
    vi.mocked(approveGoalPlan).mockReset();
    vi.mocked(rejectGoalPlan).mockReset();
    vi.mocked(approveRun).mockReset();
    vi.mocked(cancelRun).mockReset();
    vi.mocked(rejectRunDiff).mockReset();
    vi.mocked(getRunArtifacts).mockClear();
    vi.mocked(getRunArtifactAccess).mockReset();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("renders kanban board", async () => {
    renderQuest();
    expect(await screen.findByTestId("quest-page")).toBeTruthy();
    expect(await screen.findByTestId("quest-board")).toBeTruthy();
    expect(await screen.findByText("Add feature")).toBeTruthy();
    expect(await screen.findByTestId("quest-stream-status")).toBeTruthy();
  });

  it("shows compose pane for Goal→Plan", async () => {
    renderQuest();
    expect(await screen.findByTestId("quest-wb-compose")).toBeTruthy();
    expect(screen.getByTestId("quest-wb-goal-input")).toBeTruthy();
    expect(screen.getByTestId("quest-wb-route")).toBeTruthy();
  });

  it("routes goal to plan preview", async () => {
    vi.mocked(createRunFromGoal).mockResolvedValue({
      id: "gplan_new",
      spaceId: "local",
      goal: "Add dark mode",
      scenarioName: "feature_delivery",
      scenarioVersion: "1.0.0",
      policyProfile: "default",
      inputs: {},
      steps: [{ id: "s1", role: "dev", kind: "agent" }],
      status: "draft",
      createdAt: 1,
    });
    renderQuest();
    fireEvent.change(await screen.findByTestId("quest-wb-goal-input"), {
      target: { value: "Add dark mode" },
    });
    fireEvent.click(screen.getByTestId("quest-wb-route"));
    await waitFor(() => {
      expect(createRunFromGoal).toHaveBeenCalled();
      expect(screen.getByTestId("quest-wb-plan-preview")).toBeTruthy();
    });
  });

  it("approves plan from Quest workbench", async () => {
    vi.mocked(createRunFromGoal).mockResolvedValue({
      id: "gplan_new",
      spaceId: "local",
      goal: "Add dark mode",
      scenarioName: "feature_delivery",
      scenarioVersion: "1.0.0",
      policyProfile: "default",
      inputs: {},
      steps: [{ id: "s1", role: "dev", kind: "agent" }],
      status: "draft",
      createdAt: 1,
    });
    vi.mocked(approveGoalPlan).mockResolvedValue({
      id: "gplan_new",
      spaceId: "local",
      goal: "Add dark mode",
      scenarioName: "feature_delivery",
      scenarioVersion: "1.0.0",
      policyProfile: "default",
      inputs: {},
      steps: [{ id: "s1", role: "dev", kind: "agent" }],
      status: "approved",
      runId: "run_new",
      createdAt: 1,
    });
    renderQuest();
    fireEvent.change(await screen.findByTestId("quest-wb-goal-input"), {
      target: { value: "Add dark mode" },
    });
    fireEvent.click(screen.getByTestId("quest-wb-route"));
    await screen.findByTestId("quest-wb-approve");
    fireEvent.click(screen.getByTestId("quest-wb-approve"));
    await waitFor(() => {
      expect(approveGoalPlan).toHaveBeenCalledWith(
        "gplan_new",
        expect.objectContaining({ actorId: "console" }),
      );
    });
  });

  it("loads plan preview when selecting a Plans card", async () => {
    vi.mocked(getGoalPlan).mockResolvedValue({
      id: "gplan_1",
      spaceId: "local",
      goal: "Add feature",
      scenarioName: "feature_delivery",
      scenarioVersion: "1.0.0",
      policyProfile: "default",
      inputs: {},
      steps: [{ id: "s1", role: "dev", kind: "agent" }],
      status: "draft",
      createdAt: 1,
    });
    renderQuest();
    fireEvent.click(await screen.findByText("Add feature"));
    await waitFor(() => {
      expect(getGoalPlan).toHaveBeenCalledWith("gplan_1");
      expect(screen.getByTestId("quest-wb-plan-preview")).toBeTruthy();
    });
    expect(screen.getByTestId("quest-wb-approve")).toBeTruthy();
    expect(screen.queryByText(/在 Runs 页批准/)).toBeNull();
  });

  it("shows sub-run tree after selecting a run", async () => {
    renderQuest();
    (await screen.findByText("feature_delivery@1.0.0")).click();
    expect(await screen.findByTestId("quest-run-tree")).toBeTruthy();
    expect(await screen.findByTestId("quest-run-tree-list")).toBeTruthy();
  });

  it("rejects current file from Diff actions", async () => {
    vi.mocked(rejectRunDiff).mockResolvedValue({
      runId: "run_1",
      scope: "file",
      filePath: "a.go",
      comment: {
        id: "drc_1",
        runId: "run_1",
        filePath: "a.go",
        lineIndex: -1,
        side: "reject",
        body: "reject file a.go",
        createdAt: 1,
      },
      canceled: false,
      status: "waiting_approval",
    });
    renderQuest();
    (await screen.findByText("feature_delivery@1.0.0")).click();
    const rejectFileBtn = await screen.findByTestId("quest-diff-reject-file");
    await waitFor(() => {
      expect(rejectFileBtn).not.toBeDisabled();
    });
    fireEvent.click(rejectFileBtn);
    await waitFor(() => {
      expect(rejectRunDiff).toHaveBeenCalledWith(
        "run_1",
        expect.objectContaining({ scope: "file", filePath: "a.go" }),
      );
    });
  });

  it("approves waiting gate from Diff actions", async () => {
    vi.mocked(approveRun).mockResolvedValue({ runId: "run_1", ok: true });
    renderQuest();
    (await screen.findByText("feature_delivery@1.0.0")).click();
    const approveBtn = await screen.findByTestId("quest-diff-approve-gate");
    await waitFor(() => {
      expect(approveBtn).not.toBeDisabled();
    });
    fireEvent.click(approveBtn);
    await waitFor(() => {
      expect(approveRun).toHaveBeenCalled();
    });
  });

  it("shows gate panel with approve and cancel for waiting_approval", async () => {
    vi.mocked(approveRun).mockResolvedValue({ runId: "run_1", ok: true });
    vi.mocked(cancelRun).mockResolvedValue({ runId: "run_1", status: "canceled" });
    renderQuest();
    (await screen.findByText("feature_delivery@1.0.0")).click();
    expect(await screen.findByTestId("quest-gate-panel")).toBeTruthy();
    expect(screen.getByTestId("quest-gate-detail").textContent).toContain("need human review");
    fireEvent.click(screen.getByTestId("quest-gate-approve"));
    await waitFor(() => {
      expect(approveRun).toHaveBeenCalled();
    });
    fireEvent.click(screen.getByTestId("quest-gate-cancel"));
    await waitFor(() => {
      expect(cancelRun).toHaveBeenCalledWith("run_1");
    });
  });

  it("lists artifacts and issues access link", async () => {
    vi.mocked(getRunArtifactAccess).mockResolvedValue({
      runId: "run_1",
      name: "diff.patch",
      uri: "artifacts/diff.patch",
      signedUrl: "https://example.test/diff.patch",
      expiresAt: 1,
      digest: "sha256:abc",
    });
    renderQuest();
    (await screen.findByText("feature_delivery@1.0.0")).click();
    expect(await screen.findByTestId("quest-artifacts-pane")).toBeTruthy();
    await waitFor(() => {
      expect(getRunArtifacts).toHaveBeenCalledWith("run_1");
    });
    fireEvent.click(await screen.findByTestId("quest-artifact-link-diff.patch"));
    await waitFor(() => {
      expect(getRunArtifactAccess).toHaveBeenCalledWith("run_1", "diff.patch");
      expect(screen.getByTestId("quest-artifact-access").textContent).toContain(
        "https://example.test/diff.patch",
      );
    });
  });
});
