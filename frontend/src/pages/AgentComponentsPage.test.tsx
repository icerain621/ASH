import { screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";
import { AgentComponentsPage } from "./AgentComponentsPage";
import { renderPage } from "@/test/renderPage";

vi.mock("@tanstack/react-router", () => ({
  Link: ({ children, ...props }: { children: ReactNode; to?: string; className?: string }) => (
    <a href={props.to ?? "#"} className={props.className}>
      {children}
    </a>
  ),
}));

vi.mock("@/modules/agent-catalog/api/agentCatalog.api", () => ({
  listAgentComponents: vi.fn(async () => ({
    items: [
      { id: "mem.retrieve", kind: "memory", status: "approved" },
      { id: "tool.custom.x", kind: "tool", status: "candidate" },
    ],
  })),
  registerAgentComponent: vi.fn(),
}));

describe("AgentComponentsPage", () => {
  it("lists component status separately from templates", async () => {
    renderPage(<AgentComponentsPage />);
    expect(screen.getByTestId("agent-components-page")).toBeInTheDocument();
    expect(await screen.findByTestId("component-status-tool.custom.x")).toHaveTextContent("candidate");
    expect(screen.getByTestId("component-status-mem.retrieve")).toHaveTextContent("approved");
  });
});
