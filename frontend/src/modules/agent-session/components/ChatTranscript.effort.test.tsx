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
            payload: { turnId: "t1", text: "ok", source: "llm", llmModel: "gpt-4o-mini", reasoningEffort: "max" },
          },
        ]}
      />,
    );
    const bubble = screen.getByTestId("agent-chat-bubble-assistant");
    expect(bubble).toHaveTextContent("助手 · gpt-4o-mini · Max");
    expect(bubble).not.toHaveAttribute("aria-busy");
    expect(screen.getByTestId("agent-chat-transcript")).toHaveAttribute("aria-label", "对话");
    expect(screen.getByLabelText("消息列表")).toBeInTheDocument();
  });

  it("marks the selected assistant bubble with aria-current", () => {
    render(
      <ChatTranscript
        selectedId="assistant:t1"
        events={[
          {
            id: "m1",
            runId: "",
            seq: 1,
            ts: 1,
            severity: "info",
            type: "assistant.message",
            payload: { turnId: "t1", text: "ok", source: "llm", llmModel: "gpt-4o-mini", reasoningEffort: "high" },
          },
        ]}
      />,
    );
    expect(screen.getByTestId("agent-chat-bubble-assistant")).toHaveAttribute(
      "aria-current",
      "true",
    );
  });

  it("marks streaming assistant bubbles aria-busy and stopped badge as status", () => {
    render(
      <ChatTranscript
        events={[
          {
            id: "d1",
            runId: "",
            seq: 1,
            ts: 1,
            severity: "info",
            type: "assistant.delta",
            payload: {
              turnId: "t1",
              text: "partial",
              source: "llm",
              llmModel: "gpt-4o-mini",
              providerKind: "static",
              reasoningEffort: "high",
            },
          },
        ]}
      />,
    );
    const bubble = screen.getByTestId("agent-chat-bubble-assistant");
    expect(bubble).toHaveAttribute("data-streaming", "1");
    expect(bubble).toHaveAttribute("aria-busy", "true");
    expect(bubble).toHaveTextContent(/助手 · static · gpt-4o-mini · High/);
  });
});
