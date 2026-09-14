import { describe, expect, it } from "vitest";
import type { AgentSessionView } from "../api/session.api";
import type { AgentWorkspaceView } from "../api/workspace.api";
import { groupSessionsByWorkspace } from "./workspaceGroups";

function sess(partial: Partial<AgentSessionView> & { id: string }): AgentSessionView {
  return {
    spaceId: "local",
    status: "active",
    ...partial,
  };
}

function ws(partial: Partial<AgentWorkspaceView> & { id: string }): AgentWorkspaceView {
  return {
    spaceId: "local",
    title: "WS",
    sessionIds: [],
    status: "active",
    ...partial,
  };
}

describe("groupSessionsByWorkspace", () => {
  it("groups by sessionIds and leaves unassigned", () => {
    const workspaces = [
      ws({ id: "aw_1", title: "Feat", sessionIds: ["sess_a", "sess_b"] }),
    ];
    const sessions = [
      sess({ id: "sess_a", workspaceId: "aw_1" }),
      sess({ id: "sess_b" }),
      sess({ id: "sess_c" }),
    ];
    const groups = groupSessionsByWorkspace(sessions, workspaces);
    expect(groups).toHaveLength(2);
    expect(groups[0].workspace?.id).toBe("aw_1");
    expect(groups[0].sessions.map((s) => s.id)).toEqual(["sess_a", "sess_b"]);
    expect(groups[1].workspace).toBeNull();
    expect(groups[1].sessions.map((s) => s.id)).toEqual(["sess_c"]);
  });

  it("picks up workspaceId membership missing from sessionIds", () => {
    const workspaces = [ws({ id: "aw_1", sessionIds: [] })];
    const sessions = [sess({ id: "sess_x", workspaceId: "aw_1" })];
    const groups = groupSessionsByWorkspace(sessions, workspaces);
    expect(groups[0].sessions.map((s) => s.id)).toEqual(["sess_x"]);
  });

  it("shows unassigned bucket when no workspaces", () => {
    const sessions = [sess({ id: "sess_a" })];
    const groups = groupSessionsByWorkspace(sessions, []);
    expect(groups).toHaveLength(1);
    expect(groups[0].workspace).toBeNull();
    expect(groups[0].sessions).toHaveLength(1);
  });
});
