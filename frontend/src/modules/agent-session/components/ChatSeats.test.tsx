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
    const permTrigger = screen.getByTestId("agent-chat-seat-permission");
    expect(permTrigger).toHaveTextContent(/询问/);
    expect(permTrigger).toHaveAttribute("data-permission", "read-only");
    expect(permTrigger).toHaveAttribute(
      "title",
      "询问审批 · 本会话审批（完全访问需确认）",
    );

    const model = screen.getByTestId("agent-chat-seat-model") as HTMLSelectElement;
    fireEvent.change(model, { target: { value: "execgo" } });
    await waitFor(() =>
      expect(updateSession).toHaveBeenCalledWith("sess_1", { providerKind: "execgo" }),
    );

    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(true);
    fireEvent.click(permTrigger);
    fireEvent.click(screen.getByTestId("agent-chat-seat-permission-full"));
    await waitFor(() =>
      expect(updateSession).toHaveBeenCalledWith("sess_1", { permissionMode: "full" }),
    );
    expect(confirmSpy).toHaveBeenCalledWith(FULL_ACCESS_CONFIRM_MESSAGE);
    confirmSpy.mockRestore();
  });

  it("projects ASH_LLM chatModel on the model trigger", async () => {
    listAgentModels.mockResolvedValue({
      items: [
        { id: "static", label: "static", providerKind: "static" },
        { id: "execgo", label: "execgo", providerKind: "execgo" },
      ],
      chatModel: "deepseek-reasoner",
    });
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
    expect(await screen.findByTestId("agent-chat-seat-chat-model")).toHaveTextContent(
      "deepseek-reasoner",
    );
    const trigger = screen.getByTestId("agent-chat-seat-model-trigger");
    expect(trigger).toHaveTextContent(/deepseek-reasoner/);
    expect(trigger).toHaveAttribute(
      "aria-label",
      "选择模型，当前 static，Chat deepseek-reasoner，推理等级 High",
    );
    fireEvent.click(trigger);
    expect(screen.getByTestId("agent-chat-seat-model-menu")).toHaveAttribute(
      "aria-label",
      "模型与推理等级",
    );
    expect(screen.getByTestId("agent-chat-seat-model-open")).toHaveTextContent(/deepseek-reasoner/);
    fireEvent.click(screen.getByTestId("agent-chat-seat-model-open"));
    expect(screen.getByTestId("agent-chat-seat-chat-model-hint")).toHaveTextContent(
      /Chat · deepseek-reasoner/,
    );
  });

  it("focuses the first option when drilling into a model submenu", async () => {
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
    fireEvent.click(await screen.findByTestId("agent-chat-seat-model-trigger"));
    fireEvent.click(screen.getByTestId("agent-chat-seat-effort-open"));
    await waitFor(() =>
      expect(document.activeElement).toBe(screen.getByTestId("agent-chat-seat-effort-low")),
    );
    fireEvent.keyDown(document, { key: "Escape" });
    await waitFor(() =>
      expect(document.activeElement).toBe(screen.getByTestId("agent-chat-seat-effort-open")),
    );
    fireEvent.click(screen.getByTestId("agent-chat-seat-model-open"));
    await waitFor(() =>
      expect(document.activeElement).toBe(screen.getByTestId("agent-chat-seat-model-pick-static")),
    );
  });

  it("jumps effort options via typeahead characters", async () => {
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
    fireEvent.click(await screen.findByTestId("agent-chat-seat-model-trigger"));
    fireEvent.click(screen.getByTestId("agent-chat-seat-effort-open"));
    await waitFor(() =>
      expect(document.activeElement).toBe(screen.getByTestId("agent-chat-seat-effort-low")),
    );
    fireEvent.keyDown(document, { key: "h" });
    expect(document.activeElement).toBe(screen.getByTestId("agent-chat-seat-effort-high"));
    fireEvent.keyDown(document, { key: "l" });
    expect(document.activeElement).toBe(screen.getByTestId("agent-chat-seat-effort-low"));
    expect(screen.getByTestId("agent-chat-seat-model-menu")).toHaveAttribute(
      "aria-orientation",
      "vertical",
    );
  });

  it("jumps permission options via typeahead", async () => {
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
    fireEvent.click(await screen.findByTestId("agent-chat-seat-permission"));
    await waitFor(() =>
      expect(document.activeElement).toBe(
        screen.getByTestId("agent-chat-seat-permission-read-only"),
      ),
    );
    fireEvent.keyDown(document, { key: "完" });
    expect(document.activeElement).toBe(screen.getByTestId("agent-chat-seat-permission-full"));
    expect(screen.getByTestId("agent-chat-seat-permission-menu")).toHaveAttribute(
      "aria-orientation",
      "vertical",
    );
  });

  it("closes model menu with ArrowLeft on the root pane", async () => {
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
    fireEvent.click(await screen.findByTestId("agent-chat-seat-model-trigger"));
    expect(screen.getByTestId("agent-chat-seat-model-menu")).toBeInTheDocument();
    screen.getByTestId("agent-chat-seat-model-open").focus();
    fireEvent.keyDown(document, { key: "ArrowLeft" });
    await waitFor(() =>
      expect(screen.queryByTestId("agent-chat-seat-model-menu")).not.toBeInTheDocument(),
    );
    await waitFor(() =>
      expect(document.activeElement).toBe(screen.getByTestId("agent-chat-seat-model-trigger")),
    );
  });

  it("drills into model submenu with ArrowRight and returns with ArrowLeft", async () => {
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
    fireEvent.click(await screen.findByTestId("agent-chat-seat-model-trigger"));
    const modelOpen = screen.getByTestId("agent-chat-seat-model-open");
    expect(modelOpen).toHaveAttribute("aria-haspopup", "menu");
    expect(screen.getByTestId("agent-chat-seat-effort-open")).toHaveAttribute(
      "aria-haspopup",
      "menu",
    );
    modelOpen.focus();
    fireEvent.keyDown(document, { key: "ArrowRight" });
    await waitFor(() =>
      expect(document.activeElement).toBe(screen.getByTestId("agent-chat-seat-model-pick-static")),
    );
    fireEvent.keyDown(document, { key: "ArrowLeft" });
    await waitFor(() =>
      expect(document.activeElement).toBe(screen.getByTestId("agent-chat-seat-model-open")),
    );
    fireEvent.keyDown(document, { key: "ArrowDown" });
    expect(document.activeElement).toBe(screen.getByTestId("agent-chat-seat-effort-open"));
    fireEvent.keyDown(document, { key: "ArrowRight" });
    await waitFor(() =>
      expect(document.activeElement).toBe(screen.getByTestId("agent-chat-seat-effort-low")),
    );
  });

  it("uses a short model · effort title on the trigger", async () => {
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
    const trigger = await screen.findByTestId("agent-chat-seat-model-trigger");
    await waitFor(() => expect(trigger).toHaveAttribute("title", "static · High"));
  });

  it("wires aria-controls from the model trigger to the open menu", async () => {
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
    const trigger = await screen.findByTestId("agent-chat-seat-model-trigger");
    expect(trigger).not.toHaveAttribute("aria-controls");
    fireEvent.click(trigger);
    const menu = screen.getByTestId("agent-chat-seat-model-menu");
    expect(trigger).toHaveAttribute("aria-controls", menu.id);
    expect(menu.id).toBeTruthy();
  });

  it("ArrowDown/ArrowUp moves focus among model menu items", async () => {
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
    fireEvent.click(await screen.findByTestId("agent-chat-seat-model-trigger"));
    const modelOpen = screen.getByTestId("agent-chat-seat-model-open");
    const effortOpen = screen.getByTestId("agent-chat-seat-effort-open");
    HTMLElement.prototype.scrollIntoView = vi.fn();
    fireEvent.keyDown(document, { key: "ArrowDown" });
    expect(document.activeElement).toBe(modelOpen);
    expect(HTMLElement.prototype.scrollIntoView).toHaveBeenCalled();
    fireEvent.keyDown(document, { key: "ArrowDown" });
    expect(document.activeElement).toBe(effortOpen);
    fireEvent.keyDown(document, { key: "ArrowUp" });
    expect(document.activeElement).toBe(modelOpen);
    fireEvent.keyDown(document, { key: "End" });
    expect(document.activeElement).toBe(effortOpen);
    fireEvent.keyDown(document, { key: "Home" });
    expect(document.activeElement).toBe(modelOpen);
  });

  it("opens model menu from trigger with ArrowDown/ArrowUp", async () => {
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
    const trigger = await screen.findByTestId("agent-chat-seat-model-trigger");
    fireEvent.keyDown(trigger, { key: "ArrowDown" });
    expect(screen.getByTestId("agent-chat-seat-model-menu")).toBeInTheDocument();
    await waitFor(() =>
      expect(document.activeElement).toBe(screen.getByTestId("agent-chat-seat-model-open")),
    );
    fireEvent.keyDown(document, { key: "Escape" });
    await waitFor(() =>
      expect(screen.queryByTestId("agent-chat-seat-model-menu")).not.toBeInTheDocument(),
    );
    fireEvent.keyDown(trigger, { key: "ArrowUp" });
    expect(screen.getByTestId("agent-chat-seat-model-menu")).toBeInTheDocument();
    await waitFor(() =>
      expect(document.activeElement).toBe(screen.getByTestId("agent-chat-seat-effort-open")),
    );
  });

  it("opens permission menu from trigger with ArrowDown/ArrowUp", async () => {
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
    const trigger = await screen.findByTestId("agent-chat-seat-permission");
    fireEvent.keyDown(trigger, { key: "ArrowDown" });
    expect(screen.getByTestId("agent-chat-seat-permission-menu")).toBeInTheDocument();
    await waitFor(() =>
      expect(document.activeElement).toBe(
        screen.getByTestId("agent-chat-seat-permission-read-only"),
      ),
    );
    fireEvent.keyDown(document, { key: "Escape" });
    await waitFor(() =>
      expect(screen.queryByTestId("agent-chat-seat-permission-menu")).not.toBeInTheDocument(),
    );
    fireEvent.keyDown(trigger, { key: "ArrowUp" });
    expect(screen.getByTestId("agent-chat-seat-permission-menu")).toBeInTheDocument();
    await waitFor(() =>
      expect(document.activeElement).toBe(screen.getByTestId("agent-chat-seat-permission-full")),
    );
  });

  it("closes model menu when focus leaves the seat root", async () => {
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
    fireEvent.click(await screen.findByTestId("agent-chat-seat-model-trigger"));
    expect(screen.getByTestId("agent-chat-seat-model-menu")).toBeInTheDocument();
    fireEvent.blur(screen.getByTestId("agent-chat-seat-model-open"), {
      relatedTarget: document.body,
    });
    await waitFor(() =>
      expect(screen.queryByTestId("agent-chat-seat-model-menu")).not.toBeInTheDocument(),
    );
  });

  it("closes permission menu when focus leaves the seat root", async () => {
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
    fireEvent.click(await screen.findByTestId("agent-chat-seat-permission"));
    expect(screen.getByTestId("agent-chat-seat-permission-menu")).toBeInTheDocument();
    fireEvent.blur(screen.getByTestId("agent-chat-seat-permission-read-only"), {
      relatedTarget: document.body,
    });
    await waitFor(() =>
      expect(screen.queryByTestId("agent-chat-seat-permission-menu")).not.toBeInTheDocument(),
    );
  });

  it("Escape backs from submenu to root, then closes; outside pointerdown closes", async () => {
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
    fireEvent.click(await screen.findByTestId("agent-chat-seat-model-trigger"));
    expect(screen.getByTestId("agent-chat-seat-model-menu")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("agent-chat-seat-effort-open"));
    expect(screen.getByTestId("agent-chat-seat-effort-max")).toBeInTheDocument();
    fireEvent.keyDown(document, { key: "Escape" });
    await waitFor(() => expect(screen.getByTestId("agent-chat-seat-effort-open")).toBeInTheDocument());
    expect(screen.queryByTestId("agent-chat-seat-effort-max")).not.toBeInTheDocument();
    await waitFor(() =>
      expect(document.activeElement).toBe(screen.getByTestId("agent-chat-seat-effort-open")),
    );
    fireEvent.keyDown(document, { key: "Escape" });
    await waitFor(() =>
      expect(screen.queryByTestId("agent-chat-seat-model-menu")).not.toBeInTheDocument(),
    );
    await waitFor(() =>
      expect(document.activeElement).toBe(screen.getByTestId("agent-chat-seat-model-trigger")),
    );
    fireEvent.click(screen.getByTestId("agent-chat-seat-model-trigger"));
    fireEvent.click(screen.getByTestId("agent-chat-seat-model-open"));
    expect(screen.getByTestId("agent-chat-seat-model-pick-static")).toBeInTheDocument();
    fireEvent.pointerDown(document.body);
    await waitFor(() =>
      expect(screen.queryByTestId("agent-chat-seat-model-menu")).not.toBeInTheDocument(),
    );
  });

  it("marks the current effort as menuitemradio aria-checked", async () => {
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
    fireEvent.click(await screen.findByTestId("agent-chat-seat-model-trigger"));
    fireEvent.click(screen.getByTestId("agent-chat-seat-effort-open"));
    expect(screen.getByTestId("agent-chat-seat-effort-high")).toHaveAttribute("aria-checked", "true");
    expect(screen.getByTestId("agent-chat-seat-effort-max")).toHaveAttribute("aria-checked", "false");
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
    await waitFor(() =>
      expect(document.activeElement).toBe(screen.getByTestId("agent-chat-seat-model-trigger")),
    );
  });

  it("re-selecting the current effort closes without PATCH and restores trigger focus", async () => {
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
    fireEvent.click(await screen.findByTestId("agent-chat-seat-model-trigger"));
    fireEvent.click(screen.getByTestId("agent-chat-seat-effort-open"));
    updateSession.mockClear();
    fireEvent.click(screen.getByTestId("agent-chat-seat-effort-high"));
    await waitFor(() =>
      expect(screen.queryByTestId("agent-chat-seat-model-menu")).not.toBeInTheDocument(),
    );
    expect(updateSession).not.toHaveBeenCalled();
    await waitFor(() =>
      expect(document.activeElement).toBe(screen.getByTestId("agent-chat-seat-model-trigger")),
    );
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
    expect(screen.getByTestId("agent-chat-seat-permission")).toHaveAttribute(
      "data-permission",
      "read-only",
    );
    expect(screen.getByTestId("agent-chat-seat-permission")).toHaveTextContent(/询问/);
    expect(screen.getByTestId("agent-chat-seat-permission")).toHaveAttribute(
      "title",
      expect.stringContaining("询问审批"),
    );
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
    fireEvent.click(screen.getByTestId("agent-chat-seat-permission"));
    fireEvent.click(screen.getByTestId("agent-chat-seat-permission-workspace-write"));
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
    fireEvent.click(screen.getByTestId("agent-chat-seat-permission"));
    fireEvent.click(screen.getByTestId("agent-chat-seat-permission-full"));
    expect(updateSession).not.toHaveBeenCalledWith("sess_1", { permissionMode: "full" });
    expect(screen.getByTestId("agent-chat-seat-permission-menu")).toBeInTheDocument();
    confirmSpy.mockRestore();
  });

  it("keeps permission menu and model menu mutually exclusive", async () => {
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
    fireEvent.click(await screen.findByTestId("agent-chat-seat-model-trigger"));
    expect(screen.getByTestId("agent-chat-seat-model-menu")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("agent-chat-seat-permission"));
    await waitFor(() =>
      expect(screen.queryByTestId("agent-chat-seat-model-menu")).not.toBeInTheDocument(),
    );
    expect(screen.getByTestId("agent-chat-seat-permission-menu")).toBeInTheDocument();
    const first = screen.getByTestId("agent-chat-seat-permission-read-only");
    const last = screen.getByTestId("agent-chat-seat-permission-full");
    fireEvent.keyDown(document, { key: "End" });
    expect(document.activeElement).toBe(last);
    fireEvent.keyDown(document, { key: "Home" });
    expect(document.activeElement).toBe(first);
    fireEvent.click(screen.getByTestId("agent-chat-seat-model-trigger"));
    await waitFor(() =>
      expect(screen.queryByTestId("agent-chat-seat-permission-menu")).not.toBeInTheDocument(),
    );
    expect(screen.getByTestId("agent-chat-seat-model-menu")).toBeInTheDocument();
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
    expect(plan).toHaveAttribute("aria-label", "绑定 Plan");
    fireEvent.change(plan, { target: { value: "plan_bad" } });
    fireEvent.blur(plan);
    expect(await screen.findByTestId("agent-chat-seats-error")).toHaveTextContent(
      "未找到该 Plan，请检查 planId",
    );
    await waitFor(() => expect(plan).toHaveValue("plan_ok"));
  });

  it("commits planId on Enter and reverts draft on Escape", async () => {
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
    fireEvent.change(plan, { target: { value: "plan_next" } });
    fireEvent.keyDown(plan, { key: "Enter" });
    await waitFor(() =>
      expect(updateSession).toHaveBeenCalledWith("sess_1", { planId: "plan_next" }),
    );
    fireEvent.change(plan, { target: { value: "plan_draft" } });
    fireEvent.keyDown(plan, { key: "Escape" });
    expect(plan).toHaveValue("plan_ok");
    expect(updateSession).not.toHaveBeenCalledWith("sess_1", { planId: "plan_draft" });
  });

  it("clears a bound planId from the plan seat chip control", async () => {
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
    expect(screen.getByTestId("agent-chat-seat-plan-clear")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("agent-chat-seat-plan-clear"));
    await waitFor(() =>
      expect(updateSession).toHaveBeenCalledWith("sess_1", { planId: "" }),
    );
    expect(screen.getByTestId("agent-chat-seat-plan")).toHaveValue("");
  });
});
