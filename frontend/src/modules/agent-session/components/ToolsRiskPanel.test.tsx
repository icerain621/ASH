import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ToolsRiskPanel } from "./ToolsRiskPanel";

const listToolRiskCatalog = vi.fn();
const patchAgentSession = vi.fn();

vi.mock("@/modules/platform/api/platform.api", () => ({
  listToolRiskCatalog: (...a: unknown[]) => listToolRiskCatalog(...a),
}));

vi.mock("@/modules/agent-session/api/session.api", () => ({
  patchAgentSession: (...a: unknown[]) => patchAgentSession(...a),
}));

function wrap(ui: React.ReactNode) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(<QueryClientProvider client={qc}>{ui}</QueryClientProvider>);
}

describe("ToolsRiskPanel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    listToolRiskCatalog.mockResolvedValue({
      items: [{ name: "shell.exec", label: "Shell", risk: "danger", defaultDeny: true }],
    });
  });

  it("disables tool toggle until a session is selected", async () => {
    wrap(<ToolsRiskPanel open onClose={vi.fn()} session={null} />);
    const toggle = await screen.findByTestId("agent-tools-toggle-shell.exec");
    expect(toggle).toBeDisabled();
    expect(toggle).toHaveAttribute("title", "需要先选择会话");
  });

  it("explains enable/disable when session is present", async () => {
    wrap(
      <ToolsRiskPanel
        open
        onClose={vi.fn()}
        session={{ id: "sess_1", status: "active", disabledTools: ["shell.exec"] } as never}
      />,
    );
    const toggle = await screen.findByTestId("agent-tools-toggle-shell.exec");
    await waitFor(() => expect(toggle).toBeEnabled());
    expect(toggle).toHaveAttribute("title", "启用 shell.exec（需确认）");
  });

  it("requires confirm before toggling tool enable/disable", async () => {
    patchAgentSession.mockResolvedValue({ ok: true });
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    wrap(
      <ToolsRiskPanel
        open
        onClose={vi.fn()}
        session={{ id: "sess_1", status: "active", disabledTools: ["shell.exec"] } as never}
      />,
    );
    const toggle = await screen.findByTestId("agent-tools-toggle-shell.exec");
    await waitFor(() => expect(toggle).toBeEnabled());
    fireEvent.click(toggle);
    expect(confirmSpy).toHaveBeenCalled();
    expect(patchAgentSession).not.toHaveBeenCalled();

    confirmSpy.mockReturnValue(true);
    fireEvent.click(toggle);
    await waitFor(() => {
      expect(patchAgentSession).toHaveBeenCalled();
    });
    confirmSpy.mockRestore();
  });
});
