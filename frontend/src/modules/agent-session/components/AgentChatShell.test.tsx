import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import type { ComponentProps } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { AgentChatShell } from "./AgentChatShell";

const listAgentSessions = vi.fn();
const createAgentSession = vi.fn();
const listSessionEvents = vi.fn();
const submitSessionIntent = vi.fn();
const getAgentSession = vi.fn();
const patchAgentSession = vi.fn();
const closeAgentSession = vi.fn();
const purgeAgentSession = vi.fn();
const listAgentWorkspaces = vi.fn();
const createAgentWorkspace = vi.fn();
const listAgentCommands = vi.fn();
const listAgentModels = vi.fn();
const updateSession = vi.fn();

vi.mock("@/modules/agent-session/api/session.api", async () => {
  const actual = await vi.importActual<typeof import("../api/session.api")>("../api/session.api");
  return {
    ...actual,
    listAgentSessions: (...args: unknown[]) => listAgentSessions(...args),
    createAgentSession: (...args: unknown[]) => createAgentSession(...args),
    listSessionEvents: (...args: unknown[]) => listSessionEvents(...args),
    submitSessionIntent: (...args: unknown[]) => submitSessionIntent(...args),
    getAgentSession: (...args: unknown[]) => getAgentSession(...args),
    patchAgentSession: (...args: unknown[]) => patchAgentSession(...args),
    closeAgentSession: (...args: unknown[]) => closeAgentSession(...args),
    purgeAgentSession: (...args: unknown[]) => purgeAgentSession(...args),
    listAgentCommands: (...args: unknown[]) => listAgentCommands(...args),
    listAgentModels: (...args: unknown[]) => listAgentModels(...args),
    updateSession: (...args: unknown[]) => updateSession(...args),
  };
});

vi.mock("@/modules/agent-session/api/workspace.api", () => ({
  listAgentWorkspaces: (...args: unknown[]) => listAgentWorkspaces(...args),
  createAgentWorkspace: (...args: unknown[]) => createAgentWorkspace(...args),
  patchAgentWorkspace: vi.fn(),
  attachAgentWorkspaceSession: vi.fn(),
  closeAgentWorkspace: vi.fn(),
}));

vi.mock("@/services/sse/runStream", () => ({
  useRunStream: vi.fn(() => ({ lines: [], status: "idle" as const })),
  useSessionStream: vi.fn(() => ({ lines: [], status: "idle" as const })),
  sessionStreamPath: (id: string, url?: string | null) =>
    url || `/api/v1/agents/sessions/${id}/stream`,
}));

vi.mock("@/modules/interactions/api/interactions.api", () => ({
  getInteractionByRun: vi.fn(async () => ({
    runId: "run_1",
    thread: { id: "th_1", spaceId: "local", runId: "run_1", kind: "main", status: "open" },
  })),
  getInteractionThread: vi.fn(async () => ({
    threadId: "th_1",
    runId: "run_1",
    spaceId: "local",
    nodes: [],
    links: [],
    digest: "thd_test",
    headSeq: 0,
  })),
}));

function renderShell(props: Partial<ComponentProps<typeof AgentChatShell>> = {}) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <AgentChatShell
        selectedSessionId={props.selectedSessionId ?? null}
        onSelectSession={props.onSelectSession ?? vi.fn()}
        runStatus={props.runStatus}
        gateReason={props.gateReason}
        onIntentSuccess={props.onIntentSuccess}
      />
    </QueryClientProvider>,
  );
}

