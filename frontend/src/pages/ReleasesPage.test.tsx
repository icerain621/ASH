import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { ReleasesPage } from "./ReleasesPage";
import { createRollbackDrill, listReleases } from "@/modules/closure/api/closure.api";
import { renderPage } from "@/test/renderPage";

vi.mock("@/modules/closure/api/closure.api", () => ({
  listReleases: vi.fn().mockResolvedValue({
    items: [{ id: "rel_1", version: "v0.1.0", status: "draft", gateStatus: "block" }],
  }),
  getReleaseChecklist: vi.fn().mockResolvedValue({ items: [] }),
  createRelease: vi.fn(),
  patchReleaseChecklist: vi.fn(),
  evaluateReleaseGate: vi.fn(),
  createRollbackDrill: vi.fn().mockResolvedValue({ ok: true }),
}));

describe("ReleasesPage", () => {
  it("renders releases heading and create release control", async () => {
    renderPage(<ReleasesPage />);
    expect(screen.getByRole("heading", { name: "发布与灰度回滚" })).toBeInTheDocument();
    await waitFor(() => {
      expect(screen.getByTestId("release-create")).toBeInTheDocument();
    });
    const create = screen.getByTestId("release-create");
    expect(create).toBeDisabled();
    expect(create).toHaveAttribute("title", "需要填写 Version");
    expect(screen.getByRole("button", { name: "刷新" })).toHaveAttribute("title", "刷新发布列表");
  });

  it("tolerates null releases items", async () => {
    vi.mocked(listReleases).mockResolvedValueOnce({ items: null as unknown as [] });
    renderPage(<ReleasesPage />);
    await waitFor(() => {
      expect(screen.getByRole("heading", { name: "发布与灰度回滚" })).toBeInTheDocument();
      expect(screen.getByRole("button", { name: "创建 release" })).toBeInTheDocument();
    });
  });

  it("disables create release until version and title are filled", async () => {
    renderPage(<ReleasesPage />);
    const create = await screen.findByTestId("release-create");
    expect(create).toBeDisabled();
    expect(create).toHaveAttribute("title", "需要填写 Version");

    fireEvent.change(screen.getByTestId("release-version"), {
      target: { value: "v9.9.9" },
    });
    expect(create).toBeDisabled();
    expect(create).toHaveAttribute("title", "需要填写 Title");

    fireEvent.change(screen.getByTestId("release-title"), {
      target: { value: "spot title" },
    });
    expect(create).not.toBeDisabled();
    expect(create).toHaveAttribute("title", "创建 release（需确认）");
  });

  it("requires confirm before creating a release", async () => {
    const { createRelease } = await import("@/modules/closure/api/closure.api");
    vi.mocked(createRelease).mockResolvedValue({ id: "rel_new" } as never);
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    renderPage(<ReleasesPage />);
    fireEvent.change(await screen.findByTestId("release-version"), { target: { value: "v9.9.9" } });
    fireEvent.change(screen.getByTestId("release-title"), { target: { value: "spot title" } });
    fireEvent.click(screen.getByTestId("release-create"));
    expect(confirmSpy).toHaveBeenCalled();
    expect(createRelease).not.toHaveBeenCalled();
    confirmSpy.mockRestore();
  });

  it("disables rollback record until scenario is filled", async () => {
    renderPage(<ReleasesPage />);
    const submit = await screen.findByTestId("release-rollback-submit");
    await waitFor(() => {
      expect(submit).toHaveAttribute("title", "需要填写回滚场景");
    });
    expect(submit).toBeDisabled();

    fireEvent.change(screen.getByTestId("release-rollback-scenario"), {
      target: { value: "rollback image" },
    });
    expect(submit).not.toBeDisabled();
    expect(submit).toHaveAttribute("title", "记录回滚演练（需确认）");
  });

  it("requires confirm before running gate", async () => {
    const { evaluateReleaseGate } = await import("@/modules/closure/api/closure.api");
    vi.mocked(evaluateReleaseGate).mockClear();
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    renderPage(<ReleasesPage />);
    const gate = await screen.findByTestId("release-gate-run");
    await waitFor(() => {
      expect(gate).toHaveAttribute("title", "运行 gate（需确认）");
    });
    fireEvent.click(gate);
    expect(confirmSpy).toHaveBeenCalled();
    expect(evaluateReleaseGate).not.toHaveBeenCalled();
    confirmSpy.mockRestore();
  });

  it("requires confirm before recording rollback drill", async () => {
    vi.mocked(createRollbackDrill).mockClear();
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    renderPage(<ReleasesPage />);
    fireEvent.change(await screen.findByTestId("release-rollback-scenario"), {
      target: { value: "rollback image" },
    });
    const submit = screen.getByTestId("release-rollback-submit");
    expect(submit).toHaveAttribute("title", "记录回滚演练（需确认）");
    fireEvent.click(submit);
    expect(confirmSpy).toHaveBeenCalled();
    expect(createRollbackDrill).not.toHaveBeenCalled();

    confirmSpy.mockReturnValue(true);
    fireEvent.click(submit);
    await waitFor(() => {
      expect(createRollbackDrill).toHaveBeenCalledWith(
        "rel_1",
        expect.objectContaining({ scenario: "rollback image" }),
      );
    });
    confirmSpy.mockRestore();
  });
});
