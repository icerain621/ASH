import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, expect, it, vi } from "vitest";
import { QuestPage } from "./QuestPage";

vi.mock("@/modules/quest/api/quest.api", () => ({
  getQuestBoard: vi.fn(async () => ({
    columns: {
      plans: [{ id: "gplan_1", kind: "plan", title: "Add feature", status: "draft", column: "plans", spaceId: "local", updatedAt: 1 }],
      running: [],
      waiting_approval: [],
      finished: [{ id: "run_1", kind: "run", title: "feature_delivery@1.0.0", status: "finished", column: "finished", runId: "run_1", spaceId: "local", updatedAt: 2 }],
    },
  })),
  getRunDiff: vi.fn(async () => ({ runId: "run_1", raw: "", files: [], contextRefs: [] })),
  listDiffComments: vi.fn(async () => ({ items: [] })),
  createDiffComment: vi.fn(),
  rateRunStep: vi.fn(),
}));

vi.mock("@/modules/runs/api/runs.api", () => ({
  getRunTimeline: vi.fn(async () => ({ items: [] })),
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
}));

vi.mock("@/services/http/client", () => ({
  getCurrentSpaceId: () => "local",
}));

describe("QuestPage", () => {
  it("renders kanban board", async () => {
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={qc}>
        <QuestPage />
      </QueryClientProvider>,
    );
    expect(await screen.findByTestId("quest-page")).toBeTruthy();
    expect(await screen.findByTestId("quest-board")).toBeTruthy();
    expect(await screen.findByText("Add feature")).toBeTruthy();
  });

  it("shows sub-run tree after selecting a run", async () => {
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={qc}>
        <QuestPage />
      </QueryClientProvider>,
    );
    (await screen.findByText("feature_delivery@1.0.0")).click();
    expect(await screen.findByTestId("quest-run-tree")).toBeTruthy();
    expect(await screen.findByTestId("quest-run-tree-list")).toBeTruthy();
  });
});