describe("AgentChatShell", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    listAgentWorkspaces.mockResolvedValue({
      items: [
        {
          id: "aw_1",
          spaceId: "local",
          title: "Feat ship",
          sessionIds: ["sess_a"],
          status: "active",
        },
      ],
    });
    createAgentWorkspace.mockResolvedValue({
      id: "aw_new",
      spaceId: "local",
      title: "新工作区",
      sessionIds: [],
      status: "active",
    });
    listAgentCommands.mockResolvedValue({
      items: [
        { name: "/help", description: "列出可用命令", source: "builtin" },
        { name: "/clear", description: "清空", source: "builtin" },
      ],
    });
    listAgentModels.mockResolvedValue({
      items: [
        { id: "static", label: "static", providerKind: "static" },
        { id: "execgo", label: "execgo", providerKind: "execgo" },
        { id: "acp_sdk", label: "acp_sdk", providerKind: "acp_sdk" },
      ],
    });
    updateSession.mockImplementation(async (id: string, body: Record<string, unknown>) => ({
      id,
      spaceId: "local",
      status: "active",
      ...body,
    }));
    listAgentSessions.mockResolvedValue({
      items: [
        {
          id: "sess_a",
          spaceId: "local",
          status: "active",
          goal: "Ship chat",
          runId: "run_1",
          workspaceId: "aw_1",
          updatedAt: 2,
        },
        {
          id: "sess_b",
          spaceId: "local",
          status: "active",
          updatedAt: 1,
        },
      ],
    });
    createAgentSession.mockResolvedValue({
      id: "sess_new",
      spaceId: "local",
      status: "active",
      updatedAt: 3,
    });
    getAgentSession.mockResolvedValue({
      id: "sess_a",
      spaceId: "local",
      status: "active",
      runId: "run_1",
      goal: "Ship chat",
      workspaceId: "aw_1",
    });
    listSessionEvents.mockResolvedValue({
      sessionId: "sess_a",
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
      ],
    });
    submitSessionIntent.mockResolvedValue({
      id: "sess_a",
      spaceId: "local",
      status: "active",
      runId: "run_1",
    });
    patchAgentSession.mockResolvedValue({
      id: "sess_a",
      spaceId: "local",
      status: "active",
      title: "Renamed",
      runId: "run_1",
    });
    closeAgentSession.mockResolvedValue({
      id: "sess_a",
      spaceId: "local",
      status: "closed",
      title: "Ship chat",
    });
    purgeAgentSession.mockResolvedValue({ id: "sess_a", purged: true });
  });

  it("renders three-column shell with session list", async () => {
    renderShell();
    expect(await screen.findByTestId("agent-chat-shell")).toBeTruthy();
    expect(screen.getByTestId("agent-session-history")).toBeTruthy();
    expect(screen.getByTestId("agent-workspace-list")).toBeTruthy();
    expect(await screen.findByText("Ship chat")).toBeTruthy();
    expect(screen.getByTestId("agent-chat-view-tabs")).toBeTruthy();
    expect(screen.getByTestId("agent-chat-resize-sidebar")).toBeTruthy();
    expect(screen.getByTestId("agent-chat-resize-details")).toBeTruthy();
    expect(screen.getByTestId("agent-chat-details")).toBeTruthy();
  });

  it("creates a blank session on New without requiring goal", async () => {
    const onSelectSession = vi.fn();
    renderShell({ onSelectSession });
    fireEvent.click(await screen.findByTestId("agent-history-new"));
    await waitFor(() => {
      expect(createAgentSession).toHaveBeenCalledWith({});
      expect(onSelectSession).toHaveBeenCalledWith(
        expect.objectContaining({ id: "sess_new" }),
      );
    });
  });

  it("creates workspace and new session under active workspace", async () => {
    const onSelectSession = vi.fn();
    createAgentSession.mockResolvedValue({
      id: "sess_ws",
      spaceId: "local",
      status: "active",
      workspaceId: "aw_new",
    });
    renderShell({ onSelectSession });
    fireEvent.click(await screen.findByTestId("agent-workspace-new"));
    await waitFor(() => {
      expect(createAgentWorkspace).toHaveBeenCalledWith({ title: "新工作区" });
    });
    fireEvent.click(screen.getByTestId("agent-history-new"));
    await waitFor(() => {
      expect(createAgentSession).toHaveBeenCalledWith({ workspaceId: "aw_new" });
      expect(onSelectSession).toHaveBeenCalledWith(
        expect.objectContaining({ id: "sess_ws", workspaceId: "aw_new" }),
      );
    });
  });

  it("switches Chat and Trajectory tabs", async () => {
    renderShell({ selectedSessionId: "sess_a" });
    expect(await screen.findByTestId("agent-chat-transcript")).toBeTruthy();
    fireEvent.click(screen.getByTestId("agent-chat-tab-trajectory"));
    expect(await screen.findByTestId("agent-chat-trajectory")).toBeTruthy();
    fireEvent.click(screen.getByTestId("agent-chat-tab-chat"));
    expect(await screen.findByTestId("agent-chat-transcript")).toBeTruthy();
  });

  it("toggles details pane", async () => {
    renderShell({ selectedSessionId: "sess_a" });
    const details = await screen.findByTestId("agent-chat-details");
    expect(details).toHaveAttribute("data-open", "1");
    fireEvent.click(screen.getByTestId("agent-chat-details-toggle"));
    expect(details).toHaveAttribute("data-open", "0");
  });

  it("shows stop when run is running and submits stop intent", async () => {
    renderShell({ selectedSessionId: "sess_a", runStatus: "running" });
    fireEvent.click(await screen.findByTestId("agent-intent-stop"));
    await waitFor(() => {
      expect(submitSessionIntent).toHaveBeenCalledWith(
        "sess_a",
        expect.objectContaining({ action: "stop" }),
      );
    });
  });

  it("renames and closes sessions from history list", async () => {
    const onSelectSession = vi.fn();
    renderShell({ selectedSessionId: "sess_a", onSelectSession });
    fireEvent.click(await screen.findByTestId("agent-history-rename-btn-sess_a"));
    fireEvent.change(screen.getByTestId("agent-history-rename-input-sess_a"), {
      target: { value: "Renamed" },
    });
    fireEvent.click(screen.getByTestId("agent-history-rename-save-sess_a"));
    await waitFor(() => {
      expect(patchAgentSession).toHaveBeenCalledWith("sess_a", { title: "Renamed" });
    });
    fireEvent.click(screen.getByTestId("agent-history-close-sess_a"));
    await waitFor(() => {
      expect(closeAgentSession).toHaveBeenCalledWith("sess_a");
      expect(onSelectSession).toHaveBeenCalledWith(null);
    });
  });

  it("purges session after confirm", async () => {
    const onSelectSession = vi.fn();
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(true);
    renderShell({ selectedSessionId: "sess_a", onSelectSession });
    fireEvent.click(await screen.findByTestId("agent-history-purge-sess_a"));
    await waitFor(() => {
      expect(confirmSpy).toHaveBeenCalled();
      expect(purgeAgentSession).toHaveBeenCalledWith("sess_a");
      expect(onSelectSession).toHaveBeenCalledWith(null);
    });
    confirmSpy.mockRestore();
  });
});
