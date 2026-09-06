import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { AutomationPage } from "./AutomationPage";
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
      expect(screen.getByTestId("skills-pack-verify-btn")).toBeInTheDocument();
      expect(screen.getByText("已安装")).toBeInTheDocument();
    });
  });
});
