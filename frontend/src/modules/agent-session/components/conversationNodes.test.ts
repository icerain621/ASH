import { describe, expect, it } from "vitest";
import { eventVisibility, type SessionEventEnvelope } from "../api/session.api";
import {
  isThreadVisibleEvent,
  mergeAssistantBubbles,
  resolveConversationNode,
} from "./conversationNodes";

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

  it("resolves assistant.delta and assistant.message as 助手", () => {
    const delta = resolveConversationNode(
      ev({ type: "assistant.delta", payload: { turnId: "t1", text: "你好", index: 0 } }),
    );
    expect(delta.kind).toBe("assistant");
    expect(delta.title).toBe("助手");
    expect(delta.summary).toBe("你好");
    const msg = resolveConversationNode(
      ev({ type: "assistant.message", payload: { turnId: "t1", text: "你好世界", source: "echo" } }),
    );
    expect(msg.kind).toBe("assistant");
    expect(msg.title).toBe("助手");
    expect(msg.summary).toBe("你好世界");
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

  it("merges assistant.delta by turnId and completes on assistant.message", () => {
    const bubbles = mergeAssistantBubbles([
      ev({ id: "u1", seq: 1, type: "session.turn", payload: { prompt: "hi", turnId: "turn_1" } }),
      ev({
        id: "d0",
        seq: 2,
        type: "assistant.delta",
        payload: { turnId: "turn_1", text: "已收", index: 0 },
      }),
      ev({
        id: "d1",
        seq: 3,
        type: "assistant.delta",
        payload: { turnId: "turn_1", text: "到：hi", index: 1 },
      }),
      ev({
        id: "m1",
        seq: 4,
        type: "assistant.message",
        payload: { turnId: "turn_1", text: "已收到：hi", source: "echo", stopped: false },
      }),
    ]);
    expect(bubbles).toHaveLength(2);
    expect(bubbles[0].role).toBe("user");
    expect(bubbles[0].summary).toBe("hi");
    expect(bubbles[1].role).toBe("assistant");
    expect(bubbles[1].title).toBe("助手");
    expect(bubbles[1].summary).toBe("已收到：hi");
    expect(bubbles[1].streaming).toBe(false);
    expect(bubbles[1].type).toBe("assistant.message");
    expect(bubbles[1].id).toBe("assistant:turn_1");
  });

  it("keeps a streaming bubble when only deltas arrived", () => {
    const bubbles = mergeAssistantBubbles([
      ev({
        id: "d0",
        seq: 1,
        type: "assistant.delta",
        payload: { turnId: "t2", text: "A", index: 0 },
      }),
      ev({
        id: "d1",
        seq: 2,
        type: "assistant.delta",
        payload: { turnId: "t2", text: "B", index: 1 },
      }),
    ]);
    expect(bubbles).toHaveLength(1);
    expect(bubbles[0].summary).toBe("AB");
    expect(bubbles[0].streaming).toBe(true);
    expect(bubbles[0].role).toBe("assistant");
  });

  it("pairs tool.called and tool.result into one tool card", () => {
    const bubbles = mergeAssistantBubbles([
      ev({
        id: "c1",
        seq: 1,
        type: "tool.called",
        payload: { name: "git.status", callId: "tc_1", args: "{}" },
      }),
      ev({
        id: "r1",
        seq: 2,
        type: "tool.result",
        payload: { name: "git.status", callId: "tc_1", output: "clean" },
      }),
    ]);
    expect(bubbles).toHaveLength(1);
    expect(bubbles[0].kind).toBe("tool.card");
    expect(bubbles[0].toolStatus).toBe("ok");
    expect(bubbles[0].toolName).toBe("git.status");
    expect(bubbles[0].summary).toContain("clean");
    expect(bubbles[0].summary).toContain("←");
  });

  it("keeps running tool card when result has not arrived", () => {
    const bubbles = mergeAssistantBubbles([
      ev({
        id: "c1",
        seq: 1,
        type: "tool.called",
        payload: { name: "shell.exec", args: "ls" },
      }),
    ]);
    expect(bubbles).toHaveLength(1);
    expect(bubbles[0].toolStatus).toBe("running");
    expect(bubbles[0].summary).toContain("执行中");
  });

  it("surfaces stopped badge on assistant.message", () => {
    const bubbles = mergeAssistantBubbles([
      ev({
        id: "m1",
        seq: 1,
        type: "assistant.message",
        payload: { turnId: "t9", text: "partial…", stopped: true },
      }),
    ]);
    expect(bubbles).toHaveLength(1);
    expect(bubbles[0].stopped).toBe(true);
  });
});
