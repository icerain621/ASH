import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ChatComposer } from "./ChatComposer";

const listAgentModels = vi.hoisted(() => vi.fn());
const listAgentCommands = vi.hoisted(() => vi.fn());
const listModelProviders = vi.hoisted(() => vi.fn());

vi.mock("../api/session.api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/session.api")>();
  return {
    ...actual,
    listAgentModels: (...args: unknown[]) => listAgentModels(...args),
    listAgentCommands: (...args: unknown[]) => listAgentCommands(...args),
  };
});

vi.mock("@/modules/platform/api/platform.api", () => ({
  listModelProviders: (...args: unknown[]) => listModelProviders(...args),
}));

function wrap(ui: ReactNode) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(<QueryClientProvider client={qc}>{ui}</QueryClientProvider>);
}

describe("ChatComposer", () => {
  beforeEach(() => {
    listAgentModels.mockReset();
    listAgentCommands.mockReset();
    listModelProviders.mockReset();
    listAgentModels.mockResolvedValue({
      items: [{ id: "execgo", label: "execgo", providerKind: "execgo" }],
    });
    listAgentCommands.mockResolvedValue({ items: [] });
    listModelProviders.mockResolvedValue({ items: [] });
  });

  it("docks a DSH-like card: draft above a chip toolbar with seats and send", () => {
    wrap(
      <ChatComposer
        mode="prompt"
        session={{
          id: "sess_1",
          spaceId: "local",
          status: "active",
          providerKind: "execgo",
          permissionMode: "read-only",
        }}
        onIntent={vi.fn()}
        onOpenTools={vi.fn()}
        onOpenSkills={vi.fn()}
      />,
    );

    const card = screen.getByTestId("agent-chat-composer");
    const field = screen.getByTestId("agent-intent-field");
    const toolbar = screen.getByTestId("agent-intent-toolbar");
    const prompt = screen.getByTestId("agent-intent-prompt");
    const seats = screen.getByTestId("agent-chat-seats");
    const send = screen.getByTestId("agent-intent-send");

    expect(card).toContainElement(field);
    expect(field).toContainElement(prompt);
    expect(field).toContainElement(toolbar);
    expect(toolbar).toContainElement(seats);
    expect(toolbar).toContainElement(send);
    expect(screen.queryByTestId("agent-chat-permission-strip")).toBeNull();

    const pos = (el: HTMLElement) =>
      prompt.compareDocumentPosition(el) & Node.DOCUMENT_POSITION_FOLLOWING
        ? "after-prompt"
        : "before-prompt";
    expect(pos(toolbar)).toBe("after-prompt");
  });

  it("converges plus-menu extras and shows model plus reasoning effort", async () => {
    wrap(
      <ChatComposer
        mode="prompt"
        session={{
          id: "sess_1",
          spaceId: "local",
          status: "active",
          providerKind: "execgo",
          permissionMode: "read-only",
        }}
        onIntent={vi.fn()}
        onOpenTools={vi.fn()}
        onOpenSkills={vi.fn()}
      />,
    );

    const prompt = screen.getByTestId("agent-intent-prompt");
    expect(prompt).toHaveAttribute("placeholder", "发消息或创建任务，/ 调用指令");

    expect(screen.getByTestId("agent-composer-plus")).toBeInTheDocument();
    expect(screen.queryByTestId("agent-composer-tools")).toBeNull();
    fireEvent.click(screen.getByTestId("agent-composer-plus"));
    expect(screen.getByTestId("agent-composer-plus-menu")).toBeInTheDocument();
    expect(screen.getByTestId("agent-intent-slash")).toBeInTheDocument();
    expect(screen.getByTestId("agent-composer-tools")).toBeInTheDocument();
    expect(screen.getByTestId("agent-composer-skills")).toBeInTheDocument();

    const modelTrigger = await screen.findByTestId("agent-chat-seat-model-trigger");
    expect(modelTrigger).toHaveTextContent(/execgo/);
    expect(modelTrigger).toHaveTextContent(/High/);
    fireEvent.click(modelTrigger);
    expect(screen.getByTestId("agent-chat-seat-model-menu")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("agent-chat-seat-effort-open"));
    fireEvent.click(screen.getByTestId("agent-chat-seat-effort-max"));
    expect(screen.getByTestId("agent-chat-seat-model-trigger")).toHaveTextContent(/Max/);
  });
});
