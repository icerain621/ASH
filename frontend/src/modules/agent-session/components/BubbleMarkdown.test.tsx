import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { BubbleMarkdown } from "./BubbleMarkdown";

describe("BubbleMarkdown", () => {
  it("renders bold, code, and fences", () => {
    render(
      <BubbleMarkdown text={"hello **world** and `x`\n\n```js\nconst a = 1\n```"} />,
    );
    expect(screen.getByTestId("agent-chat-markdown").querySelector("strong")?.textContent).toBe(
      "world",
    );
    expect(screen.getByTestId("agent-chat-markdown").querySelector(".ash-md-code")?.textContent).toBe(
      "x",
    );
    expect(screen.getByTestId("agent-chat-markdown").querySelector("pre code")?.textContent).toContain(
      "const a = 1",
    );
  });

  it("escapes raw HTML", () => {
    render(<BubbleMarkdown text={"<script>alert(1)</script>"} />);
    expect(screen.getByTestId("agent-chat-markdown").innerHTML).not.toContain("<script>");
    expect(screen.getByTestId("agent-chat-markdown").textContent).toContain("<script>");
  });
});
