import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  attachAgentWorkspaceSession,
  closeAgentWorkspace,
  createAgentWorkspace,
  listAgentWorkspaces,
  patchAgentWorkspace,
} from "./workspace.api";

const api = vi.fn();

vi.mock("@/services/http/client", () => ({
  api: (...args: unknown[]) => api(...args),
}));

describe("workspace.api", () => {
  beforeEach(() => {
    api.mockReset();
    api.mockResolvedValue({ items: [] });
  });

  it("lists with limit", async () => {
    await listAgentWorkspaces({ limit: 20 });
    expect(api).toHaveBeenCalledWith("/agent-workspaces?limit=20");
  });

  it("creates workspace", async () => {
    api.mockResolvedValue({ id: "aw_1", title: "A", sessionIds: [], status: "active" });
    await createAgentWorkspace({ title: "A", repoRoot: "." });
    expect(api).toHaveBeenCalledWith("/agent-workspaces", {
      method: "POST",
      body: JSON.stringify({ title: "A", repoRoot: "." }),
    });
  });

  it("patches title", async () => {
    api.mockResolvedValue({ id: "aw_1", title: "B" });
    await patchAgentWorkspace("aw_1", { title: "B" });
    expect(api).toHaveBeenCalledWith("/agent-workspaces/aw_1", {
      method: "PATCH",
      body: JSON.stringify({ title: "B" }),
    });
  });

  it("attaches session", async () => {
    api.mockResolvedValue({ id: "aw_1", sessionIds: ["sess_1"] });
    await attachAgentWorkspaceSession("aw_1", "sess_1");
    expect(api).toHaveBeenCalledWith("/agent-workspaces/aw_1/sessions", {
      method: "POST",
      body: JSON.stringify({ sessionId: "sess_1" }),
    });
  });

  it("soft-closes workspace", async () => {
    api.mockResolvedValue({ id: "aw_1", status: "closed" });
    await closeAgentWorkspace("aw_1");
    expect(api).toHaveBeenCalledWith("/agent-workspaces/aw_1", { method: "DELETE" });
  });
});
