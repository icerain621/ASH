import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { CompliancePage } from "./CompliancePage";
import { getSpaceAuditReport, scanSecrets, type AuditReport } from "@/modules/compliance/api/compliance.api";
import { renderPage } from "@/test/renderPage";

vi.mock("@tanstack/react-router", () => ({
  Link: ({ children, ...props }: { children: React.ReactNode; to?: string; className?: string }) => (
    <a href={props.to || "#"} className={props.className}>
      {children}
    </a>
  ),
}));

vi.mock("@/modules/compliance/api/compliance.api", () => ({
  scanSecrets: vi.fn().mockResolvedValue({ findings: [], leakCount: 0 }),
  exportComplianceBundle: vi.fn(),
  getSpaceAuditReport: vi.fn().mockResolvedValue({
    spaceId: "local",
    window: "7d",
    total: 0,
    buckets: { approve: 0, deny: 0, hook: 0, spawn: 0, other: 0 },
  }),
}));

vi.mock("@/modules/doctor/api/doctor.api", () => ({
  runDoctor: vi.fn(),
  getDoctorReport: vi.fn(),
}));

vi.mock("@/modules/runs/api/runs.api", () => ({
  listRuns: vi.fn().mockResolvedValue({ items: [] }),
}));

vi.mock("@/modules/platform/api/platform.api", () => ({
  getAuditPolicy: vi.fn().mockResolvedValue({ retentionDays: 365, redactPayload: false }),
  getAuthMe: vi.fn().mockResolvedValue({
    userId: "u1",
    spaceId: "local",
    role: "admin",
    permissions: ["read"],
    user: { id: "u1", displayName: "Dev" },
  }),
  getPermissionMatrix: vi.fn().mockResolvedValue({ roles: [], actions: [] }),
  updateAuditPolicy: vi.fn(),
  getPluginABIProfile: vi.fn().mockResolvedValue({
    version: "v0",
    currentAbi: "v0",
    supportedProtocols: ["grpc"],
    protoFiles: [],
  }),
  getStorageProfile: vi.fn().mockResolvedValue({
    backend: "local",
    database: { dialect: "sqlite" },
    artifactStore: { ready: true, kind: "fs" },
  }),
  listPlugins: vi.fn().mockResolvedValue({ items: [] }),
  listSpaceMembers: vi.fn().mockResolvedValue({ items: [] }),
  listSpaceResourceScopes: vi.fn().mockResolvedValue({ items: [] }),
}));

