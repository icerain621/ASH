import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ThreadComparePanel } from "./ThreadComparePanel";

const getInteractionByRun = vi.fn();
const compareInteractionThreads = vi.fn();

vi.mock("../api/interactions.api", () => ({
  getInteractionByRun: (...a: unknown[]) => getInteractionByRun(...a),
  compareInteractionThreads: (...a: unknown[]) => compareInteractionThreads(...a),
}));

describe("ThreadComparePanel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    getInteractionByRun.mockImplementation(async (runId: string) => ({
      runId,
      thread: { id: `th_${runId}`, spaceId: "local", runId, kind: "main", status: "open" },
    }));
    compareInteractionThreads.mockResolvedValue({
      leftThreadId: "th_run_a",
      rightThreadId: "th_run_b",
      leftDigest: "thd_left",
      rightDigest: "thd_right",
      nodesAdded: ["1|session.turn"],
      nodesRemoved: [],
      nodesChanged: [],
      linksAdded: [],
      linksRemoved: ["2|hit_used|m1"],
    });
  });

  it("compares two runs via thread ids", async () => {
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={qc}>
        <ThreadComparePanel />
      </QueryClientProvider>,
    );
    fireEvent.change(screen.getByTestId("thread-compare-left"), { target: { value: "run_a" } });
    fireEvent.change(screen.getByTestId("thread-compare-right"), { target: { value: "run_b" } });
    await waitFor(() => expect(screen.getByTestId("thread-compare-submit")).not.toBeDisabled());
    fireEvent.click(screen.getByTestId("thread-compare-submit"));
    await waitFor(() => expect(screen.getByTestId("thread-compare-result")).toBeInTheDocument());
    expect(compareInteractionThreads).toHaveBeenCalledWith("th_run_a", "th_run_b");
    expect(screen.getByTestId("thread-compare-result")).toHaveTextContent("不同");
  });
});
