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

  it("keeps prompt textarea and send actions in a dedicated input shell", () => {
    wrap(<IntentBar mode="prompt" busy={false} onIntent={vi.fn()} />);
    const field = screen.getByTestId("agent-intent-field");
    const prompt = screen.getByTestId("agent-intent-prompt");
    const send = screen.getByTestId("agent-intent-send");
    expect(field).toContainElement(prompt);
    expect(field).toContainElement(screen.getByTestId("agent-intent-toolbar"));
    expect(screen.getByTestId("agent-intent-toolbar")).toContainElement(send);
    expect(prompt).toHaveAttribute("aria-label", "意图");
    expect(prompt.tagName).toBe("TEXTAREA");
    expect(prompt).toHaveClass("agent-intent-input");
  });

  it("explains why send stays disabled for empty or whitespace prompt", () => {
    wrap(<IntentBar mode="prompt" busy={false} onIntent={vi.fn()} />);
    const send = screen.getByTestId("agent-intent-send");
    expect(send).toBeDisabled();
    expect(send).toHaveAttribute("title", "需要输入内容");
    fireEvent.change(screen.getByTestId("agent-intent-prompt"), { target: { value: "   " } });
    expect(send).toBeDisabled();
    expect(send).toHaveAttribute("title", "需要输入内容");
    fireEvent.change(screen.getByTestId("agent-intent-prompt"), { target: { value: "hi" } });
    expect(send).not.toBeDisabled();
    expect(send).toHaveAttribute("title", "发送");
  });

  it("disables send for bare slash until a command is chosen", async () => {
    const onIntent = vi.fn();
    wrap(<IntentBar mode="prompt" busy={false} onIntent={onIntent} />);
    const send = screen.getByTestId("agent-intent-send");
    fireEvent.change(screen.getByTestId("agent-intent-prompt"), { target: { value: "/" } });
    await waitFor(() => expect(screen.getByTestId("agent-command-help")).toBeInTheDocument());
    expect(send).toBeDisabled();
    expect(send).toHaveAttribute("title", "选择命令或继续输入");
    fireEvent.click(send);
    expect(onIntent).not.toHaveBeenCalled();
    fireEvent.change(screen.getByTestId("agent-intent-prompt"), { target: { value: "/help" } });
    expect(send).not.toBeDisabled();
    fireEvent.click(send);
    expect(onIntent).toHaveBeenCalledWith({ action: "command", command: "/help", args: undefined });
  });

  it("shows four gate buttons: once / session / reject / cancel run", async () => {
    const onIntent = vi.fn();
    wrap(
      <IntentBar
        mode="gate"
        busy={false}
        gateReason="need human review"
        gateTool="danger.tool"
        onIntent={onIntent}
      />,
    );
    expect(screen.getByTestId("agent-intent-bar")).toHaveAttribute("data-mode", "gate");
    expect(screen.getByTestId("agent-intent-gate-reason")).toHaveAttribute("role", "status");
    expect(screen.getByTestId("agent-intent-allow-once")).toHaveAttribute(
      "title",
      "允许一次执行 danger.tool",
    );
    expect(screen.getByTestId("agent-intent-reject")).toHaveAttribute("title", "拒绝本次工具调用");
    expect(screen.getByTestId("agent-intent-cancel")).toHaveAttribute("title", "取消当前 Run");
    await waitFor(() =>
      expect(document.activeElement).toBe(screen.getByTestId("agent-intent-allow-once")),
    );
    fireEvent.keyDown(document, { key: "ArrowRight" });
    expect(document.activeElement).toBe(screen.getByTestId("agent-intent-allow-session"));
    fireEvent.keyDown(document, { key: "End" });
    expect(document.activeElement).toBe(screen.getByTestId("agent-intent-cancel"));
    fireEvent.keyDown(document, { key: "Home" });
    expect(document.activeElement).toBe(screen.getByTestId("agent-intent-allow-once"));
    fireEvent.click(screen.getByTestId("agent-intent-allow-once"));
    expect(onIntent).toHaveBeenCalledWith({
      action: "approve",
      scope: "once",
      tool: "danger.tool",
      reason: "allow once from IntentBar",
    });
    fireEvent.click(screen.getByTestId("agent-intent-allow-session"));
    expect(onIntent).toHaveBeenCalledWith({
      action: "approve",
      scope: "session",
      tool: "danger.tool",
      reason: "allow session from IntentBar",
    });
    fireEvent.click(screen.getByTestId("agent-intent-reject"));
    expect(onIntent).toHaveBeenCalledWith({ action: "reject", reason: "rejected from IntentBar" });
    fireEvent.click(screen.getByTestId("agent-intent-cancel"));
    expect(onIntent).toHaveBeenCalledWith({ action: "cancel" });
  });

  it("shows stop in prompt mode when canStop", () => {
    const onIntent = vi.fn();
    wrap(<IntentBar mode="prompt" canStop busy={false} onIntent={onIntent} />);
    fireEvent.click(screen.getByTestId("agent-intent-stop"));
    expect(onIntent).toHaveBeenCalledWith({ action: "stop" });
  });

  it("sends steer intent when canStop (running)", () => {
    const onIntent = vi.fn();
    wrap(<IntentBar mode="prompt" canStop busy={false} onIntent={onIntent} />);
    expect(screen.getByTestId("agent-intent-bar")).toHaveAttribute("data-steer", "true");
    expect(screen.getByTestId("agent-intent-steer-hint")).toHaveAttribute("role", "status");
    expect(screen.getByTestId("agent-intent-steer-hint")).toBeInTheDocument();
    expect(screen.getByTestId("agent-intent-send")).toHaveTextContent("续写");
    fireEvent.change(screen.getByTestId("agent-intent-prompt"), {
      target: { value: "go another way" },
    });
    fireEvent.click(screen.getByTestId("agent-intent-send"));
    expect(onIntent).toHaveBeenCalledWith({ action: "steer", prompt: "go another way" });
  });

  it("shows queue chip and enqueues without stop clearing it", () => {
    const onIntent = vi.fn();
    wrap(
      <IntentBar
        mode="prompt"
        canStop
        busy={false}
        queueItems={["after this"]}
        onIntent={onIntent}
      />,
    );
    const chip = screen.getByTestId("agent-intent-queue-chip");
    expect(chip).toHaveAttribute("role", "status");
    expect(chip).toHaveTextContent("队列 1");
    expect(chip).toHaveTextContent("after this");
    expect(chip).toHaveTextContent("停止只结束当前");
    expect(screen.getByTestId("agent-intent-bar")).toHaveAttribute("data-queue-count", "1");
    fireEvent.change(screen.getByTestId("agent-intent-prompt"), { target: { value: "next please" } });
    fireEvent.click(screen.getByTestId("agent-intent-queue"));
    expect(onIntent).toHaveBeenCalledWith({ action: "queue", prompt: "next please" });
    fireEvent.click(screen.getByTestId("agent-intent-stop"));
    expect(onIntent).toHaveBeenCalledWith({ action: "stop" });
    expect(onIntent).not.toHaveBeenCalledWith(expect.objectContaining({ action: "steer" }));
  });

  it("alt+enter queues while running and enter still steers", () => {
    const onIntent = vi.fn();
    wrap(<IntentBar mode="prompt" canStop busy={false} onIntent={onIntent} />);
    const area = screen.getByTestId("agent-intent-prompt");
    fireEvent.change(area, { target: { value: "hold" } });
    fireEvent.keyDown(area, { key: "Enter", altKey: true });
    expect(onIntent).toHaveBeenCalledWith({ action: "queue", prompt: "hold" });
    fireEvent.change(area, { target: { value: "interrupt" } });
    fireEvent.keyDown(area, { key: "Enter", altKey: false });
    expect(onIntent).toHaveBeenCalledWith({ action: "steer", prompt: "interrupt" });
  });

  it("opens command menu on slash and fills without sending", async () => {
    const onIntent = vi.fn();
    wrap(<IntentBar mode="prompt" busy={false} onIntent={onIntent} />);
    fireEvent.click(screen.getByTestId("agent-composer-plus"));
    fireEvent.click(screen.getByTestId("agent-intent-slash"));
    await waitFor(() => expect(screen.getByTestId("agent-command-help")).toBeInTheDocument());
    expect(listAgentCommands).toHaveBeenCalled();
    expect(screen.getByTestId("agent-command-help")).toHaveAttribute(
      "title",
      "填入 /help（不自动发送）",
    );
    fireEvent.click(screen.getByTestId("agent-command-help"));
    expect(onIntent).not.toHaveBeenCalled();
    expect(screen.getByTestId("agent-intent-prompt")).toHaveValue("/help ");
  });

  it("wires aria-controls from the plus trigger to the open menu", () => {
    wrap(<IntentBar mode="prompt" busy={false} onIntent={vi.fn()} />);
    const trigger = screen.getByTestId("agent-composer-plus");
    expect(trigger).not.toHaveAttribute("aria-controls");
    fireEvent.click(trigger);
    const menu = screen.getByTestId("agent-composer-plus-menu");
    expect(trigger).toHaveAttribute("aria-controls", menu.id);
    expect(menu.id).toBeTruthy();
  });

  it("focuses the first plus menu item on open and ArrowDown/ArrowUp/Home/End cycles", async () => {
    wrap(
      <IntentBar
        mode="prompt"
        busy={false}
        onIntent={vi.fn()}
        plusItems={
          <>
            <li>
              <button type="button" role="menuitem" data-testid="agent-plus-extra">
                附件
              </button>
            </li>
          </>
        }
      />,
    );
    fireEvent.click(screen.getByTestId("agent-composer-plus"));
    const slash = screen.getByTestId("agent-intent-slash");
    const extra = screen.getByTestId("agent-plus-extra");
    await waitFor(() => expect(document.activeElement).toBe(slash));
    fireEvent.keyDown(document, { key: "ArrowDown" });
    expect(document.activeElement).toBe(extra);
    fireEvent.keyDown(document, { key: "ArrowUp" });
    expect(document.activeElement).toBe(slash);
    fireEvent.keyDown(document, { key: "End" });
    expect(document.activeElement).toBe(extra);
    fireEvent.keyDown(document, { key: "Home" });
    expect(document.activeElement).toBe(slash);
  });

  it("moves focus to the prompt after choosing 命令 from the plus menu", async () => {
    wrap(<IntentBar mode="prompt" busy={false} onIntent={vi.fn()} />);
    fireEvent.click(screen.getByTestId("agent-composer-plus"));
    fireEvent.click(screen.getByTestId("agent-intent-slash"));
    await waitFor(() =>
      expect(screen.queryByTestId("agent-composer-plus-menu")).not.toBeInTheDocument(),
    );
    await waitFor(() =>
      expect(document.activeElement).toBe(screen.getByTestId("agent-intent-prompt")),
    );
    expect(screen.getByTestId("agent-intent-prompt")).toHaveValue("/");
    await waitFor(() => expect(screen.getByTestId("agent-command-menu")).toBeInTheDocument());
  });

  it("closes the plus menu on Escape and outside pointerdown", async () => {
    wrap(<IntentBar mode="prompt" busy={false} onIntent={vi.fn()} />);
    fireEvent.click(screen.getByTestId("agent-composer-plus"));
    expect(screen.getByTestId("agent-composer-plus-menu")).toBeInTheDocument();
    fireEvent.keyDown(document, { key: "Escape" });
    await waitFor(() =>
      expect(screen.queryByTestId("agent-composer-plus-menu")).not.toBeInTheDocument(),
    );
    await waitFor(() =>
      expect(document.activeElement).toBe(screen.getByTestId("agent-composer-plus")),
    );
    fireEvent.click(screen.getByTestId("agent-composer-plus"));
    expect(screen.getByTestId("agent-composer-plus-menu")).toBeInTheDocument();
    fireEvent.pointerDown(document.body);
    await waitFor(() =>
      expect(screen.queryByTestId("agent-composer-plus-menu")).not.toBeInTheDocument(),
    );
  });

  it("jumps plus menu items via typeahead and marks vertical orientation", async () => {
    wrap(
      <IntentBar
        mode="prompt"
        busy={false}
        onIntent={vi.fn()}
        plusItems={
          <>
            <li>
              <button type="button" role="menuitem" data-testid="agent-plus-extra">
                Tools
              </button>
            </li>
          </>
        }
      />,
    );
    fireEvent.click(screen.getByTestId("agent-composer-plus"));
    const menu = screen.getByTestId("agent-composer-plus-menu");
    expect(menu).toHaveAttribute("aria-orientation", "vertical");
    await waitFor(() =>
      expect(document.activeElement).toBe(screen.getByTestId("agent-intent-slash")),
    );
    fireEvent.keyDown(document, { key: "t" });
    expect(document.activeElement).toBe(screen.getByTestId("agent-plus-extra"));
  });

  it("opens the plus menu from the trigger with ArrowDown/ArrowUp", async () => {
    wrap(
      <IntentBar
        mode="prompt"
        busy={false}
        onIntent={vi.fn()}
        plusItems={
          <>
            <li>
              <button type="button" role="menuitem" data-testid="agent-plus-extra">
                附件
              </button>
            </li>
          </>
        }
      />,
    );
    const trigger = screen.getByTestId("agent-composer-plus");
    fireEvent.keyDown(trigger, { key: "ArrowDown" });
    expect(screen.getByTestId("agent-composer-plus-menu")).toBeInTheDocument();
    await waitFor(() =>
      expect(document.activeElement).toBe(screen.getByTestId("agent-intent-slash")),
    );
    fireEvent.keyDown(document, { key: "Escape" });
    await waitFor(() =>
      expect(screen.queryByTestId("agent-composer-plus-menu")).not.toBeInTheDocument(),
    );
    fireEvent.keyDown(trigger, { key: "ArrowUp" });
    expect(screen.getByTestId("agent-composer-plus-menu")).toBeInTheDocument();
    await waitFor(() =>
      expect(document.activeElement).toBe(screen.getByTestId("agent-plus-extra")),
    );
  });

  it("closes the plus menu when focus leaves the + seat", async () => {
    wrap(<IntentBar mode="prompt" busy={false} onIntent={vi.fn()} />);
    fireEvent.click(screen.getByTestId("agent-composer-plus"));
    expect(screen.getByTestId("agent-composer-plus-menu")).toBeInTheDocument();
    fireEvent.blur(screen.getByTestId("agent-intent-slash"), {
      relatedTarget: document.body,
    });
    await waitFor(() =>
      expect(screen.queryByTestId("agent-composer-plus-menu")).not.toBeInTheDocument(),
    );
  });

  it("dismisses slash menu when prompt blurs outside the listbox", async () => {
    wrap(<IntentBar mode="prompt" busy={false} onIntent={vi.fn()} />);
    const area = screen.getByTestId("agent-intent-prompt");
    fireEvent.change(area, { target: { value: "/" } });
    await screen.findByTestId("agent-command-menu");
    fireEvent.blur(area, { relatedTarget: screen.getByTestId("agent-intent-send") });
    await waitFor(() =>
      expect(screen.queryByTestId("agent-command-menu")).not.toBeInTheDocument(),
    );
    expect(area).toHaveValue("/");
  });

  it("keeps plus menu and slash command menu mutually exclusive", async () => {
    wrap(<IntentBar mode="prompt" busy={false} onIntent={vi.fn()} />);
    fireEvent.click(screen.getByTestId("agent-composer-plus"));
    expect(screen.getByTestId("agent-composer-plus-menu")).toBeInTheDocument();
    fireEvent.change(screen.getByTestId("agent-intent-prompt"), { target: { value: "/" } });
    await waitFor(() => expect(screen.getByTestId("agent-command-menu")).toBeInTheDocument());
    expect(screen.queryByTestId("agent-composer-plus-menu")).not.toBeInTheDocument();
    fireEvent.click(screen.getByTestId("agent-composer-plus"));
    expect(screen.getByTestId("agent-composer-plus-menu")).toBeInTheDocument();
    await waitFor(() =>
      expect(screen.queryByTestId("agent-command-menu")).not.toBeInTheDocument(),
    );
  });

  it("dismisses slash command menu on Escape without clearing the draft", async () => {
    wrap(<IntentBar mode="prompt" busy={false} onIntent={vi.fn()} />);
    fireEvent.change(screen.getByTestId("agent-intent-prompt"), { target: { value: "/" } });
    await waitFor(() => expect(screen.getByTestId("agent-command-menu")).toBeInTheDocument());
    fireEvent.keyDown(document, { key: "Escape" });
    await waitFor(() =>
      expect(screen.queryByTestId("agent-command-menu")).not.toBeInTheDocument(),
    );
    expect(screen.getByTestId("agent-intent-prompt")).toHaveValue("/");
    await waitFor(() =>
      expect(document.activeElement).toBe(screen.getByTestId("agent-intent-prompt")),
    );
    fireEvent.change(screen.getByTestId("agent-intent-prompt"), { target: { value: "/h" } });
    await waitFor(() => expect(screen.getByTestId("agent-command-menu")).toBeInTheDocument());
    fireEvent.pointerDown(document.body);
    await waitFor(() =>
      expect(screen.queryByTestId("agent-command-menu")).not.toBeInTheDocument(),
    );
  });

  it("requires confirm before sending /clear command", () => {
    const onIntent = vi.fn();
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    wrap(<IntentBar mode="prompt" busy={false} onIntent={onIntent} />);
    fireEvent.change(screen.getByTestId("agent-intent-prompt"), {
      target: { value: "/clear now" },
    });
    const send = screen.getByTestId("agent-intent-send");
    expect(send).toHaveAttribute("title", "执行 /clear（需确认）");
    fireEvent.click(send);
    expect(confirmSpy).toHaveBeenCalled();
    expect(onIntent).not.toHaveBeenCalled();

    confirmSpy.mockReturnValue(true);
    fireEvent.click(send);
    expect(onIntent).toHaveBeenCalledWith({
      action: "command",
      command: "/clear",
      args: "now",
    });
    confirmSpy.mockRestore();
  });

  it("sends non-destructive slash commands without confirm", () => {
    const onIntent = vi.fn();
    const confirmSpy = vi.spyOn(window, "confirm");
    wrap(<IntentBar mode="prompt" busy={false} onIntent={onIntent} />);
    fireEvent.change(screen.getByTestId("agent-intent-prompt"), {
      target: { value: "/help" },
    });
    fireEvent.click(screen.getByTestId("agent-intent-send"));
    expect(confirmSpy).not.toHaveBeenCalled();
    expect(onIntent).toHaveBeenCalledWith({
      action: "command",
      command: "/help",
      args: undefined,
    });
    confirmSpy.mockRestore();
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

  it("wires slash listbox aria-controls and aria-activedescendant on the prompt", async () => {
    HTMLElement.prototype.scrollIntoView = vi.fn();
    wrap(<IntentBar mode="prompt" busy={false} onIntent={vi.fn()} />);
    const area = screen.getByTestId("agent-intent-prompt");
    expect(area).toHaveAttribute("aria-haspopup", "listbox");
    expect(area).not.toHaveAttribute("aria-controls");
    expect(area).toHaveAttribute("aria-expanded", "false");
    fireEvent.change(area, { target: { value: "/" } });
    const menu = await screen.findByTestId("agent-command-menu");
    expect(area).toHaveAttribute("aria-controls", menu.id);
    expect(area).toHaveAttribute("aria-expanded", "true");
    expect(area).toHaveAttribute("aria-autocomplete", "list");
    expect(menu).toHaveAttribute("aria-orientation", "vertical");
    const help = await screen.findByTestId("agent-command-help");
    expect(menu).not.toHaveAttribute("aria-busy");
    expect(area).toHaveAttribute("aria-activedescendant", help.id);
    await waitFor(() => expect(HTMLElement.prototype.scrollIntoView).toHaveBeenCalled());
    vi.mocked(HTMLElement.prototype.scrollIntoView).mockClear();
    fireEvent.keyDown(area, { key: "ArrowDown" });
    const clear = screen.getByTestId("agent-command-clear");
    expect(area).toHaveAttribute("aria-activedescendant", clear.id);
    await waitFor(() => expect(HTMLElement.prototype.scrollIntoView).toHaveBeenCalled());
    fireEvent.keyDown(area, { key: "Home" });
    expect(area).toHaveAttribute("aria-activedescendant", help.id);
    fireEvent.keyDown(area, { key: "End" });
    expect(area).toHaveAttribute("aria-activedescendant", clear.id);
  });

  it("marks slash listbox aria-busy while commands load", async () => {
    listAgentCommands.mockReturnValueOnce(new Promise(() => {}));
    wrap(<IntentBar mode="prompt" busy={false} onIntent={vi.fn()} />);
    fireEvent.change(screen.getByTestId("agent-intent-prompt"), { target: { value: "/" } });
    const menu = await screen.findByTestId("agent-command-menu");
    expect(menu).toHaveAttribute("aria-busy", "true");
    expect(screen.getByRole("status")).toHaveTextContent("加载命令…");
  });

  it("announces empty slash matches with role=status", async () => {
    wrap(<IntentBar mode="prompt" busy={false} onIntent={vi.fn()} />);
    fireEvent.change(screen.getByTestId("agent-intent-prompt"), {
      target: { value: "/zzznomatch" },
    });
    await screen.findByTestId("agent-command-menu");
    await waitFor(() =>
      expect(screen.getByRole("status")).toHaveTextContent("无匹配命令"),
    );
  });
});
