import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { AgentSessionPanel } from "./AgentSessionPanel";

const createAgentSession = vi.fn();
const listSessionEvents = vi.fn();
const submitSessionIntent = vi.fn();
const listInteractionSessionThreads = vi.fn();

vi.mock("@/modules/agent-session/api/session.api", async () => {
  const actual = await vi.importActual<typeof import("../api/session.api")>("../api/session.api");
  return {
    ...actual,
    createAgentSession: (...args: unknown[]) => createAgentSession(...args),
    listSessionEvents: (...args: unknown[]) => listSessionEvents(...args),
    submitSessionIntent: (...args: unknown[]) => submitSessionIntent(...args),
  };
});

vi.mock("@/modules/interactions/api/interactions.api", () => ({
  listInteractionSessionThreads: (...args: unknown[]) => listInteractionSessionThreads(...args),
}));

describe("AgentSessionPanel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    createAgentSession.mockResolvedValue({
      id: "sess_1",
      spaceId: "local",
      status: "active",
      runId: "run_1",
    });
    listSessionEvents.mockResolvedValue({
      sessionId: "sess_1",
      runId: "run_1",
      items: [
        {
          id: "evt_1",
          runId: "run_1",
          seq: 1,
          ts: 1,
          type: "session.turn",
          severity: "info",
          visibility: "model_visible",
          payload: { prompt: "hello" },
        },
        {
          id: "evt_2",
          runId: "run_1",
          seq: 2,
          ts: 2,
          type: "metric.kpi",
          severity: "info",
          visibility: "audit",
          payload: { n: 1 },
        },
      ],
    });
    submitSessionIntent.mockResolvedValue({ id: "sess_1", spaceId: "local", status: "active", runId: "run_1" });
    listInteractionSessionThreads.mockResolvedValue({
      sessionId: "sess_1",
      items: [{ id: "th_1", spaceId: "local", sessionId: "sess_1", runId: "run_1", kind: "main", status: "open" }],
    });
  });

  it("binds session and renders visible conversation nodes only", async () => {
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={qc}>
        <AgentSessionPanel runId="run_1" runStatus="running" />
      </QueryClientProvider>,
    );
    await waitFor(() => expect(createAgentSession).toHaveBeenCalledWith({ runId: "run_1" }));
    await waitFor(() => expect(screen.getByTestId("agent-session-thread")).toBeInTheDocument());
    const nodes = await screen.findAllByTestId("agent-conversation-node");
    expect(nodes).toHaveLength(1);
    expect(nodes[0]).toHaveAttribute("data-node-kind", "session.turn");
    await waitFor(() => expect(listInteractionSessionThreads).toHaveBeenCalledWith("sess_1"));
    expect(await screen.findByTestId("session-threads-list")).toBeInTheDocument();
  });

  it("sends prompt via intent bar", async () => {
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={qc}>
        <AgentSessionPanel runId="run_1" runStatus="running" />
      </QueryClientProvider>,
    );
    await waitFor(() => expect(screen.getByTestId("agent-intent-prompt")).toBeEnabled());
    fireEvent.change(screen.getByTestId("agent-intent-prompt"), { target: { value: "next" } });
    fireEvent.click(screen.getByTestId("agent-intent-send"));
    await waitFor(() =>
      expect(submitSessionIntent).toHaveBeenCalledWith("sess_1", {
        action: "prompt",
        prompt: "next",
        reason: undefined,
        actorId: "console",
      }),
    );
  });
});
