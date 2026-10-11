import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { AgentSessionView } from "../api/session.api";
import type { AgentWorkspaceView } from "../api/workspace.api";
import { SessionHistoryList } from "./SessionHistoryList";

function session(partial: Partial<AgentSessionView> & { id: string }): AgentSessionView {
  return {
    spaceId: "local",
    status: "open",
    title: partial.title ?? partial.id,
    ...partial,
  };
}

function workspace(partial: Partial<AgentWorkspaceView> & { id: string }): AgentWorkspaceView {
  return {
    spaceId: "local",
    title: partial.title ?? partial.id,
    sessionIds: partial.sessionIds ?? [],
    status: "open",
    ...partial,
  };
}

describe("SessionHistoryList keyboard", () => {
  it("moves focus among workspace toggles and session rows", () => {
    const ws = workspace({ id: "aw_1", title: "主仓", sessionIds: ["s1", "s2"] });
    const items = [
      session({ id: "s1", title: "一", workspaceId: "aw_1" }),
      session({ id: "s2", title: "二", workspaceId: "aw_1" }),
    ];
    render(
      <SessionHistoryList
        items={items}
        workspaces={[ws]}
        selectedSessionId="s1"
        activeWorkspaceId="aw_1"
        onSelect={vi.fn()}
        onSelectWorkspace={vi.fn()}
        onNew={vi.fn()}
        onNewWorkspace={vi.fn()}
      />,
    );

    const list = screen.getByTestId("agent-workspace-list");
    expect(list).toHaveAttribute("role", "navigation");
    expect(list).toHaveAttribute("aria-label", "工作区与会话");

    const toggle = screen.getByTestId("agent-workspace-toggle-aw_1");
    const s1 = screen.getByTestId("agent-history-item-s1");
    const s2 = screen.getByTestId("agent-history-item-s2");
    toggle.focus();
    expect(toggle).toHaveFocus();

    fireEvent.keyDown(toggle, { key: "ArrowDown" });
    expect(s1).toHaveFocus();

    fireEvent.keyDown(s1, { key: "ArrowDown" });
    expect(s2).toHaveFocus();

    fireEvent.keyDown(s2, { key: "Home" });
    expect(toggle).toHaveFocus();

    fireEvent.keyDown(toggle, { key: "End" });
    expect(s2).toHaveFocus();

    fireEvent.keyDown(s2, { key: "ArrowUp" });
    expect(s1).toHaveFocus();
  });
});
