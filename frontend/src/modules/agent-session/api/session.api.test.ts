import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  closeAgentSession,
  eventVisibility,
  listAgentCommands,
  listAgentModels,
  listAgentSessions,
  patchAgentSession,
  purgeAgentSession,
  updateSession,
} from "./session.api";

const api = vi.fn();

vi.mock("@/services/http/client", () => ({
  api: (...args: unknown[]) => api(...args),
}));

describe("eventVisibility", () => {
  it("defaults session.turn to model_visible", () => {
    expect(eventVisibility({ type: "session.turn" })).toBe("model_visible");
  });
  it("respects explicit ui_only", () => {
    expect(eventVisibility({ type: "session.turn", visibility: "ui_only" })).toBe("ui_only");
  });
  it("maps gate.waiting_approval", () => {
    expect(eventVisibility({ type: "gate.waiting_approval" })).toBe("ui_only");
  });
  it("maps tool and step events to ui_only by default", () => {
    expect(eventVisibility({ type: "tool.called" })).toBe("ui_only");
    expect(eventVisibility({ type: "step.finished" })).toBe("ui_only");
  });
});

describe("listAgentSessions", () => {
  beforeEach(() => {
    api.mockReset();
    api.mockResolvedValue({ items: [] });
  });

  it("calls GET /agents/sessions with optional limit", async () => {
    await listAgentSessions({ limit: 20 });
    expect(api).toHaveBeenCalledWith("/agents/sessions?limit=20");
  });

  it("passes includeClosed=1 when requested", async () => {
    await listAgentSessions({ includeClosed: true });
    expect(api).toHaveBeenCalledWith("/agents/sessions?includeClosed=1");
  });
});

describe("commands and models", () => {
  beforeEach(() => {
    api.mockReset();
    api.mockResolvedValue({ items: [] });
  });

  it("GETs /agents/commands", async () => {
    await listAgentCommands();
    expect(api).toHaveBeenCalledWith("/agents/commands");
  });

  it("GETs /agents/models", async () => {
    await listAgentModels();
    expect(api).toHaveBeenCalledWith("/agents/models");
  });
});

describe("patch/close session", () => {
  beforeEach(() => {
    api.mockReset();
    api.mockResolvedValue({ id: "sess_1", title: "x", status: "active" });
  });

  it("PATCHes title", async () => {
    await patchAgentSession("sess_1", { title: "Renamed" });
    expect(api).toHaveBeenCalledWith("/agents/sessions/sess_1", {
      method: "PATCH",
      body: JSON.stringify({ title: "Renamed" }),
    });
  });

  it("updateSession aliases PATCH seats", async () => {
    await updateSession("sess_1", { providerKind: "static", permissionMode: "full" });
    expect(api).toHaveBeenCalledWith("/agents/sessions/sess_1", {
      method: "PATCH",
      body: JSON.stringify({ providerKind: "static", permissionMode: "full" }),
    });
  });

  it("DELETEs to soft-close", async () => {
    await closeAgentSession("sess_1");
    expect(api).toHaveBeenCalledWith("/agents/sessions/sess_1", { method: "DELETE" });
  });

  it("DELETEs with purge=1 for hard delete", async () => {
    api.mockResolvedValue({ id: "sess_1", purged: true });
    await purgeAgentSession("sess_1");
    expect(api).toHaveBeenCalledWith("/agents/sessions/sess_1?purge=1", { method: "DELETE" });
  });
});
