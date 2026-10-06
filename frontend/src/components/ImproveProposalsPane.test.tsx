import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  ImproveProposalsPane,
  isLowScoreTriggered,
  lowScoreSourceLabel,
} from "./ImproveProposalsPane";

const listImproveProposals = vi.fn();
const createImproveProposal = vi.fn();
const startImproveExperiment = vi.fn();
const rollbackImproveProposal = vi.fn();

vi.mock("@/modules/improve/api/improve.api", () => ({
  listImproveProposals: (...a: unknown[]) => listImproveProposals(...a),
  createImproveProposal: (...a: unknown[]) => createImproveProposal(...a),
  startImproveExperiment: (...a: unknown[]) => startImproveExperiment(...a),
  startImproveCanary: vi.fn(),
  promoteImproveProposal: vi.fn(),
  rollbackImproveProposal: (...a: unknown[]) => rollbackImproveProposal(...a),
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

  it("disables experiment and canary when baseline run is missing", async () => {
    listImproveProposals.mockResolvedValue({
      items: [
        {
          id: "imp_nobase",
          title: "Low score without run",
          baselineRunId: "",
          status: "draft",
          canaryPercent: 0,
          source: "low_score",
          scoreEventId: "sev_nobase",
        },
      ],
    });
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={qc}>
        <ImproveProposalsPane />
      </QueryClientProvider>,
    );
    await waitFor(() => expect(screen.getByTestId("improve-proposal-imp_nobase")).toBeInTheDocument());
    expect(screen.getByTestId("improve-experiment-imp_nobase")).toBeDisabled();
    expect(screen.getByTestId("improve-canary-imp_nobase")).toBeDisabled();
    expect(screen.getByTestId("improve-baseline-imp_nobase")).toHaveTextContent("—");
  });

  it("disables create with title explaining missing baseline", async () => {
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={qc}>
        <ImproveProposalsPane />
      </QueryClientProvider>,
    );
    await waitFor(() => expect(screen.getByTitle("需要填写基线 Run ID")).toBeDisabled());
  });

  it("disables create when title is cleared", async () => {
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={qc}>
        <ImproveProposalsPane />
      </QueryClientProvider>,
    );
    const create = await screen.findByTestId("improve-create");
    fireEvent.change(screen.getByTestId("improve-title"), { target: { value: "" } });
    fireEvent.change(screen.getByTestId("improve-baseline"), { target: { value: "run_x" } });
    expect(create).toBeDisabled();
    expect(create).toHaveAttribute("title", "需要填写标题");
  });

  it("requires confirm before creating a proposal", async () => {
    createImproveProposal.mockResolvedValue({ id: "imp_new" });
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={qc}>
        <ImproveProposalsPane />
      </QueryClientProvider>,
    );
    fireEvent.change(await screen.findByTestId("improve-baseline"), { target: { value: "run_x" } });
    const create = screen.getByTestId("improve-create");
    expect(create).toHaveAttribute("title", "创建提案（需确认）");
    fireEvent.click(create);
    expect(confirmSpy).toHaveBeenCalled();
    expect(createImproveProposal).not.toHaveBeenCalled();
    confirmSpy.mockRestore();
  });

  it("disables canary when percent is out of 1–100", async () => {
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={qc}>
        <ImproveProposalsPane />
      </QueryClientProvider>,
    );
    await waitFor(() => expect(screen.getByTestId("improve-canary-imp_low")).toBeInTheDocument());
    const canary = screen.getByTestId("improve-canary-imp_low");
    fireEvent.change(screen.getByTestId("improve-canary-percent"), { target: { value: "150" } });
    expect(canary).toBeDisabled();
    expect(canary).toHaveAttribute("title", "灰度 % 须为 1–100 的整数");
    fireEvent.change(screen.getByTestId("improve-canary-percent"), { target: { value: "-1" } });
    expect(canary).toBeDisabled();
    fireEvent.change(screen.getByTestId("improve-canary-percent"), { target: { value: "10" } });
    expect(canary).not.toBeDisabled();
    expect(canary).toHaveAttribute("title", "启动灰度（需确认）");
  });

  it("disables promote on draft and allows rollback", async () => {
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={qc}>
        <ImproveProposalsPane />
      </QueryClientProvider>,
    );
    await waitFor(() => expect(screen.getByTestId("improve-promote-imp_low")).toBeDisabled());
    expect(screen.getByTestId("improve-promote-imp_low")).toHaveAttribute(
      "title",
      "需先实验或灰度后再晋升",
    );
    expect(screen.getByTestId("improve-promote-imp_low")).toHaveAttribute(
      "aria-label",
      "需先实验或灰度后再晋升",
    );
    expect(screen.getByTestId("improve-rollback-imp_low")).not.toBeDisabled();
    expect(screen.getByTestId("improve-rollback-imp_low")).toHaveAttribute("aria-label", "回滚");
    expect(screen.getByTestId("improve-rollback-imp_low")).toHaveAttribute("title", "回滚（需确认）");
    expect(screen.getByTestId("improve-create")).toHaveAttribute("aria-label", "需要填写基线 Run ID");
  });

  it("requires confirm before experiment and rollback", async () => {
    startImproveExperiment.mockResolvedValue({ ok: true });
    rollbackImproveProposal.mockResolvedValue({ ok: true });
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={qc}>
        <ImproveProposalsPane />
      </QueryClientProvider>,
    );
    await waitFor(() => expect(screen.getByTestId("improve-experiment-imp_low")).toBeInTheDocument());
    expect(screen.getByTestId("improve-experiment-imp_low")).toHaveAttribute("title", "启动实验（需确认）");
    fireEvent.click(screen.getByTestId("improve-experiment-imp_low"));
    expect(confirmSpy).toHaveBeenCalled();
    expect(startImproveExperiment).not.toHaveBeenCalled();

    fireEvent.click(screen.getByTestId("improve-rollback-imp_low"));
    expect(rollbackImproveProposal).not.toHaveBeenCalled();

    confirmSpy.mockReturnValue(true);
    fireEvent.click(screen.getByTestId("improve-rollback-imp_low"));
    await waitFor(() => {
      expect(rollbackImproveProposal).toHaveBeenCalledWith("imp_low", expect.anything());
    });
    confirmSpy.mockRestore();
  });

  it("enables promote when status is canary", async () => {
    listImproveProposals.mockResolvedValue({
      items: [
        {
          id: "imp_canary",
          title: "Canary ready",
          baselineRunId: "run_base",
          status: "canary",
          canaryPercent: 10,
        },
      ],
    });
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={qc}>
        <ImproveProposalsPane />
      </QueryClientProvider>,
    );
    await waitFor(() => expect(screen.getByTestId("improve-promote-imp_canary")).not.toBeDisabled());
  });
});
