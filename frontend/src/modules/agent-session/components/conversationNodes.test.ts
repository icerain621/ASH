import { describe, expect, it } from "vitest";
import { eventVisibility, type SessionEventEnvelope } from "../api/session.api";
import { isThreadVisibleEvent, resolveConversationNode } from "./conversationNodes";

function ev(partial: Partial<SessionEventEnvelope> & { type: string }): SessionEventEnvelope {
  return {
    id: partial.id ?? "evt_1",
    runId: partial.runId ?? "run_1",
    seq: partial.seq ?? 1,
    ts: partial.ts ?? 1,
    severity: partial.severity ?? "info",
    type: partial.type,
    visibility: partial.visibility,
    payload: partial.payload,
  };
}

describe("conversationNodes", () => {
  it("hides audit events from the thread", () => {
    expect(isThreadVisibleEvent(ev({ type: "metric.kpi", visibility: "audit" }))).toBe(false);
    expect(isThreadVisibleEvent(ev({ type: "session.turn" }))).toBe(true);
    expect(isThreadVisibleEvent(ev({ type: "gate.waiting_approval" }))).toBe(true);
  });

  it("resolves known event types to node kinds", () => {
    expect(resolveConversationNode(ev({ type: "session.turn" })).kind).toBe("session.turn");
    expect(resolveConversationNode(ev({ type: "gate.waiting_approval" })).kind).toBe("gate.waiting_approval");
    expect(resolveConversationNode(ev({ type: "run.started" })).kind).toBe("default");
  });

  it("uses eventVisibility when visibility field is missing", () => {
    const item = ev({ type: "gate.waiting_approval" });
    expect(eventVisibility(item)).toBe("ui_only");
    expect(isThreadVisibleEvent(item)).toBe(true);
  });
});
