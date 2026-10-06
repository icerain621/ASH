import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { MemoryLinkPanel } from "./MemoryLinkPanel";

const getInteractionByRun = vi.fn();
const listInteractionMemoryLinks = vi.fn();
const getInteractionThread = vi.fn();
const sealInteractionThread = vi.fn();
const replayInteractionThread = vi.fn();

vi.mock("../api/interactions.api", () => ({
  getInteractionByRun: (...a: unknown[]) => getInteractionByRun(...a),
  listInteractionMemoryLinks: (...a: unknown[]) => listInteractionMemoryLinks(...a),
  getInteractionThread: (...a: unknown[]) => getInteractionThread(...a),
  sealInteractionThread: (...a: unknown[]) => sealInteractionThread(...a),
  replayInteractionThread: (...a: unknown[]) => replayInteractionThread(...a),
}));

describe("MemoryLinkPanel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    getInteractionByRun.mockResolvedValue({
      runId: "run_1",
      thread: { id: "th_1", spaceId: "local", runId: "run_1", kind: "main", status: "open" },
    });
    listInteractionMemoryLinks.mockResolvedValue({
      threadId: "th_1",
      items: [{ id: "ml_1", spaceId: "local", sessionId: "", threadId: "th_1", runId: "run_1", eventSeq: 2, memoryId: "mem_1", linkType: "hit_used" }],
    });
    getInteractionThread.mockResolvedValue({
      threadId: "th_1",
      runId: "run_1",
      spaceId: "local",
      nodes: [],
      links: [],
      digest: "thd_abc123456789",
      headSeq: 2,
    });
  });

  it("lists memory links for the run thread", async () => {
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={qc}>
        <MemoryLinkPanel runId="run_1" />
      </QueryClientProvider>,
    );
    await waitFor(() => expect(screen.getByTestId("memory-link-list")).toBeInTheDocument());
    expect(screen.getByTestId("memory-link-item")).toHaveTextContent("hit_used");
    expect(screen.getByTestId("memory-link-item")).toHaveTextContent("mem_1");
  });

  it("surfaces run-not-found instead of empty links", async () => {
    getInteractionByRun.mockRejectedValueOnce(new Error("run not found"));
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={qc}>
        <MemoryLinkPanel runId="run_missing" />
      </QueryClientProvider>,
    );
    expect(await screen.findByTestId("memory-link-error")).toHaveTextContent("未找到该 Run，请检查 Run ID");
    expect(screen.queryByTestId("memory-link-empty")).toBeNull();
    expect(screen.getByTestId("memory-link-seal")).toHaveAttribute("title", "需要先解析到 Thread");
  });

  it("requires confirm before seal and replay", async () => {
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={qc}>
        <MemoryLinkPanel runId="run_1" />
      </QueryClientProvider>,
    );
    await waitFor(() => expect(screen.getByTestId("memory-link-list")).toBeInTheDocument());
    const seal = screen.getByTestId("memory-link-seal");
    const replay = screen.getByTestId("memory-link-replay");
    expect(seal).toHaveAttribute("title", "封印 Thread（需确认）");
    expect(replay).toHaveAttribute("title", "复现校验 digest（需确认）");

    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    fireEvent.click(seal);
    fireEvent.click(replay);
    expect(sealInteractionThread).not.toHaveBeenCalled();
    expect(replayInteractionThread).not.toHaveBeenCalled();

    confirmSpy.mockReturnValue(true);
    fireEvent.click(seal);
    await waitFor(() => expect(sealInteractionThread).toHaveBeenCalledWith("th_1"));
    fireEvent.click(replay);
    await waitFor(() => expect(replayInteractionThread).toHaveBeenCalledWith("th_1"));
    confirmSpy.mockRestore();
  });
});
