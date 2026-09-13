import { render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  ImproveProposalsPane,
  isLowScoreTriggered,
  lowScoreSourceLabel,
} from "./ImproveProposalsPane";

const listImproveProposals = vi.fn();

vi.mock("@/modules/improve/api/improve.api", () => ({
  listImproveProposals: (...a: unknown[]) => listImproveProposals(...a),
  createImproveProposal: vi.fn(),
  startImproveExperiment: vi.fn(),
  startImproveCanary: vi.fn(),
  promoteImproveProposal: vi.fn(),
  rollbackImproveProposal: vi.fn(),
}));

describe("ImproveProposalsPane helpers", () => {
  it("detects low_score from source, scoreEventId, or changeSummary", () => {
    expect(isLowScoreTriggered({ source: "low_score" })).toBe(true);
    expect(isLowScoreTriggered({ scoreEventId: "sev_1" })).toBe(true);
    expect(isLowScoreTriggered({ changeSummary: "source=low_score" })).toBe(true);
    expect(isLowScoreTriggered({ source: "manual", changeSummary: "M1 replay" })).toBe(false);
  });

  it("formats muted source line", () => {
    expect(lowScoreSourceLabel({ scoreEventId: "sev_abcdef123456" })).toMatch(/由低分触发 · scoreEvent/);
    expect(lowScoreSourceLabel({})).toBe("由低分触发");
  });
});

describe("ImproveProposalsPane", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    listImproveProposals.mockResolvedValue({
      items: [
        {
          id: "imp_low",
          title: "Low score draft",
          baselineRunId: "run_base",
          status: "draft",
          canaryPercent: 0,
          source: "low_score",
          scoreEventId: "sev_abc123456789",
          changeSummary: "source=low_score",
        },
        {
          id: "imp_manual",
          title: "Manual proposal",
          baselineRunId: "run_base2",
          status: "draft",
          canaryPercent: 0,
          changeSummary: "M1 replay compare",
        },
      ],
    });
  });

  it("shows muted low_score source line for auto drafts", async () => {
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={qc}>
        <ImproveProposalsPane />
      </QueryClientProvider>,
    );
    await waitFor(() => expect(screen.getByTestId("improve-low-score-source")).toBeInTheDocument());
    expect(screen.getByTestId("improve-low-score-source")).toHaveTextContent("由低分触发");
    expect(screen.getByTestId("improve-low-score-source")).toHaveTextContent("scoreEvent");
    expect(screen.queryByText("Manual proposal")?.closest("td")?.querySelector("[data-testid=improve-low-score-source]")).toBeNull();
  });
});
