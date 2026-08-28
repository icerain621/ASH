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
  getRunDiff: vi.fn(),
  listDiffComments: vi.fn(),
  createDiffComment: vi.fn(),
  rateRunStep: vi.fn(),
}));

vi.mock("@/modules/runs/api/runs.api", () => ({
  getRunTimeline: vi.fn(async () => ({ items: [] })),
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
});
