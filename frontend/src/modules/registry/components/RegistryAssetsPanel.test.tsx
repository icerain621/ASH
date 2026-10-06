import { describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { RegistryAssetsPanel, validatePolicyBodyJson } from "./RegistryAssetsPanel";

vi.mock("@/modules/registry/api/registry.api", () => ({
  listAgentAssets: vi.fn().mockResolvedValue({ items: [] }),
  listMemoryAssets: vi.fn().mockResolvedValue({ items: [] }),
  createAgentAsset: vi.fn(),
  createMemoryAsset: vi.fn(),
  patchAgentAssetStatus: vi.fn(),
  patchMemoryAssetStatus: vi.fn(),
  getSpacePolicy: vi.fn().mockResolvedValue({ pack: null, effective: null }),
  getSpaceQuotas: vi.fn().mockResolvedValue(null),
  putSpacePolicy: vi.fn().mockResolvedValue({ ok: true }),
}));

describe("validatePolicyBodyJson", () => {
  it("requires non-empty object JSON", () => {
    expect(validatePolicyBodyJson("")).toBe("需要填写 BodyJSON");
    expect(validatePolicyBodyJson("   ")).toBe("需要填写 BodyJSON");
    expect(validatePolicyBodyJson("{")).toBe("BodyJSON 必须是合法 JSON 对象");
    expect(validatePolicyBodyJson("[]")).toBe("BodyJSON 必须是合法 JSON 对象");
    expect(validatePolicyBodyJson("null")).toBe("BodyJSON 必须是合法 JSON 对象");
    expect(validatePolicyBodyJson('"x"')).toBe("BodyJSON 必须是合法 JSON 对象");
    expect(validatePolicyBodyJson("{}")).toBe("BodyJSON 对象不能为空");
    expect(validatePolicyBodyJson('{"toolApprovalPresets":[]}')).toBeNull();
  });
});

describe("RegistryAssetsPanel confirms", () => {
  it("requires confirm before writing policy pack", async () => {
    const { putSpacePolicy } = await import("@/modules/registry/api/registry.api");
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={qc}>
        <RegistryAssetsPanel spaceId="local" />
      </QueryClientProvider>,
    );
    const btn = await screen.findByTestId("registry-put-policy");
    expect(btn).toHaveAttribute("title", "写入 citation/multiSign/SLA 策略包（需确认）");
    fireEvent.click(btn);
    expect(confirmSpy).toHaveBeenCalled();
    expect(putSpacePolicy).not.toHaveBeenCalled();
    confirmSpy.mockRestore();
  });

  it("requires confirm before creating agent or memory assets", async () => {
    const { createAgentAsset, createMemoryAsset } = await import("@/modules/registry/api/registry.api");
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={qc}>
        <RegistryAssetsPanel spaceId="local" />
      </QueryClientProvider>,
    );
    const agent = await screen.findByTestId("registry-create-agent");
    expect(agent).toHaveAttribute("title", "登记 Agent 资产（需确认）");
    fireEvent.click(agent);
    expect(confirmSpy).toHaveBeenCalled();
    expect(createAgentAsset).not.toHaveBeenCalled();

    const memory = screen.getByTestId("registry-create-memory");
    expect(memory).toHaveAttribute("title", "登记 Memory 资产（需确认）");
    fireEvent.click(memory);
    expect(createMemoryAsset).not.toHaveBeenCalled();
    confirmSpy.mockRestore();
  });

  it("requires confirm before toggling agent asset status", async () => {
    const { listAgentAssets, patchAgentAssetStatus } = await import("@/modules/registry/api/registry.api");
    vi.mocked(listAgentAssets).mockResolvedValueOnce({
      items: [{ id: "ag_1", name: "bot", kind: "agent", status: "active" }],
    } as never);
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={qc}>
        <RegistryAssetsPanel spaceId="local" />
      </QueryClientProvider>,
    );
    const toggle = await screen.findByTestId("toggle-ag_1");
    expect(toggle).toHaveAttribute("title", "停用该资产（需确认）");
    fireEvent.click(toggle);
    expect(confirmSpy).toHaveBeenCalled();
    expect(patchAgentAssetStatus).not.toHaveBeenCalled();
    confirmSpy.mockRestore();
    vi.mocked(listAgentAssets).mockResolvedValue({ items: [] });
  });
});
