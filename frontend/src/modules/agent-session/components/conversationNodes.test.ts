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

  it("resolves tool and step events", () => {
    const called = resolveConversationNode(
      ev({ type: "tool.called", payload: { name: "git.status", args: "{}" } }),
    );
    expect(called.kind).toBe("tool.called");
    expect(called.title).toContain("git.status");
    const result = resolveConversationNode(
      ev({ type: "tool.result", payload: { name: "git.status", output: "clean" } }),
    );
    expect(result.kind).toBe("tool.result");
    expect(result.summary).toContain("clean");
    expect(resolveConversationNode(ev({ type: "tool.progress" })).kind).toBe("tool");
    expect(resolveConversationNode(ev({ type: "step.started", payload: { name: "plan" } })).kind).toBe(
      "step",
    );
  });

  it("uses eventVisibility when visibility field is missing", () => {
    const item = ev({ type: "gate.waiting_approval" });
    expect(eventVisibility(item)).toBe("ui_only");
    expect(isThreadVisibleEvent(item)).toBe(true);
  });
});
