import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { AgentModeSwitch } from "./AgentModeSwitch";

describe("AgentModeSwitch", () => {
  it("calls onChange with general", () => {
    const onChange = vi.fn();
    render(<AgentModeSwitch value="coding" onChange={onChange} />);
    fireEvent.click(screen.getByTestId("agent-mode-general"));
    expect(onChange).toHaveBeenCalledWith("general");
  });

  it("exposes mode taglines as title", () => {
    render(<AgentModeSwitch value="coding" onChange={() => {}} />);
    expect(screen.getByTestId("agent-mode-coding").getAttribute("title")).toMatch(/交付编排/);
    expect(screen.getByTestId("agent-mode-general").getAttribute("title")).toMatch(/通用助手/);
  });
});
