import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { ChatTranscript } from "./ChatTranscript";

describe("ChatTranscript brand hero", () => {
  it("shows hero and dismiss", () => {
    const onDismiss = vi.fn();
    render(
      <ChatTranscript
        events={[]}
        agentMode="general"
        showBrandHero
        onDismissBrandHero={onDismiss}
      />,
    );
    expect(screen.getByTestId("agent-ash-hero")).toBeInTheDocument();
    expect(screen.getByTestId("agent-ash-hero-tag")).toHaveTextContent(/通用助手/);
    fireEvent.click(screen.getByTestId("agent-ash-hero-dismiss"));
    expect(onDismiss).toHaveBeenCalled();
  });
});
