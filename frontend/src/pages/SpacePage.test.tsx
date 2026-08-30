import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { SpacePage } from "./SpacePage";
import { renderPage } from "@/test/renderPage";

vi.mock("@tanstack/react-router", () => ({
  Link: ({ children, ...props }: { children: React.ReactNode; to?: string; className?: string }) => (
    <a href={props.to || "#"} className={props.className}>
      {children}
    </a>
  ),
}));

vi.mock("@/modules/platform/api/platform.api", () => ({
  listOrgs: vi.fn().mockResolvedValue({ items: [] }),
  listSpaces: vi.fn().mockResolvedValue({ items: [] }),
  listRoles: vi.fn().mockResolvedValue({ items: [] }),
  listSpaceMembers: vi.fn().mockResolvedValue({ items: [] }),
  listSpaceResourceScopes: vi.fn().mockResolvedValue({ items: [] }),
  getPermissionMatrix: vi.fn().mockResolvedValue({ roles: [], actions: [] }),
  getAuthMe: vi.fn().mockResolvedValue({ userId: "u1", spaceId: "local" }),
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
  it("renders space heading and Dev Token control", async () => {
    renderPage(<SpacePage />);
    expect(screen.getByRole("heading", { name: "空间" })).toBeInTheDocument();
    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Dev Token" })).toBeInTheDocument();
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
