import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { CompliancePage } from "./CompliancePage";
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
  });
});
