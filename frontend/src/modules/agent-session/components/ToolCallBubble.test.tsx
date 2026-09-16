import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { ToolCallBubble } from "./ToolCallBubble";
import type { MergedChatBubble } from "./conversationNodes";

function toolItem(partial: Partial<MergedChatBubble> = {}): MergedChatBubble {
  return {
    id: "tool:tc_1",
    role: "tool",
    type: "tool.card",
    kind: "tool.card",
    title: "工具 · git.status",
    summary: "← {}\n→ clean",
    toolStatus: "ok",
    toolName: "git.status",
    toolInput: "{}",
    toolOutput: "clean",
    ...partial,
  };
}

describe("ToolCallBubble", () => {
  it("starts collapsed and expands IN/OUT", () => {
    const onSelect = vi.fn();
    render(<ToolCallBubble item={toolItem()} onSelect={onSelect} />);
    expect(screen.getByTestId("agent-chat-bubble-tool")).toHaveAttribute("data-expanded", "0");
    expect(screen.queryByTestId("agent-tool-io")).toBeNull();
    fireEvent.click(screen.getByTestId("agent-tool-toggle"));
    expect(screen.getByTestId("agent-chat-bubble-tool")).toHaveAttribute("data-expanded", "1");
    expect(screen.getByTestId("agent-tool-input")).toHaveTextContent("{}");
    expect(screen.getByTestId("agent-tool-output")).toHaveTextContent("clean");
    fireEvent.click(screen.getByTestId("agent-chat-bubble-tool"));
    expect(onSelect).toHaveBeenCalled();
  });

  it("auto-expands error cards", () => {
    render(
      <ToolCallBubble
        item={toolItem({ toolStatus: "error", toolOutput: "boom", summary: "→ boom" })}
      />,
    );
    expect(screen.getByTestId("agent-chat-bubble-tool")).toHaveAttribute("data-expanded", "1");
    expect(screen.getByTestId("agent-tool-io")).toBeTruthy();
  });
});
