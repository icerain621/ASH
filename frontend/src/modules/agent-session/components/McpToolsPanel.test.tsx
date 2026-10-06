import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { McpToolsPanel } from "./McpToolsPanel";

const listMCPTools = vi.fn();
const registerMCPTool = vi.fn();
const executeMCPTool = vi.fn();
const patchMCPTool = vi.fn();

vi.mock("@/modules/platform/api/platform.api", () => ({
  listMCPTools: (...a: unknown[]) => listMCPTools(...a),
  registerMCPTool: (...a: unknown[]) => registerMCPTool(...a),
  executeMCPTool: (...a: unknown[]) => executeMCPTool(...a),
  patchMCPTool: (...a: unknown[]) => patchMCPTool(...a),
}));

function wrap(ui: React.ReactNode) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(<QueryClientProvider client={qc}>{ui}</QueryClientProvider>);
}

describe("McpToolsPanel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    listMCPTools.mockResolvedValue({
      items: [
        {
          id: "mcp_1",
          name: "echo",
          server: "local",
          risk: "low",
          status: "disabled",
        },
      ],
    });
  });

  it("disables register until name and server are filled", async () => {
    wrap(<McpToolsPanel open onClose={vi.fn()} />);
    await waitFor(() => expect(screen.getByTestId("agent-mcp-register-btn")).toBeInTheDocument());
    const btn = screen.getByTestId("agent-mcp-register-btn");
    expect(btn).toBeDisabled();
    expect(btn).toHaveAttribute("title", "需要填写 name");

    fireEvent.change(screen.getByTestId("agent-mcp-name"), { target: { value: "echo" } });
    expect(btn).toBeDisabled();
    expect(btn).toHaveAttribute("title", "需要填写 server");

    fireEvent.change(screen.getByTestId("agent-mcp-server"), { target: { value: "local" } });
    expect(btn).toBeEnabled();
    expect(btn).toHaveAttribute("title", "登记 MCP 工具（需确认）");
    expect(screen.getByTestId("agent-mcp-close")).toHaveAttribute("title", "关闭 MCP 面板");
  });

  it("requires confirm before registering MCP tool", async () => {
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    wrap(<McpToolsPanel open onClose={vi.fn()} />);
    await waitFor(() => expect(screen.getByTestId("agent-mcp-register-btn")).toBeInTheDocument());
    fireEvent.change(screen.getByTestId("agent-mcp-name"), { target: { value: "echo" } });
    fireEvent.change(screen.getByTestId("agent-mcp-server"), { target: { value: "local" } });
    fireEvent.click(screen.getByTestId("agent-mcp-register-btn"));
    expect(confirmSpy).toHaveBeenCalled();
    expect(registerMCPTool).not.toHaveBeenCalled();
    confirmSpy.mockRestore();
  });

  it("requires confirm before toggling MCP tool status", async () => {
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    wrap(<McpToolsPanel open onClose={vi.fn()} />);
    const toggle = await screen.findByTestId("agent-mcp-toggle-mcp_1");
    expect(toggle).toHaveAttribute("title", "启用此工具（需确认）");
    fireEvent.click(toggle);
    expect(confirmSpy).toHaveBeenCalled();
    expect(patchMCPTool).not.toHaveBeenCalled();
    confirmSpy.mockRestore();
  });

  it("explains why try-exec stays disabled for stopped tools", async () => {
    wrap(<McpToolsPanel open onClose={vi.fn()} />);
    const tryBtn = await screen.findByTestId("agent-mcp-try-mcp_1");
    expect(tryBtn).toBeDisabled();
    expect(tryBtn).toHaveAttribute("title", "工具已停用，先启用再试执行");
  });

  it("requires confirm before try-exec on enabled tools", async () => {
    listMCPTools.mockResolvedValue({
      items: [
        {
          id: "mcp_2",
          name: "echo",
          server: "local",
          risk: "low",
          status: "registered",
        },
      ],
    });
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    wrap(<McpToolsPanel open onClose={vi.fn()} />);
    const tryBtn = await screen.findByTestId("agent-mcp-try-mcp_2");
    expect(tryBtn).toHaveAttribute("title", "试执行（需确认）");
    fireEvent.click(tryBtn);
    expect(confirmSpy).toHaveBeenCalledWith(expect.stringContaining("echo"));
    expect(executeMCPTool).not.toHaveBeenCalled();
    confirmSpy.mockRestore();
  });
});
