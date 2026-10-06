import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { ChatTranscript } from "./ChatTranscript";

describe("ChatTranscript reasoning effort", () => {
  it("shows projected effort on the assistant bubble title", () => {
    render(
      <ChatTranscript
        events={[
          {
            id: "m1",
            runId: "",
            seq: 1,
            ts: 1,
            severity: "info",
            type: "assistant.message",
            payload: { turnId: "t1", text: "ok", source: "llm", reasoningEffort: "max" },
          },
        ]}
      />,
    );
    expect(screen.getByTestId("agent-chat-bubble-assistant")).toHaveTextContent("助手 · Max");
  });
});
