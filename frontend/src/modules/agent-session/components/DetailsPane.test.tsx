import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { DetailsPane } from "./DetailsPane";

describe("DetailsPane", () => {
  it("shows reasoning effort from assistant.message payload", () => {
    render(
      <DetailsPane
        open
        selection={{
          id: "m1",
          type: "assistant.message",
          kind: "assistant",
          title: "助手",
          summary: "ok",
          payload: { text: "ok", source: "llm", reasoningEffort: "max" },
        }}
      />,
    );
    expect(screen.getByTestId("agent-chat-details-effort")).toHaveTextContent(/Max/);
  });

  it("shows compact summary for replay", () => {
    render(
      <DetailsPane
        open
        selection={{
          id: "c1",
          type: "harness.compaction",
          kind: "compact",
          title: "Compact",
          summary: "short",
          payload: { summary: "compacted 2 turns / 1 replies", priorTurns: 2, priorReplies: 1 },
        }}
      />,
    );
    expect(screen.getByTestId("agent-details-compact-summary")).toHaveTextContent(
      "compacted 2 turns / 1 replies",
    );
  });
});
