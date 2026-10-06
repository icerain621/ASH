import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { AutomationPage } from "./AutomationPage";
import { getAuditExportAccess, getPluginABIProfile, listApprovals, listAuditExports, listPlugins, listSecrets, deleteSecret, applyAuditRetention, rotateSecret, createAuditExport, verifyPlugin, approveApproval, rejectApproval } from "@/modules/platform/api/platform.api";
import { renderPage } from "@/test/renderPage";

vi.mock("@/components/ImproveProposalsPane", () => ({
  ImproveProposalsPane: () => <div>Improve proposals</div>,
}));

vi.mock("@/components/HarnessProfilesPane", () => ({
  HarnessProfilesPane: () => <div data-testid="harness-profiles-pane">Harness profiles</div>,
}));

vi.mock("@/modules/skills/api/skills.api", () => ({
  listSkills: vi.fn().mockResolvedValue({
    items: [
      {
        id: "ash-test-discipline",
        name: "ash-test-discipline",
        description: "Prefer smallest safe change",
        path: ".ash/skills/ash-test-discipline/SKILL.md",
        relPath: ".ash/skills/ash-test-discipline/SKILL.md",
        contextRef: "skill:ash-test-discipline",
      },
    ],
    repoRoot: ".",
  }),
  getSkill: vi.fn(),
  installSkillPack: vi.fn(),
  verifySkillPack: vi.fn(),
  listSkillCatalog: vi.fn().mockResolvedValue({
    ok: true,
    source: ".ash/skill-catalog.json",
    items: [
      {
        name: "ash-test-discipline",
        version: "1.0.0",
        publisher: "local",
        url: "packs/ash-test-discipline.zip",
      },
    ],
  }),
  installSkillFromCatalog: vi.fn(),
}));

vi.mock("@/modules/platform/api/platform.api", () => ({
  listModelProviders: vi.fn().mockResolvedValue({ items: [] }),
  listMCPTools: vi.fn().mockResolvedValue({ items: [] }),
  listToolRiskCatalog: vi.fn().mockResolvedValue({
    items: [{ name: "runtime.command", risk: "danger", defaultDeny: true, label: "runtime.command（危险）" }],
  }),
  listPlugins: vi.fn().mockResolvedValue({ items: [] }),
  getPluginABIProfile: vi.fn().mockResolvedValue({
    version: "v0",
    supportedProtocols: ["grpc"],
    breakingPolicy: "reject",
    protoFiles: [],
  }),
  getPluginHealth: vi.fn().mockResolvedValue({ items: [] }),
  getStorageProfile: vi.fn().mockResolvedValue({
    backend: "local",
    database: { dialect: "sqlite", dataDir: ".ash", urlConfigured: false },
    artifactStore: { ready: true, kind: "fs", objectStore: false },
  }),
  listSecrets: vi.fn().mockResolvedValue({ items: [] }),
  listApprovals: vi.fn().mockResolvedValue({ items: [] }),
  listAuditLogs: vi.fn().mockResolvedValue({ items: [] }),
  listAuditExports: vi.fn().mockResolvedValue({ items: [] }),
  getAuditPolicy: vi.fn().mockResolvedValue({ retentionDays: 365, redactPayload: false }),
  getAuditExportAccess: vi.fn(),
  createSecret: vi.fn(),
  deleteSecret: vi.fn(),
  rotateSecret: vi.fn(),
  approveApproval: vi.fn(),
  rejectApproval: vi.fn(),
  createAuditExport: vi.fn(),
  updateAuditPolicy: vi.fn(),
  applyAuditRetention: vi.fn(),
  verifyPlugin: vi.fn(),
}));

