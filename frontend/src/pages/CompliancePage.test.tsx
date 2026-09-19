import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { CompliancePage } from "./CompliancePage";
import { getSpaceAuditReport } from "@/modules/compliance/api/compliance.api";
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
});
