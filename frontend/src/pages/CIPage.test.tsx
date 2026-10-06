import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { CIPage } from "./CIPage";
import { renderPage } from "@/test/renderPage";
import { ApiError } from "@/services/http/client";
import { diagnoseCIFailure, listCIRuns, listRepoConnections } from "@/modules/closure/api/closure.api";

vi.mock("@/modules/closure/api/closure.api", () => ({
  listRepoConnections: vi.fn().mockResolvedValue({
    items: [{ id: "conn_1", name: "fixture", provider: "github" }],
  }),
  listCIRuns: vi.fn().mockResolvedValue({ items: [] }),
  listCIJobs: vi.fn().mockResolvedValue({ items: [] }),
  listCIDiagnoses: vi.fn().mockResolvedValue({ items: [] }),
  diagnoseCIFailure: vi.fn(),
  adoptCIDiagnosis: vi.fn(),
  dismissCIDiagnosis: vi.fn(),
}));

vi.mock("@/modules/health/api/health.api", () => ({
  getReadyz: vi.fn().mockResolvedValue({
    status: "ready",
    liveGateHints: ["ASH_CI_FIXTURE=1"],
  }),
}));

describe("CIPage", () => {
  it("renders CI console and fixture hint when readyz reports fixture mode", async () => {
    renderPage(<CIPage />);
    expect(screen.getByRole("heading", { name: "CI 诊断控制台" })).toBeInTheDocument();
    await waitFor(() => {
      expect(screen.getByText(/Worker 已启用/)).toBeInTheDocument();
    });
  });

  it("shows CI_PROVIDER_UNAVAILABLE when sync runs fails", async () => {
    vi.mocked(listCIRuns).mockImplementation(async (opts?: { sync?: boolean }) => {
      if (opts?.sync) {
        throw new ApiError("CI_PROVIDER_UNAVAILABLE", "github circuit open");
      }
      return { items: [] };
    });
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(true);
    renderPage(<CIPage />);
    await waitFor(() => {
      expect(screen.getByRole("button", { name: /同步 runs/ })).toBeEnabled();
    });
    fireEvent.click(screen.getByRole("button", { name: /同步 runs/ }));
    await waitFor(() => {
      expect(screen.getByText(/CI_PROVIDER_UNAVAILABLE: github circuit open/)).toBeInTheDocument();
    });
    confirmSpy.mockRestore();
  });

  it("enables manual diagnose when a log is pasted without a repo connection", async () => {
    vi.mocked(listRepoConnections).mockResolvedValue({ items: [] });
    vi.mocked(diagnoseCIFailure).mockResolvedValue({
      id: "ci_diag_manual",
      connectionId: "",
      runId: "",
      jobId: "",
      status: "pending",
      rootCause: "test failed",
      decisionStatus: "pending",
      confidence: 0.5,
      fixSuggestions: [],
      evidenceRefs: [],
      adopted: false,
    });
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    renderPage(<CIPage />);
    const button = await screen.findByRole("button", { name: "诊断失败" });
    expect(button).toBeDisabled();
    fireEvent.change(screen.getByPlaceholderText(/粘贴 CI 失败日志/), {
      target: { value: "--- FAIL: TestAPI" },
    });
    expect(button).toBeEnabled();
    expect(button).toHaveAttribute("title", "诊断失败（需确认）");
    fireEvent.click(button);
    expect(confirmSpy).toHaveBeenCalled();
    expect(diagnoseCIFailure).not.toHaveBeenCalled();
    confirmSpy.mockReturnValue(true);
    fireEvent.click(button);
    await waitFor(() => {
      expect(diagnoseCIFailure).toHaveBeenCalledWith(
        expect.objectContaining({ logText: "--- FAIL: TestAPI" }),
      );
    });
    confirmSpy.mockRestore();
  });

  it("tolerates null list payloads without crashing", async () => {
    vi.mocked(listRepoConnections).mockResolvedValue({ items: null as unknown as [] });
    const { listCIDiagnoses } = await import("@/modules/closure/api/closure.api");
    vi.mocked(listCIDiagnoses).mockResolvedValue({ items: null as unknown as [] });
    renderPage(<CIPage />);
    expect(await screen.findByRole("heading", { name: "CI 诊断控制台" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "诊断失败" })).toBeDisabled();
  });

  it("explains why sync buttons stay disabled without connection or run", async () => {
    vi.mocked(listRepoConnections).mockResolvedValue({ items: [] });
    renderPage(<CIPage />);
    const syncRuns = await screen.findByTestId("ci-sync-runs");
    const syncJobs = screen.getByTestId("ci-sync-jobs");
    expect(syncRuns).toBeDisabled();
    expect(syncRuns).toHaveAttribute("title", "需要先选择 Repo 连接");
    expect(syncJobs).toBeDisabled();
    expect(syncJobs).toHaveAttribute("title", "需要先选择 Workflow Run");
  });

  it("requires confirm before syncing runs", async () => {
    vi.mocked(listRepoConnections).mockResolvedValue({
      items: [{ id: "conn_1", name: "fixture", provider: "github" }],
    } as never);
    vi.mocked(listCIRuns).mockClear();
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    renderPage(<CIPage />);
    const syncRuns = await screen.findByTestId("ci-sync-runs");
    await waitFor(() => {
      expect(syncRuns).not.toBeDisabled();
    });
    expect(syncRuns).toHaveAttribute("title", "同步 runs（需确认）");
    fireEvent.click(syncRuns);
    expect(confirmSpy).toHaveBeenCalled();
    expect(listCIRuns).not.toHaveBeenCalledWith(expect.objectContaining({ sync: true }));
    confirmSpy.mockRestore();
  });

  it("explains why diagnose stays disabled without log or job", async () => {
    vi.mocked(listRepoConnections).mockResolvedValue({ items: [] });
    renderPage(<CIPage />);
    const diagnose = await screen.findByRole("button", { name: "诊断失败" });
    expect(diagnose).toBeDisabled();
    expect(diagnose).toHaveAttribute("title", "需要粘贴失败日志，或选择 Repo 连接 / job");
    expect(screen.getByRole("button", { name: "诊断选中 job" })).toHaveAttribute(
      "title",
      "需要先选择 Workflow Job",
    );
  });

  it("requires confirm before adopting or dismissing a diagnosis", async () => {
    const { listCIDiagnoses, adoptCIDiagnosis, dismissCIDiagnosis } = await import(
      "@/modules/closure/api/closure.api"
    );
    vi.mocked(listCIDiagnoses).mockResolvedValue({
      items: [
        {
          id: "ci_diag_1",
          rootCause: "flake timeout",
          decisionStatus: "pending",
          confidence: 0.8,
          fixSuggestions: [],
          evidenceRefs: [],
          adopted: false,
        },
      ],
    } as never);
    vi.mocked(adoptCIDiagnosis).mockClear();
    vi.mocked(dismissCIDiagnosis).mockClear();
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    renderPage(<CIPage />);
    const adopt = await screen.findByTestId("ci-diagnosis-adopt-ci_diag_1");
    const dismiss = screen.getByTestId("ci-diagnosis-dismiss-ci_diag_1");
    expect(adopt).toHaveAttribute("title", "采纳诊断（需确认）");
    expect(dismiss).toHaveAttribute("title", "驳回诊断（需确认）");
    fireEvent.click(adopt);
    expect(confirmSpy).toHaveBeenCalled();
    expect(adoptCIDiagnosis).not.toHaveBeenCalled();
    fireEvent.click(dismiss);
    expect(dismissCIDiagnosis).not.toHaveBeenCalled();
    confirmSpy.mockRestore();
  });
});