describe("CompliancePage", () => {
  it("renders compliance heading and TR2 run control", async () => {
    renderPage(<CompliancePage />);
    expect(screen.getByRole("heading", { name: "合规控制台" })).toBeInTheDocument();
    await waitFor(() => {
      expect(screen.getByRole("button", { name: "运行 TR2" })).toBeInTheDocument();
      expect(screen.getByText("M3-01")).toBeInTheDocument();
    });
    expect(screen.getByTestId("audit-report-empty")).toHaveTextContent("无审计事件");
  });

  it("switches audit report window and shows bucket counts", async () => {
    vi.mocked(getSpaceAuditReport).mockImplementation(async (_space, window) => {
      if (window === "24h") {
        return {
          spaceId: "local",
          window: "24h",
          total: 3,
          buckets: { approve: 1, deny: 1, hook: 1, spawn: 0, other: 0 },
        };
      }
      return {
        spaceId: "local",
        window: "7d",
        total: 0,
        buckets: { approve: 0, deny: 0, hook: 0, spawn: 0, other: 0 },
      };
    });
    renderPage(<CompliancePage />);
    expect(await screen.findByTestId("audit-report-empty")).toBeInTheDocument();
    fireEvent.change(screen.getByTestId("audit-report-window"), { target: { value: "24h" } });
    expect(await screen.findByTestId("audit-report-counts")).toHaveTextContent("合计 3");
    expect(getSpaceAuditReport).toHaveBeenCalledWith("local", "24h");
  });

  it("shows other bucket when most events are uncategorized", async () => {
    vi.mocked(getSpaceAuditReport).mockResolvedValue({
      spaceId: "local",
      window: "7d",
      total: 10,
      buckets: { approve: 0, deny: 0, hook: 0, spawn: 0, other: 10 },
      byEvent: [
        { eventType: "agent.session", count: 7 },
        { eventType: "review.decided", count: 3 },
      ],
    });
    renderPage(<CompliancePage />);
    const counts = await screen.findByTestId("audit-report-counts");
    expect(counts).toHaveTextContent("合计 10");
    expect(counts).toHaveTextContent("其他 10");
    const byEvent = await screen.findByTestId("audit-report-by-event");
    expect(byEvent).toHaveTextContent("agent.session");
    expect(byEvent).toHaveTextContent("7");
  });

  it("tolerates missing buckets without crashing", async () => {
    vi.mocked(getSpaceAuditReport).mockResolvedValue({
      spaceId: "local",
      window: "7d",
      total: 2,
      buckets: null as unknown as AuditReport["buckets"],
    });
    renderPage(<CompliancePage />);
    expect(await screen.findByTestId("audit-report-counts")).toHaveTextContent("合计 2");
    expect(screen.getByTestId("audit-report-counts")).toHaveTextContent("其他 0");
  });

  it("renders the empty scan row when findings is null", async () => {
    vi.mocked(scanSecrets).mockResolvedValueOnce({
      spaceId: "local",
      scanned: 74,
      leakCount: 0,
      redactEnabled: false,
      findings: null as unknown as [],
    });
    renderPage(<CompliancePage />);
    expect(await screen.findByText(/未发现明文 secret/)).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "合规控制台" })).toBeInTheDocument();
  });

  it("tolerates null storage nested fields and null matrix slices", async () => {
    const platform = await import("@/modules/platform/api/platform.api");
    vi.mocked(platform.getStorageProfile).mockResolvedValueOnce({
      backend: "local",
      database: null as unknown as { dialect: string },
      artifactStore: null as unknown as { ready: boolean; kind: string },
    } as never);
    vi.mocked(platform.getPermissionMatrix).mockResolvedValueOnce({
      builtinRoles: null,
      scenarioTools: null,
    } as never);
    vi.mocked(platform.listSpaceResourceScopes).mockResolvedValueOnce({ items: null } as never);
    renderPage(<CompliancePage />);
    await waitFor(() => {
      expect(screen.getByRole("heading", { name: "合规控制台" })).toBeInTheDocument();
      expect(screen.getByRole("heading", { name: "存储与插件" })).toBeInTheDocument();
    });
  });

  it("requires confirm before enabling redact", async () => {
    const platform = await import("@/modules/platform/api/platform.api");
    vi.mocked(platform.updateAuditPolicy).mockResolvedValue({ retentionDays: 365, redactPayload: true } as never);
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    renderPage(<CompliancePage />);
    const btn = await screen.findByTestId("compliance-redact-enable");
    expect(btn).toHaveAttribute("title", "开启后审计列表 API 与导出包会对载荷脱敏（需确认）");
    fireEvent.click(btn);
    expect(confirmSpy).toHaveBeenCalled();
    expect(platform.updateAuditPolicy).not.toHaveBeenCalled();

    confirmSpy.mockReturnValue(true);
    fireEvent.click(btn);
    await waitFor(() => {
      expect(platform.updateAuditPolicy).toHaveBeenCalled();
    });
    confirmSpy.mockRestore();
  });

  it("requires confirm before exporting audit package", async () => {
    const { exportComplianceBundle } = await import("@/modules/compliance/api/compliance.api");
    vi.mocked(exportComplianceBundle).mockResolvedValue({ exportId: "exp_x" } as never);
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    renderPage(<CompliancePage />);
    const btn = await screen.findByTestId("compliance-export-package");
    expect(btn).toHaveAttribute("title", "导出含诊断报告的审计包（需确认）");
    fireEvent.click(btn);
    expect(confirmSpy).toHaveBeenCalled();
    expect(exportComplianceBundle).not.toHaveBeenCalled();
    confirmSpy.mockRestore();
  });

  it("exposes secret scan control with result-aware title", async () => {
    vi.mocked(scanSecrets).mockResolvedValue({
      findings: [],
      leakCount: 0,
      scanned: 12,
    } as never);
    renderPage(<CompliancePage />);
    const scan = await screen.findByTestId("compliance-secret-scan");
    await waitFor(() => {
      expect(scan).toHaveAttribute("title", "重新扫描审计载荷（上次 0 处 / 12 条）");
    });
    expect(screen.getByTestId("compliance-doctor-run")).toHaveAttribute(
      "title",
      "运行 TR2 合规诊断套件",
    );
  });
});
