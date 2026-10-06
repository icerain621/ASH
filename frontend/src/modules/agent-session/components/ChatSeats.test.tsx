import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { FULL_ACCESS_CONFIRM_MESSAGE } from "../permissionModeLabels";
import { ChatSeats } from "./ChatSeats";

const listAgentModels = vi.hoisted(() => vi.fn());
const updateSession = vi.hoisted(() => vi.fn());
const listModelProviders = vi.hoisted(() => vi.fn());

vi.mock("../api/session.api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/session.api")>();
  return {
    ...actual,
    listAgentModels: (...args: unknown[]) => listAgentModels(...args),
    updateSession: (...args: unknown[]) => updateSession(...args),
  };
});

vi.mock("@/modules/platform/api/platform.api", () => ({
  listModelProviders: (...args: unknown[]) => listModelProviders(...args),
}));

function wrap(ui: ReactNode) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(<QueryClientProvider client={qc}>{ui}</QueryClientProvider>);
}

describe("ChatSeats", () => {
  beforeEach(() => {
    listAgentModels.mockReset();
    updateSession.mockReset();
    listModelProviders.mockReset();
    listAgentModels.mockResolvedValue({
      items: [
        { id: "static", label: "static", providerKind: "static" },
        { id: "execgo", label: "execgo", providerKind: "execgo" },
        { id: "acp_sdk", label: "acp_sdk", providerKind: "acp_sdk" },
      ],
    });
    listModelProviders.mockResolvedValue({
      items: [
        { id: "primary", provider: "openai", status: "not_configured", role: "primary" },
        { id: "fallback", provider: "openai", status: "available", role: "fallback" },
      ],
    });
    updateSession.mockResolvedValue({
      id: "sess_1",
      spaceId: "local",
      status: "active",
      providerKind: "execgo",
      permissionMode: "full",
    });
  });

  it("shows model-router provider health", async () => {
    wrap(
      <ChatSeats
        session={{
          id: "sess_1",
          spaceId: "local",
          status: "active",
          providerKind: "static",
          permissionMode: "read-only",
        }}
      />,
    );
    await waitFor(() => expect(listModelProviders).toHaveBeenCalled());
    expect(await screen.findByTestId("agent-chat-seat-model-health")).toHaveTextContent(
      /primary:未配置.*fallback:可用/,
    );
  });

  it("renders seats and PATCHes providerKind / permissionMode", async () => {
    wrap(
      <ChatSeats
        session={{
          id: "sess_1",
          spaceId: "local",
          status: "active",
          providerKind: "static",
          permissionMode: "read-only",
        }}
      />,
    );
    expect(screen.getByTestId("agent-chat-seats")).toBeInTheDocument();
    await waitFor(() => expect(listAgentModels).toHaveBeenCalled());
    await waitFor(() =>
      expect(screen.getByTestId("agent-chat-seat-model")).toHaveValue("static"),
    );
    expect(screen.getByTestId("agent-chat-seat-model").getAttribute("title")).toMatch(
      /^选择本会话模型 \/ provider/,
    );
    expect(screen.getByTestId("agent-chat-seat-permission")).toHaveDisplayValue(/询问/);
    expect(screen.getByTestId("agent-chat-seat-permission")).toHaveAttribute(
      "title",
      "本会话审批模式（完全访问需确认）",
    );

    const model = screen.getByTestId("agent-chat-seat-model") as HTMLSelectElement;
    fireEvent.change(model, { target: { value: "execgo" } });
    await waitFor(() =>
      expect(updateSession).toHaveBeenCalledWith("sess_1", { providerKind: "execgo" }),
    );

    const perm = screen.getByTestId("agent-chat-seat-permission") as HTMLSelectElement;
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(true);
    fireEvent.change(perm, { target: { value: "full" } });
    await waitFor(() =>
      expect(updateSession).toHaveBeenCalledWith("sess_1", { permissionMode: "full" }),
    );
    expect(confirmSpy).toHaveBeenCalledWith(FULL_ACCESS_CONFIRM_MESSAGE);
    confirmSpy.mockRestore();
  });

  it("PATCHes reasoningEffort from the model menu", async () => {
    wrap(
      <ChatSeats
        session={{
          id: "sess_1",
          spaceId: "local",
          status: "active",
          providerKind: "static",
          permissionMode: "read-only",
          reasoningEffort: "high",
        }}
      />,
    );
    await waitFor(() => expect(listAgentModels).toHaveBeenCalled());
    fireEvent.click(await screen.findByTestId("agent-chat-seat-model-trigger"));
    fireEvent.click(screen.getByTestId("agent-chat-seat-effort-open"));
    fireEvent.click(screen.getByTestId("agent-chat-seat-effort-max"));
    await waitFor(() =>
      expect(updateSession).toHaveBeenCalledWith("sess_1", { reasoningEffort: "max" }),
    );
    expect(screen.getByTestId("agent-chat-seat-model-trigger")).toHaveTextContent(/Max/);
  });

  it("defaults permission seat to read-only when session omits permissionMode", async () => {
    wrap(
      <ChatSeats
        session={{
          id: "sess_1",
          spaceId: "local",
          status: "active",
          providerKind: "static",
        }}
      />,
    );
    await waitFor(() => expect(listAgentModels).toHaveBeenCalled());
    expect(screen.getByTestId("agent-chat-seat-permission")).toHaveValue("read-only");
    expect(screen.getByTestId("agent-chat-seat-permission")).toHaveDisplayValue(/询问/);
  });

  it("shows Chinese labels for workspace-write without confirm", async () => {
    wrap(
      <ChatSeats
        session={{
          id: "sess_1",
          spaceId: "local",
          status: "active",
          providerKind: "static",
          permissionMode: "read-only",
        }}
      />,
    );
    await waitFor(() => expect(listAgentModels).toHaveBeenCalled());
    const confirmSpy = vi.spyOn(window, "confirm");
    fireEvent.change(screen.getByTestId("agent-chat-seat-permission"), {
      target: { value: "workspace-write" },
    });
    await waitFor(() =>
      expect(updateSession).toHaveBeenCalledWith("sess_1", { permissionMode: "workspace-write" }),
    );
    expect(confirmSpy).not.toHaveBeenCalled();
    confirmSpy.mockRestore();
  });

  it("cancels full permission when confirm rejected", async () => {
    wrap(
      <ChatSeats
        session={{
          id: "sess_1",
          spaceId: "local",
          status: "active",
          providerKind: "static",
          permissionMode: "read-only",
        }}
      />,
    );
    await waitFor(() => expect(listAgentModels).toHaveBeenCalled());
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    fireEvent.change(screen.getByTestId("agent-chat-seat-permission"), {
      target: { value: "full" },
    });
    expect(updateSession).not.toHaveBeenCalledWith("sess_1", { permissionMode: "full" });
    confirmSpy.mockRestore();
  });

  it("tolerates null provider status", async () => {
    listModelProviders.mockResolvedValueOnce({
      items: [{ id: "p1", role: "default", status: null }],
    });
    wrap(
      <ChatSeats
        session={{
          id: "sess_1",
          spaceId: "local",
          status: "active",
          providerKind: "static",
          permissionMode: "read-only",
        }}
      />,
    );
    await waitFor(() => {
      expect(screen.getByTestId("agent-chat-seats")).toBeInTheDocument();
      expect(screen.getByText(/未配置/)).toBeInTheDocument();
    });
  });

  it("surfaces patch errors and reverts plan draft", async () => {
    updateSession.mockRejectedValueOnce(new Error("plan not found"));
    wrap(
      <ChatSeats
        session={{
          id: "sess_1",
          spaceId: "local",
          status: "active",
          providerKind: "static",
          permissionMode: "read-only",
          planId: "plan_ok",
        }}
      />,
    );
    await waitFor(() => expect(listAgentModels).toHaveBeenCalled());
    const plan = screen.getByTestId("agent-chat-seat-plan");
    fireEvent.change(plan, { target: { value: "plan_bad" } });
    fireEvent.blur(plan);
    expect(await screen.findByTestId("agent-chat-seats-error")).toHaveTextContent(
      "未找到该 Plan，请检查 planId",
    );
    await waitFor(() => expect(plan).toHaveValue("plan_ok"));
  });
});
