import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { CIPage } from "./CIPage";
import { renderPage } from "@/test/renderPage";
import { ApiError } from "@/services/http/client";
import { listCIRuns } from "@/modules/closure/api/closure.api";

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
    renderPage(<CIPage />);
    await waitFor(() => {
      expect(screen.getByRole("button", { name: /同步 runs/ })).toBeEnabled();
    });
    fireEvent.click(screen.getByRole("button", { name: /同步 runs/ }));
    await waitFor(() => {
      expect(screen.getByText(/CI_PROVIDER_UNAVAILABLE: github circuit open/)).toBeInTheDocument();
    });
  });
});
