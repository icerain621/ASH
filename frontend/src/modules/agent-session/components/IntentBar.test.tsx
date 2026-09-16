import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { IntentBar } from "./IntentBar";

const listAgentCommands = vi.hoisted(() => vi.fn());

vi.mock("../api/session.api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/session.api")>();
  return {
    ...actual,
    listAgentCommands: (...args: unknown[]) => listAgentCommands(...args),
  };
});

function wrap(ui: ReactNode) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(<QueryClientProvider client={qc}>{ui}</QueryClientProvider>);
}

describe("IntentBar", () => {
  beforeEach(() => {
    listAgentCommands.mockReset();
    listAgentCommands.mockResolvedValue({
      items: [
        { name: "/help", description: "列出可用命令", source: "builtin" },
        { name: "/clear", description: "清空当前会话", source: "builtin" },
      ],
    });
  });

  it("submits prompt intent in agent mode", () => {
    const onIntent = vi.fn();
    wrap(<IntentBar mode="prompt" busy={false} onIntent={onIntent} />);
    fireEvent.change(screen.getByTestId("agent-intent-prompt"), { target: { value: "continue" } });
    fireEvent.click(screen.getByTestId("agent-intent-send"));
    expect(onIntent).toHaveBeenCalledWith({ action: "prompt", prompt: "continue" });
  });

  it("shows approve/cancel takeover when waiting approval", () => {
    const onIntent = vi.fn();
    wrap(
      <IntentBar
        mode="gate"
        busy={false}
        gateReason="need human review"
        onIntent={onIntent}
      />,
    );
    expect(screen.getByTestId("agent-intent-bar")).toHaveAttribute("data-mode", "gate");
    fireEvent.click(screen.getByTestId("agent-intent-approve"));
    expect(onIntent).toHaveBeenCalledWith({ action: "approve", reason: "approved from IntentBar" });
    fireEvent.click(screen.getByTestId("agent-intent-cancel"));
    expect(onIntent).toHaveBeenCalledWith({ action: "cancel" });
    fireEvent.click(screen.getByTestId("agent-intent-stop"));
    expect(onIntent).toHaveBeenCalledWith({ action: "stop" });
  });

  it("shows stop in prompt mode when canStop", () => {
    const onIntent = vi.fn();
    wrap(<IntentBar mode="prompt" canStop busy={false} onIntent={onIntent} />);
    fireEvent.click(screen.getByTestId("agent-intent-stop"));
    expect(onIntent).toHaveBeenCalledWith({ action: "stop" });
  });

  it("opens command menu on slash and fills without sending", async () => {
    const onIntent = vi.fn();
    wrap(<IntentBar mode="prompt" busy={false} onIntent={onIntent} />);
    fireEvent.click(screen.getByTestId("agent-intent-slash"));
    await waitFor(() => expect(screen.getByTestId("agent-command-help")).toBeInTheDocument());
    expect(listAgentCommands).toHaveBeenCalled();
    fireEvent.click(screen.getByTestId("agent-command-help"));
    expect(onIntent).not.toHaveBeenCalled();
    expect(screen.getByTestId("agent-intent-prompt")).toHaveValue("/help ");
  });

  it("sends leading slash text as command action", () => {
    const onIntent = vi.fn();
    wrap(<IntentBar mode="prompt" busy={false} onIntent={onIntent} />);
    fireEvent.change(screen.getByTestId("agent-intent-prompt"), {
      target: { value: "/clear now" },
    });
    fireEvent.click(screen.getByTestId("agent-intent-send"));
    expect(onIntent).toHaveBeenCalledWith({
      action: "command",
      command: "/clear",
      args: "now",
    });
  });

  it("sends on Enter and inserts newline on Shift+Enter", () => {
    const onIntent = vi.fn();
    wrap(<IntentBar mode="prompt" busy={false} onIntent={onIntent} />);
    const area = screen.getByTestId("agent-intent-prompt");
    fireEvent.change(area, { target: { value: "hello" } });
    fireEvent.keyDown(area, { key: "Enter", shiftKey: true });
    expect(onIntent).not.toHaveBeenCalled();
    fireEvent.keyDown(area, { key: "Enter", shiftKey: false });
    expect(onIntent).toHaveBeenCalledWith({ action: "prompt", prompt: "hello" });
  });

  it("fills highlighted slash command with Enter without sending", async () => {
    const onIntent = vi.fn();
    wrap(<IntentBar mode="prompt" busy={false} onIntent={onIntent} />);
    const area = screen.getByTestId("agent-intent-prompt");
    fireEvent.change(area, { target: { value: "/" } });
    await waitFor(() => expect(screen.getByTestId("agent-command-help")).toBeInTheDocument());
    fireEvent.keyDown(area, { key: "ArrowDown" });
    fireEvent.keyDown(area, { key: "Enter", shiftKey: false });
    expect(onIntent).not.toHaveBeenCalled();
    expect(area).toHaveValue("/clear ");
  });
});
