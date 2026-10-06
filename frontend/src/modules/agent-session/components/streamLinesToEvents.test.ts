import { describe, expect, it } from "vitest";
import { streamLinesToEvents } from "./AgentChatShell";
import { mergeAssistantBubbles } from "./conversationNodes";

describe("streamLinesToEvents", () => {
  it("unwraps SSE envelopes so bubbles show the prompt and one assistant reply", () => {
    const events = streamLinesToEvents(
      [
        {
          id: "turn_1",
          type: "session.turn",
          payload: JSON.stringify({
            id: "turn_1",
            seq: 1,
            type: "session.turn",
            payload: { prompt: "你好", turnId: "turn_1" },
          }),
        },
        {
          id: "d0",
          type: "assistant.delta",
          payload: JSON.stringify({
            id: "d0",
            seq: 2,
            type: "assistant.delta",
            payload: { index: 0, text: "已收到", turnId: "turn_1" },
          }),
        },
        {
          id: "d1",
          type: "assistant.delta",
          payload: JSON.stringify({
            id: "d1",
            seq: 3,
            type: "assistant.delta",
            payload: { index: 1, text: "：你好", turnId: "turn_1" },
          }),
        },
        {
          id: "a1",
          type: "assistant.message",
          payload: JSON.stringify({
            id: "a1",
            seq: 4,
            type: "assistant.message",
            payload: { text: "已收到：你好", turnId: "turn_1", source: "echo", stopped: false },
          }),
        },
      ],
      "sess_1",
    );
    const bubbles = mergeAssistantBubbles(events);
    expect(bubbles.map((b) => b.summary)).toEqual(["你好", "已收到：你好"]);
  });

  it("keeps a bare payload that is not an event envelope", () => {
    const events = streamLinesToEvents(
      [{ id: "t", type: "session.turn", payload: JSON.stringify({ prompt: "hi", turnId: "t" }) }],
      "sess_1",
    );
    expect(mergeAssistantBubbles(events).map((b) => b.summary)).toEqual(["hi"]);
  });
});
