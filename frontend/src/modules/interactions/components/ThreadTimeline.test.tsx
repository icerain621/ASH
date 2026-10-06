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
        {
          id: "n3",
          seq: 3,
          ts: 3,
          type: "assistant.message",
          visibility: "model_visible",
          payload: { text: "ok", reasoningEffort: "max" },
        },
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
    expect(nodes).toHaveLength(3);
    expect(nodes[1]).toHaveAttribute("data-linked", "1");
    expect(nodes[2]).toHaveTextContent(/assistant\.message · Max/);
    fireEvent.click(nodes[1]);
    expect(onSelectSeq).toHaveBeenCalledWith(2);
  });

  it("filters to model_visible nodes", async () => {
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={qc}>
        <ThreadTimeline runId="run_1" />
      </QueryClientProvider>,
    );
    await waitFor(() => expect(screen.getAllByTestId("thread-timeline-node")).toHaveLength(3));
    fireEvent.click(screen.getByTestId("thread-timeline-filter-model_visible"));
    const visible = screen.getAllByTestId("thread-timeline-node");
    expect(visible).toHaveLength(2);
    expect(visible[0]).toHaveTextContent("session.turn");
    expect(visible[1]).toHaveTextContent(/assistant\.message · Max/);
  });

  it("explains an empty filter instead of claiming the thread has no nodes", async () => {
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={qc}>
        <ThreadTimeline runId="run_1" />
      </QueryClientProvider>,
    );
    await waitFor(() => expect(screen.getByTestId("thread-timeline-list")).toBeInTheDocument());
    fireEvent.click(screen.getByTestId("thread-timeline-filter-tool"));
    expect(screen.getByTestId("thread-timeline-empty")).toHaveTextContent("当前筛选下没有节点");
    expect(screen.queryByText("暂无折叠节点")).toBeNull();
  });

  it("surfaces run-not-found instead of empty timeline", async () => {
    getInteractionByRun.mockRejectedValueOnce(new Error("run not found"));
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={qc}>
        <ThreadTimeline runId="run_missing" />
      </QueryClientProvider>,
    );
    expect(await screen.findByTestId("thread-timeline-error")).toHaveTextContent(
      "未找到该 Run，请检查 Run ID",
    );
    expect(screen.queryByTestId("thread-timeline-empty")).toBeNull();
  });
});
