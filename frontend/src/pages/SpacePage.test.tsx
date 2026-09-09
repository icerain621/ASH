import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { getReadyz } from "@/modules/health/api/health.api";
import { SpacePage } from "./SpacePage";
import { renderPage } from "@/test/renderPage";

vi.mock("@tanstack/react-router", () => ({
  Link: ({ children, ...props }: { children: React.ReactNode; to?: string; className?: string }) => (
    <a href={props.to || "#"} className={props.className}>
      {children}
    </a>
  ),
}));

vi.mock("@/modules/health/api/health.api", () => ({
  getReadyz: vi.fn().mockResolvedValue({ status: "ready", consoleAuthRequired: false }),
}));

vi.mock("@/modules/platform/api/platform.api", () => ({
  listOrgs: vi.fn().mockResolvedValue({ items: [] }),
  listSpaces: vi.fn().mockResolvedValue({ items: [] }),
  listRoles: vi.fn().mockResolvedValue({ items: [] }),
  listSpaceMembers: vi.fn().mockResolvedValue({ items: [] }),
  listSpaceResourceScopes: vi.fn().mockResolvedValue({ items: [] }),
  getPermissionMatrix: vi.fn().mockResolvedValue({
    roles: [],
    actions: [],
    builtinRoles: [],
    scenarioTools: [],
  }),
  getAuthMe: vi.fn().mockResolvedValue({
    user: { id: "u1" },
    space: { id: "local", name: "Local" },
    role: "viewer",
    permissions: [],
  }),
  listAuthSessions: vi.fn().mockResolvedValue({
    items: [{ sid: "asess_test", typ: "primary", did: "did_login", status: "active", exp: 0 }],
  }),
  revokeAuthSession: vi.fn(),
  refreshAuthSession: vi.fn(),
  createDeviceAuthSession: vi.fn().mockResolvedValue({
    token: "device-access",
    refreshToken: "device-refresh",
    user: { id: "u1" },
    space: { id: "local", name: "Local" },
    session: { sid: "asess_device", typ: "device", did: "laptop-1", status: "active", exp: 0 },
  }),
  listOrgTemplates: vi.fn().mockResolvedValue({
    items: [
      {
        id: "small_team",
        label: "小型团队",
        description: "fixture",
        payer: "工程经理",
        decisionMaker: "Tech Lead",
        approver: "Tech Lead",
      },
    ],
  }),
  provisionOrgTemplate: vi.fn(),
  createOrg: vi.fn(),
  createSpace: vi.fn(),
  createRole: vi.fn(),
  createSpaceMember: vi.fn(),
  updateSpaceResourceScope: vi.fn(),
  getSpaceRules: vi.fn().mockResolvedValue({
    spaceId: "local",
    version: 1,
    source: "default",
    builtin: true,
    updatedAt: 0,
    document: { version: 1, route: { hotfix: ["hotfix"] }, defaults: { policyProfile: "default" } },
  }),
  putSpaceRules: vi.fn(),
  importSpaceRules: vi.fn(),
  exportSpaceRules: vi.fn(),
  previewSpaceRules: vi.fn(),
  devLogin: vi.fn().mockResolvedValue({ token: "t", spaceId: "local" }),
}));

describe("SpacePage", () => {
  beforeEach(() => {
    localStorage.clear();
    vi.mocked(getReadyz).mockResolvedValue({ status: "ready", consoleAuthRequired: false });
  });

  it("renders space heading and Dev Token control", async () => {
    renderPage(<SpacePage />);
    expect(screen.getByRole("heading", { name: "空间" })).toBeInTheDocument();
    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Dev Token" })).toBeInTheDocument();
    });
  });

  it("hides Dev Token when console auth gate is on", async () => {
    vi.mocked(getReadyz).mockResolvedValue({ status: "ready", consoleAuthRequired: true });
    renderPage(<SpacePage />);
    await waitFor(() => {
      expect(screen.queryByRole("button", { name: "Dev Token" })).not.toBeInTheDocument();
    });
  });

  it("renders device mint form and shows one-shot result", async () => {
    const platform = await import("@/modules/platform/api/platform.api");
    localStorage.setItem("ash.auth.token", "primary-tok");
    renderPage(<SpacePage />);
    await waitFor(() => {
      expect(screen.getByTestId("device-mint-form")).toBeInTheDocument();
      expect(screen.getByTestId("device-mint-submit")).not.toBeDisabled();
    });
    fireEvent.change(screen.getByTestId("device-mint-id"), { target: { value: "laptop-1" } });
    fireEvent.click(screen.getByTestId("device-mint-submit"));
    await waitFor(() => {
      expect(platform.createDeviceAuthSession).toHaveBeenCalled();
      expect(screen.getByTestId("device-mint-result")).toBeInTheDocument();
      expect(screen.getByTestId("device-mint-token")).toHaveValue("device-access");
    });
  });

  it("renders auth sessions panel", async () => {
    renderPage(<SpacePage />);
    await waitFor(() => {
      expect(screen.getByTestId("auth-sessions-panel")).toBeInTheDocument();
      expect(screen.getByTestId("auth-sessions-table")).toBeInTheDocument();
      expect(screen.getByTestId("auth-session-refresh")).toBeInTheDocument();
    });
  });

  it("renders org templates panel", async () => {
    renderPage(<SpacePage />);
    await waitFor(() => {
      expect(screen.getByTestId("org-templates-panel")).toBeInTheDocument();
      expect(screen.getByText("小型团队")).toBeInTheDocument();
    });
  });

  it("renders space rules panel", async () => {
    renderPage(<SpacePage />);
    await waitFor(() => {
      expect(screen.getByTestId("space-rules-panel")).toBeInTheDocument();
      expect(screen.getByTestId("space-rules-editor")).toBeInTheDocument();
    });
  });
});
