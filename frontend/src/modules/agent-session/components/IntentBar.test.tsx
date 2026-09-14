import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { IntentBar } from "./IntentBar";

describe("IntentBar", () => {
  it("submits prompt intent in agent mode", () => {
    const onIntent = vi.fn();
    render(<IntentBar mode="prompt" busy={false} onIntent={onIntent} />);
    fireEvent.change(screen.getByTestId("agent-intent-prompt"), { target: { value: "continue" } });
    fireEvent.click(screen.getByTestId("agent-intent-send"));
    expect(onIntent).toHaveBeenCalledWith({ action: "prompt", prompt: "continue" });
  });

  it("shows approve/cancel takeover when waiting approval", () => {
    const onIntent = vi.fn();
    render(
      <IntentBar
        mode="gate"
        busy={false}
        gateReason="need human review"
        onIntent={onIntent}
      />,
    );
    expect(screen.getByTestId("agent-intent-bar")).toHaveAttribute("data-mode", "gate");
    fireEvent.click(screen.getByTestId("agent-intent-approve"));
    expect(onIntent).toHaveBeenCalledWith({ action: "approve", reason: "approved from IntentBar" });
    fireEvent.click(screen.getByTestId("agent-intent-cancel"));
    expect(onIntent).toHaveBeenCalledWith({ action: "cancel" });
    fireEvent.click(screen.getByTestId("agent-intent-stop"));
    expect(onIntent).toHaveBeenCalledWith({ action: "stop" });
  });

  it("shows stop in prompt mode when canStop", () => {
    const onIntent = vi.fn();
    render(<IntentBar mode="prompt" canStop busy={false} onIntent={onIntent} />);
    fireEvent.click(screen.getByTestId("agent-intent-stop"));
    expect(onIntent).toHaveBeenCalledWith({ action: "stop" });
  });
});
