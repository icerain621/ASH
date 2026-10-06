import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ThreadComparePanel } from "./ThreadComparePanel";

const getInteractionByRun = vi.fn();
const compareInteractionThreads = vi.fn();
const listInteractionSessionThreads = vi.fn();
const forkInteractionThread = vi.fn();

vi.mock("../api/interactions.api", () => ({
  getInteractionByRun: (...a: unknown[]) => getInteractionByRun(...a),
  compareInteractionThreads: (...a: unknown[]) => compareInteractionThreads(...a),
  listInteractionSessionThreads: (...a: unknown[]) => listInteractionSessionThreads(...a),
  forkInteractionThread: (...a: unknown[]) => forkInteractionThread(...a),
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

  it("explains why compare stays disabled without both sides", () => {
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={qc}>
        <ThreadComparePanel />
      </QueryClientProvider>,
    );
    const submit = screen.getByTestId("thread-compare-submit");
    expect(submit).toBeDisabled();
    expect(submit).toHaveAttribute("title", "需要填写左侧 Run");
  });

  it("surfaces resolve error quickly even when QueryClient would retry", async () => {
    getInteractionByRun.mockRejectedValue(new Error("run not found"));
    const qc = new QueryClient({
      defaultOptions: { queries: { retry: 3, retryDelay: 50_000 } },
    });
    render(
      <QueryClientProvider client={qc}>
        <ThreadComparePanel />
      </QueryClientProvider>,
    );
    fireEvent.change(screen.getByTestId("thread-compare-left"), { target: { value: "run_missing" } });
    fireEvent.change(screen.getByTestId("thread-compare-right"), { target: { value: "run_missing_b" } });
    expect(await screen.findByTestId("thread-compare-resolve-error")).toHaveTextContent(
      "左侧 Run：未找到该 Run，请检查 Run ID",
    );
    const submit = screen.getByTestId("thread-compare-submit");
    expect(submit).toBeDisabled();
    expect(submit).toHaveAttribute("title", "左侧 Run：未找到该 Run，请检查 Run ID");
    expect(submit).not.toHaveAttribute("title", "正在解析左侧 Run…");
  });

  it("labels right-side resolve failures in title and error text", async () => {
    getInteractionByRun.mockImplementation(async (runId: string) => {
      if (runId === "run_ok") {
        return {
          thread: { id: "th_ok", sessionId: "sess_1", runId: "run_ok", kind: "main", status: "open" },
        };
      }
      throw new Error("run not found");
    });
    listInteractionSessionThreads.mockResolvedValue({ items: [] });
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={qc}>
        <ThreadComparePanel />
      </QueryClientProvider>,
    );
    fireEvent.change(screen.getByTestId("thread-compare-left"), { target: { value: "run_ok" } });
    fireEvent.change(screen.getByTestId("thread-compare-right"), { target: { value: "run_missing_b" } });
    expect(await screen.findByTestId("thread-compare-resolve-error")).toHaveTextContent(
      "右侧 Run：未找到该 Run，请检查 Run ID",
    );
    expect(screen.getByTestId("thread-compare-submit")).toHaveAttribute(
      "title",
      "右侧 Run：未找到该 Run，请检查 Run ID",
    );
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

  it("renders zero counts when compare encodes empty dimensions as null", async () => {
    compareInteractionThreads.mockResolvedValue({
      leftThreadId: "th_run_a",
      rightThreadId: "th_run_b",
      leftDigest: "thd_same",
      rightDigest: "thd_same",
      nodesAdded: null,
      nodesRemoved: null,
      nodesChanged: null,
      linksAdded: null,
      linksRemoved: null,
    });
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
    expect(screen.getByTestId("thread-compare-result")).toHaveTextContent("相同");
    expect(screen.getByText("nodes +").closest("tr")).toHaveTextContent("0");
    expect(screen.getByText("links −").closest("tr")).toHaveTextContent("0");
  });

  it("syncs left run id when defaultLeftRunId prop changes", async () => {
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const { rerender } = render(
      <QueryClientProvider client={qc}>
        <ThreadComparePanel defaultLeftRunId="run_a" />
      </QueryClientProvider>,
    );
    expect(screen.getByTestId("thread-compare-left")).toHaveValue("run_a");
    rerender(
      <QueryClientProvider client={qc}>
        <ThreadComparePanel defaultLeftRunId="run_b" />
      </QueryClientProvider>,
    );
    await waitFor(() => expect(screen.getByTestId("thread-compare-left")).toHaveValue("run_b"));
  });

  it("compares a forked sibling in the same session", async () => {
    getInteractionByRun.mockImplementation(async (runId: string) => ({
      runId,
      thread: {
        id: "th_main",
        spaceId: "local",
        sessionId: "sess_1",
        runId,
        kind: "main",
        status: "open",
      },
    }));
    listInteractionSessionThreads.mockResolvedValue({
      sessionId: "sess_1",
      items: [
        { id: "th_main", spaceId: "local", sessionId: "sess_1", runId: "run_a", kind: "main", status: "open" },
        {
          id: "th_fork",
          spaceId: "local",
          sessionId: "sess_1",
          runId: "run_a",
          kind: "fork",
          parentThreadId: "th_main",
          status: "open",
        },
      ],
    });
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={qc}>
        <ThreadComparePanel />
      </QueryClientProvider>,
    );
    fireEvent.change(screen.getByTestId("thread-compare-left"), { target: { value: "run_a" } });
    await waitFor(() => expect(screen.getByRole("option", { name: /th_fork/ })).toBeInTheDocument());
    fireEvent.change(screen.getByTestId("thread-compare-sibling"), { target: { value: "th_fork" } });
    await waitFor(() => expect(screen.getByTestId("thread-compare-submit")).not.toBeDisabled());
    fireEvent.click(screen.getByTestId("thread-compare-submit"));
    await waitFor(() => expect(compareInteractionThreads).toHaveBeenCalledWith("th_main", "th_fork"));
  });

  it("requires confirm before forking a thread", async () => {
    getInteractionByRun.mockResolvedValue({
      runId: "run_a",
      thread: {
        id: "th_main",
        spaceId: "local",
        sessionId: "sess_1",
        runId: "run_a",
        kind: "main",
        status: "open",
      },
    });
    listInteractionSessionThreads.mockResolvedValue({ sessionId: "sess_1", items: [] });
    forkInteractionThread.mockResolvedValue({
      id: "th_new_fork",
      spaceId: "local",
      sessionId: "sess_1",
      runId: "run_a",
      kind: "fork",
      parentThreadId: "th_main",
      status: "open",
    });
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={qc}>
        <ThreadComparePanel />
      </QueryClientProvider>,
    );
    fireEvent.change(screen.getByTestId("thread-compare-left"), { target: { value: "run_a" } });
    const fork = await screen.findByTestId("thread-compare-fork");
    expect(fork).toHaveAttribute("title", "Fork 出 sibling Thread（需确认）");
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    fireEvent.click(fork);
    expect(forkInteractionThread).not.toHaveBeenCalled();
    confirmSpy.mockReturnValue(true);
    fireEvent.click(fork);
    await waitFor(() => expect(forkInteractionThread).toHaveBeenCalledWith("th_main"));
    confirmSpy.mockRestore();
  });
});
