import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it } from "vitest";
import { TrajectoryPane } from "./TrajectoryPane";

function wrap(node: ReactNode) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={client}>{node}</QueryClientProvider>;
}

describe("TrajectoryPane", () => {
  it("projects reasoningEffort onto the assistant.message row title", () => {
    render(
      wrap(
        <TrajectoryPane
          events={[
            {
              id: "m1",
              runId: "",
              seq: 2,
              ts: 1,
              severity: "info",
              type: "assistant.message",
              payload: { text: "ok", source: "llm", reasoningEffort: "max" },
            },
          ]}
        />,
      ),
    );
    expect(screen.getByTestId("agent-trajectory-row-2")).toHaveTextContent("助手 · Max");
  });
});
