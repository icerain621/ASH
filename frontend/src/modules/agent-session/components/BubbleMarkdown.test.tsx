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

  it("renders GFM tables", () => {
    render(
      <BubbleMarkdown
        text={"| Name | Val |\n| --- | --- |\n| a | **1** |\n| b | 2 |"}
      />,
    );
    const table = screen.getByTestId("agent-chat-md-table");
    expect(table.querySelectorAll("th")).toHaveLength(2);
    expect(table.querySelectorAll("tbody tr")).toHaveLength(2);
    expect(table.querySelector("strong")?.textContent).toBe("1");
  });

  it("renders task lists", () => {
    render(<BubbleMarkdown text={"- [ ] open\n- [x] done"} />);
    const list = screen.getByTestId("agent-chat-md-tasks");
    const items = list.querySelectorAll("li");
    expect(items).toHaveLength(2);
    expect(items[0]).toHaveAttribute("data-checked", "0");
    expect(items[1]).toHaveAttribute("data-checked", "1");
    expect(items[1].textContent).toContain("done");
  });
});
