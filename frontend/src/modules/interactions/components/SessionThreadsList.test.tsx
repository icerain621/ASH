import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { SessionThreadsList } from "./SessionThreadsList";

const listInteractionSessionThreads = vi.fn();

vi.mock("../api/interactions.api", () => ({
  listInteractionSessionThreads: (...a: unknown[]) => listInteractionSessionThreads(...a),
}));

describe("SessionThreadsList", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    listInteractionSessionThreads.mockResolvedValue({
      sessionId: "sess_1",
      items: [
        { id: "th_aaa", spaceId: "local", sessionId: "sess_1", runId: "run_a", kind: "main", status: "open" },
        { id: "th_bbb", spaceId: "local", sessionId: "sess_1", runId: "run_b", kind: "main", status: "sealed" },
      ],
    });
  });

  it("lists threads for a session", async () => {
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={qc}>
        <SessionThreadsList sessionId="sess_1" />
      </QueryClientProvider>,
    );
    await waitFor(() => expect(listInteractionSessionThreads).toHaveBeenCalledWith("sess_1"));
    const items = await screen.findAllByTestId("session-thread-item");
    expect(items).toHaveLength(2);
    expect(items[0]).toHaveTextContent("main/open");
    expect(items[1]).toHaveTextContent("main/sealed");
  });
});
