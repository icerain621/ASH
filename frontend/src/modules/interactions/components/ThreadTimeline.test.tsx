import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ThreadTimeline } from "./ThreadTimeline";

const getInteractionByRun = vi.fn();
const getInteractionThread = vi.fn();

vi.mock("../api/interactions.api", () => ({
  getInteractionByRun: (...a: unknown[]) => getInteractionByRun(...a),
  getInteractionThread: (...a: unknown[]) => getInteractionThread(...a),
}));

describe("ThreadTimeline", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    getInteractionByRun.mockResolvedValue({
      runId: "run_1",
      thread: { id: "th_1", spaceId: "local", runId: "run_1", kind: "main", status: "sealed", digest: "thd_abc" },
    });
    getInteractionThread.mockResolvedValue({
      threadId: "th_1",
      runId: "run_1",
      spaceId: "local",
      digest: "thd_abc",
      headSeq: 2,
      links: [{ id: "ml_1", spaceId: "local", sessionId: "", threadId: "th_1", runId: "run_1", eventSeq: 2, memoryId: "m1", linkType: "hit_used" }],
      nodes: [
        { id: "n1", seq: 1, ts: 1, type: "session.turn", visibility: "model_visible" },
        { id: "n2", seq: 2, ts: 2, type: "memory.hit_used", visibility: "ui_only" },
      ],
    });
  });

  it("renders sealed badge and nodes; click selects seq", async () => {
    const onSelectSeq = vi.fn();
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={qc}>
        <ThreadTimeline runId="run_1" highlightSeq={null} onSelectSeq={onSelectSeq} />
      </QueryClientProvider>,
    );
    await waitFor(() => expect(screen.getByTestId("thread-timeline-list")).toBeInTheDocument());
    expect(screen.getByTestId("thread-timeline-badge")).toHaveAttribute("data-sealed", "1");
    const nodes = screen.getAllByTestId("thread-timeline-node");
    expect(nodes).toHaveLength(2);
    expect(nodes[1]).toHaveAttribute("data-linked", "1");
    fireEvent.click(nodes[1]);
    expect(onSelectSeq).toHaveBeenCalledWith(2);
  });
});
