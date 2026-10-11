import { screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";
import { AgentTemplatesPage } from "./AgentTemplatesPage";
import { renderPage } from "@/test/renderPage";

vi.mock("@tanstack/react-router", () => ({
  Link: ({ children, ...props }: { children: ReactNode; to?: string; className?: string }) => (
    <a href={props.to ?? "#"} className={props.className}>
      {children}
    </a>
  ),
}));

vi.mock("@/modules/agent-catalog/api/agentCatalog.api", async () => {
  const actual = (await vi.importActual(
    "@/modules/agent-catalog/api/agentCatalog.api",
  )) as Record<string, unknown>;
  return {
    ...actual,
    listAgentTemplates: vi.fn(async () => ({
      items: [
        { status: "approved", manifest: { id: "tpl.react", version: "1.0.0", maxTurns: 8 } },
        { status: "candidate", manifest: { id: "tpl.custom.demo", version: "1.0.0", maxTurns: 2 } },
      ],
    })),
    submitAgentTemplateCandidate: vi.fn(),
  };
});

describe("AgentTemplatesPage", () => {
  it("shows bindable flag for approved vs candidate", async () => {
    renderPage(<AgentTemplatesPage />);
    expect(screen.getByTestId("agent-templates-page")).toBeInTheDocument();
    expect(await screen.findByTestId("template-bindable-tpl.react")).toHaveTextContent("是");
    expect(screen.getByTestId("template-bindable-tpl.custom.demo")).toHaveTextContent("否");
    expect(screen.getByTestId("template-status-tpl.custom.demo")).toHaveTextContent("candidate");
  });
});
