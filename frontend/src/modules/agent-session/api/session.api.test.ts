import { beforeEach, describe, expect, it, vi } from "vitest";
import { eventVisibility, listAgentSessions } from "./session.api";

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
});
