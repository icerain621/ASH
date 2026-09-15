import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ChatSeats } from "./ChatSeats";

const listAgentModels = vi.hoisted(() => vi.fn());
const updateSession = vi.hoisted(() => vi.fn());

vi.mock("../api/session.api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/session.api")>();
  return {
    ...actual,
    listAgentModels: (...args: unknown[]) => listAgentModels(...args),
    updateSession: (...args: unknown[]) => updateSession(...args),
  };
});

function wrap(ui: ReactNode) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(<QueryClientProvider client={qc}>{ui}</QueryClientProvider>);
}

describe("ChatSeats", () => {
  beforeEach(() => {
    listAgentModels.mockReset();
    updateSession.mockReset();
    listAgentModels.mockResolvedValue({
      items: [
        { id: "static", label: "static", providerKind: "static" },
        { id: "execgo", label: "execgo", providerKind: "execgo" },
        { id: "acp_sdk", label: "acp_sdk", providerKind: "acp_sdk" },
      ],
    });
    updateSession.mockResolvedValue({
      id: "sess_1",
      spaceId: "local",
      status: "active",
      providerKind: "execgo",
      permissionMode: "full",
    });
  });

  it("renders seats and PATCHes providerKind / permissionMode", async () => {
    wrap(
      <ChatSeats
        session={{
          id: "sess_1",
          spaceId: "local",
          status: "active",
          providerKind: "static",
          permissionMode: "read-only",
        }}
      />,
    );
    expect(screen.getByTestId("agent-chat-seats")).toBeInTheDocument();
    await waitFor(() => expect(listAgentModels).toHaveBeenCalled());
    await waitFor(() =>
      expect(screen.getByTestId("agent-chat-seat-model")).toHaveValue("static"),
    );

    const model = screen.getByTestId("agent-chat-seat-model") as HTMLSelectElement;
    fireEvent.change(model, { target: { value: "execgo" } });
    await waitFor(() =>
      expect(updateSession).toHaveBeenCalledWith("sess_1", { providerKind: "execgo" }),
    );

    const perm = screen.getByTestId("agent-chat-seat-permission") as HTMLSelectElement;
    fireEvent.change(perm, { target: { value: "full" } });
    await waitFor(() =>
      expect(updateSession).toHaveBeenCalledWith("sess_1", { permissionMode: "full" }),
    );
  });
});
