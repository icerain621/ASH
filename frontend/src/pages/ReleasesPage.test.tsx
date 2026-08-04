import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { ReleasesPage } from "./ReleasesPage";
import { renderPage } from "@/test/renderPage";

vi.mock("@/modules/closure/api/closure.api", () => ({
  listReleases: vi.fn().mockResolvedValue({ items: [] }),
  getReleaseChecklist: vi.fn().mockResolvedValue({ items: [] }),
  createRelease: vi.fn(),
  patchReleaseChecklist: vi.fn(),
  evaluateReleaseGate: vi.fn(),
  createRollbackDrill: vi.fn(),
}));

describe("ReleasesPage", () => {
  it("renders releases heading and create release control", async () => {
    renderPage(<ReleasesPage />);
    expect(screen.getByRole("heading", { name: "发布与灰度回滚" })).toBeInTheDocument();
    await waitFor(() => {
      expect(screen.getByRole("button", { name: "创建 release" })).toBeInTheDocument();
    });
  });
});