describe("AutomationPage", () => {
  it("renders automation heading and space badge", async () => {
    renderPage(<AutomationPage />);
    expect(screen.getByRole("heading", { name: "自动化" })).toBeInTheDocument();
    await waitFor(() => {
      expect(screen.getByText("Space: local")).toBeInTheDocument();
      expect(screen.getByText("Model Router")).toBeInTheDocument();
    });
  });

  it("renders built-in tool risk catalog", async () => {
    renderPage(<AutomationPage />);
    await waitFor(() => {
      expect(screen.getByTestId("tool-risk-catalog")).toBeInTheDocument();
      expect(screen.getByText("runtime.command")).toBeInTheDocument();
      expect(screen.getByText("danger")).toBeInTheDocument();
    });
  });

  it("renders harness profiles pane", async () => {
    renderPage(<AutomationPage />);
    await waitFor(() => {
      expect(screen.getByTestId("harness-profiles-pane")).toBeInTheDocument();
    });
  });

  it("renders skills catalog", async () => {
    renderPage(<AutomationPage />);
    await waitFor(() => {
      expect(screen.getByTestId("skills-catalog")).toBeInTheDocument();
      expect(screen.getAllByText("ash-test-discipline").length).toBeGreaterThan(0);
      expect(screen.getByTestId("skills-org-catalog")).toBeInTheDocument();
      expect(screen.getByTestId("catalog-private-marker")).toHaveTextContent("私有 · 组织 Hub · 不计费");
      expect(screen.getByTestId("skills-pack-verify-btn")).toBeInTheDocument();
      expect(screen.getByText("已安装")).toBeInTheDocument();
    });
  });

  it("explains why skill pack verify is disabled until path and signature are set", async () => {
    renderPage(<AutomationPage />);
    const verify = await screen.findByTestId("skills-pack-verify-btn");
    const install = screen.getByTestId("skills-pack-install-btn");
    expect(verify).toBeDisabled();
    expect(verify).toHaveAttribute("title", "需要填写 packPath");
    fireEvent.change(screen.getByTestId("skills-pack-path"), { target: { value: "   " } });
    fireEvent.change(screen.getByTestId("skills-pack-sig"), { target: { value: "   " } });
    expect(verify).toBeDisabled();
    expect(install).toBeDisabled();
    expect(verify).toHaveAttribute("title", "需要填写 packPath");
    fireEvent.change(screen.getByTestId("skills-pack-path"), { target: { value: "/tmp/x.zip" } });
    expect(verify).toHaveAttribute("title", "需要填写 signature");
    fireEvent.change(screen.getByTestId("skills-pack-sig"), { target: { value: "abc" } });
    expect(verify).not.toBeDisabled();
    expect(verify).toHaveAttribute("title", "验签（干跑）");
    expect(install).toHaveAttribute("title", "安装签名 pack（需确认）");
  });

  it("tolerates null plugin ABI protoFiles without crashing", async () => {
    vi.mocked(getPluginABIProfile).mockResolvedValueOnce({
      currentAbi: "v0",
      supportedAbis: ["v0"],
      supportedProtocols: ["grpc"],
      protoPackage: "ash.v1",
      goPackage: "github.com/ash-repwiki/ash",
      breakingPolicy: "reject",
      protoFiles: null as unknown as [],
    });
    renderPage(<AutomationPage />);
    await waitFor(() => {
      expect(screen.getByText(/0 proto/)).toBeInTheDocument();
    });
  });

  it("tolerates audit export access without digest", async () => {
    vi.mocked(listAuditExports).mockResolvedValueOnce({
      items: [{ id: "exp_1", status: "completed", digest: "" }],
    } as never);
    vi.mocked(getAuditExportAccess).mockResolvedValueOnce({
      exportId: "exp_1",
      digest: undefined,
    } as never);
    renderPage(<AutomationPage />);
    const accessBtn = await screen.findByTestId("audit-export-access-exp_1");
    expect(accessBtn).toHaveAttribute("title", "Get export access");
    fireEvent.click(accessBtn);
    await waitFor(() => {
      expect(getAuditExportAccess).toHaveBeenCalledWith("exp_1");
      expect(screen.getByText(/access exp_1/i)).toBeInTheDocument();
    });
  });

  it("disables secret create until name and value are set", async () => {
    renderPage(<AutomationPage />);
    const create = await screen.findByTestId("secret-create");
    expect(create).toBeDisabled();
    expect(create).toHaveAttribute("title", "需要填写 Name");
    fireEvent.change(screen.getByTestId("secret-name"), { target: { value: "MY_KEY" } });
    expect(create).toHaveAttribute("title", "需要填写 Value");
    fireEvent.change(screen.getByTestId("secret-value"), { target: { value: "s3cret" } });
    expect(create).not.toBeDisabled();
    expect(create).toHaveAttribute("title", "Create secret（需确认）");
  });

  it("requires confirm before creating a secret", async () => {
    const { createSecret } = await import("@/modules/platform/api/platform.api");
    vi.mocked(createSecret).mockClear();
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    renderPage(<AutomationPage />);
    fireEvent.change(await screen.findByTestId("secret-name"), { target: { value: "MY_KEY" } });
    fireEvent.change(screen.getByTestId("secret-value"), { target: { value: "s3cret" } });
    fireEvent.click(screen.getByTestId("secret-create"));
    expect(confirmSpy).toHaveBeenCalled();
    expect(createSecret).not.toHaveBeenCalled();
    confirmSpy.mockRestore();
  });

  it("requires confirm before apply retention and secret delete", async () => {
    vi.mocked(listSecrets).mockResolvedValue({
      items: [{ id: "sec_1", name: "API_KEY", createdAt: "2026-01-01T00:00:00Z" }],
    } as never);
    vi.mocked(applyAuditRetention).mockResolvedValue({ dryRun: false, matched: 2, deleted: 2 } as never);
    vi.mocked(deleteSecret).mockResolvedValue({ ok: true } as never);

    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    renderPage(<AutomationPage />);

    const apply = await screen.findByTestId("audit-retention-apply");
    expect(apply).toHaveAttribute("title", "Apply retention（需确认，会删除过期审计）");
    fireEvent.click(apply);
    expect(confirmSpy).toHaveBeenCalled();
    expect(applyAuditRetention).not.toHaveBeenCalled();

    const del = await screen.findByTestId("secret-delete-sec_1");
    fireEvent.click(del);
    expect(deleteSecret).not.toHaveBeenCalled();

    confirmSpy.mockReturnValue(true);
    fireEvent.click(apply);
    await waitFor(() => {
      expect(applyAuditRetention).toHaveBeenCalledWith({ dryRun: false });
    });
    fireEvent.click(await screen.findByTestId("secret-delete-sec_1"));
    await waitFor(() => {
      expect(deleteSecret).toHaveBeenCalledWith("sec_1");
    });
    confirmSpy.mockRestore();
    vi.mocked(listSecrets).mockResolvedValue({ items: [] });
  });

  it("requires confirm before rotating a secret", async () => {
    vi.mocked(listSecrets).mockResolvedValue({
      items: [{ id: "sec_1", name: "API_KEY", createdAt: "2026-01-01T00:00:00Z" }],
    } as never);
    vi.mocked(rotateSecret).mockResolvedValue({ ok: true } as never);

    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    renderPage(<AutomationPage />);

    const rotate = await screen.findByTestId("secret-rotate-sec_1");
    expect(rotate).toBeDisabled();
    expect(rotate).toHaveAttribute("title", "需要填写新 Value");
    fireEvent.change(screen.getByTestId("secret-rotate-value-sec_1"), { target: { value: "new-secret" } });
    expect(rotate).not.toBeDisabled();
    expect(rotate).toHaveAttribute("title", "Rotate secret（需确认）");

    fireEvent.click(rotate);
    expect(confirmSpy).toHaveBeenCalled();
    expect(rotateSecret).not.toHaveBeenCalled();

    confirmSpy.mockReturnValue(true);
    fireEvent.click(rotate);
    await waitFor(() => {
      expect(rotateSecret).toHaveBeenCalledWith("sec_1", { value: "new-secret" });
    });
    confirmSpy.mockRestore();
    vi.mocked(listSecrets).mockResolvedValue({ items: [] });
  });

  it("requires confirm before verifying a plugin", async () => {
    vi.mocked(listPlugins).mockResolvedValueOnce({
      items: [
        {
          id: "plug_1",
          name: "demo-plugin",
          protocol: "grpc",
          abi: "v1",
          compatible: true,
          status: "verified",
          endpoint: "localhost:9",
          capabilities: "read",
          exportErrors: 0,
          dropCount: 0,
        },
      ],
    } as never);
    vi.mocked(verifyPlugin).mockClear();
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    renderPage(<AutomationPage />);
    const btn = await screen.findByTestId("plugin-verify-plug_1");
    expect(btn).toHaveAttribute("title", "校验插件签名与兼容性（需确认）");
    fireEvent.click(btn);
    expect(confirmSpy).toHaveBeenCalledWith(expect.stringContaining("demo-plugin"));
    expect(verifyPlugin).not.toHaveBeenCalled();

    confirmSpy.mockReturnValue(true);
    fireEvent.click(btn);
    await waitFor(() => {
      expect(verifyPlugin).toHaveBeenCalledWith("plug_1");
    });
    confirmSpy.mockRestore();
    vi.mocked(listPlugins).mockResolvedValue({ items: [] });
  });

  it("requires confirm before creating audit export", async () => {
    vi.mocked(createAuditExport).mockResolvedValue({ id: "exp_1" } as never);
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    renderPage(<AutomationPage />);
    const btn = await screen.findByTestId("audit-export-create");
    expect(btn).toHaveAttribute("title", "Create audit export（需确认）");
    fireEvent.click(btn);
    expect(confirmSpy).toHaveBeenCalled();
    expect(createAuditExport).not.toHaveBeenCalled();
    confirmSpy.mockRestore();
  });

  it("requires confirm before saving audit policy", async () => {
    const { updateAuditPolicy } = await import("@/modules/platform/api/platform.api");
    vi.mocked(updateAuditPolicy).mockClear();
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    renderPage(<AutomationPage />);
    const save = await screen.findByTestId("audit-policy-save");
    expect(save).toHaveAttribute("title", "Save audit policy（需确认）");
    fireEvent.click(save);
    expect(confirmSpy).toHaveBeenCalled();
    expect(updateAuditPolicy).not.toHaveBeenCalled();

    confirmSpy.mockReturnValue(true);
    fireEvent.click(save);
    await waitFor(() => {
      expect(updateAuditPolicy).toHaveBeenCalled();
    });
    confirmSpy.mockRestore();
  });

  it("requires confirm before approving or rejecting approval queue items", async () => {
    vi.mocked(listApprovals).mockResolvedValueOnce({
      items: [
        {
          id: "appr_1",
          runId: "run_1",
          gate: "human",
          reason: "needs review",
          stepId: "s1",
          createdAt: "2026-01-01T00:00:00Z",
        },
      ],
    } as never);
    vi.mocked(approveApproval).mockClear();
    vi.mocked(rejectApproval).mockClear();
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    renderPage(<AutomationPage />);
    const approve = await screen.findByTestId("approval-approve-appr_1");
    const reject = screen.getByTestId("approval-reject-appr_1");
    expect(approve).toHaveAttribute("title", "批准审批（需确认）");
    expect(reject).toHaveAttribute("title", "拒绝审批（需确认）");
    fireEvent.click(approve);
    expect(confirmSpy).toHaveBeenCalled();
    expect(approveApproval).not.toHaveBeenCalled();
    fireEvent.click(reject);
    expect(rejectApproval).not.toHaveBeenCalled();

    confirmSpy.mockReturnValue(true);
    fireEvent.click(approve);
    await waitFor(() => {
      expect(approveApproval).toHaveBeenCalledWith(
        "appr_1",
        expect.objectContaining({ actorId: "console" }),
      );
    });
    confirmSpy.mockRestore();
    vi.mocked(listApprovals).mockResolvedValue({ items: [] });
  });
});
