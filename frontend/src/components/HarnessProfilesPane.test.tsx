import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { HarnessProfilesPane } from "./HarnessProfilesPane";
import { listHarnessProfiles, loadActiveHarnessProfile } from "@/modules/reviews/api/reviews.api";

vi.mock("@/modules/reviews/api/reviews.api", () => ({
  listHarnessProfiles: vi.fn(async () => ({
    items: [],
    sandboxBackends: ["local", "landlock", "docker", "remote-mock", "remote-e2b"],
  })),
  loadActiveHarnessProfile: vi.fn(async () => ({})),
  createHarnessProfile: vi.fn(),
  submitHarnessReview: vi.fn(),
  promoteHarnessProfile: vi.fn(),
  rollbackHarnessProfile: vi.fn(),
}));

describe("HarnessProfilesPane", () => {
  it("shows the sandbox backend catalog from the list envelope", async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={client}>
        <HarnessProfilesPane />
      </QueryClientProvider>,
    );
    await waitFor(() => {
      expect(screen.getByTestId("harness-sandbox-backends")).toHaveTextContent(
        "沙箱目录 · local · landlock · docker · remote-mock · remote-e2b",
      );
    });
    expect(listHarnessProfiles).toHaveBeenCalled();
    expect(loadActiveHarnessProfile).toHaveBeenCalled();
  });

  it("exposes create-draft title and disables when name is empty", async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={client}>
        <HarnessProfilesPane />
      </QueryClientProvider>,
    );
    const create = await screen.findByTestId("harness-create-draft");
    expect(create).toHaveAttribute("title", "创建 Harness Profile 草稿（需确认）");
    expect(create).not.toBeDisabled();
    fireEvent.change(screen.getByTestId("harness-name"), { target: { value: "" } });
    expect(create).toBeDisabled();
    expect(create).toHaveAttribute("title", "需要填写 Profile 名称");
  });

  it("requires confirm before creating harness draft", async () => {
    const { createHarnessProfile } = await import("@/modules/reviews/api/reviews.api");
    vi.mocked(createHarnessProfile).mockClear();
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={client}>
        <HarnessProfilesPane />
      </QueryClientProvider>,
    );
    const create = await screen.findByTestId("harness-create-draft");
    fireEvent.click(create);
    expect(confirmSpy).toHaveBeenCalled();
    expect(createHarnessProfile).not.toHaveBeenCalled();
    confirmSpy.mockRestore();
  });

  it("requires confirm before submitting harness draft for review", async () => {
    vi.mocked(listHarnessProfiles).mockResolvedValueOnce({
      items: [
        {
          id: "hp_draft",
          name: "drafty",
          version: 1,
          status: "draft",
          spec: {},
        },
      ],
      sandboxBackends: ["local"],
    } as never);
    const { submitHarnessReview } = await import("@/modules/reviews/api/reviews.api");
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={client}>
        <HarnessProfilesPane />
      </QueryClientProvider>,
    );
    const submit = await screen.findByTestId("harness-submit-hp_draft");
    expect(submit).toHaveAttribute("title", "提交编排评审（draft → in_review，需确认）");
    fireEvent.click(submit);
    expect(confirmSpy).toHaveBeenCalled();
    expect(submitHarnessReview).not.toHaveBeenCalled();
    confirmSpy.mockRestore();
  });

  it("exposes rollback title when an active profile is loaded", async () => {
    vi.mocked(loadActiveHarnessProfile).mockResolvedValueOnce({
      profile: {
        id: "hp_1",
        name: "default",
        version: 3,
        status: "active",
        spec: { sandbox: { defaultMode: "workspace-write" }, provider: { kind: "static" } },
      },
    } as never);
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={client}>
        <HarnessProfilesPane />
      </QueryClientProvider>,
    );
    const rollback = await screen.findByTestId("harness-rollback");
    expect(rollback).toHaveAttribute("title", "回滚到上一 active 版本（当前 v3，需确认）");
  });

  it("requires confirm before harness rollback", async () => {
    vi.mocked(loadActiveHarnessProfile).mockResolvedValueOnce({
      profile: {
        id: "hp_1",
        name: "default",
        version: 3,
        status: "active",
        spec: { sandbox: { defaultMode: "workspace-write" }, provider: { kind: "static" } },
      },
    } as never);
    const { rollbackHarnessProfile } = await import("@/modules/reviews/api/reviews.api");
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={client}>
        <HarnessProfilesPane />
      </QueryClientProvider>,
    );
    const rollback = await screen.findByTestId("harness-rollback");
    fireEvent.click(rollback);
    expect(confirmSpy).toHaveBeenCalled();
    expect(rollbackHarnessProfile).not.toHaveBeenCalled();
    confirmSpy.mockReturnValue(true);
    fireEvent.click(rollback);
    await waitFor(() => {
      expect(rollbackHarnessProfile).toHaveBeenCalledWith("hp_1", expect.anything());
    });
    confirmSpy.mockRestore();
  });
});
