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

vi.mock("@/modules/registry/api/registry.api", () => ({
  listAgentAssets: vi.fn().mockResolvedValue({ items: [] }),
  listMemoryAssets: vi.fn().mockResolvedValue({ items: [] }),
  getSpacePolicy: vi.fn().mockResolvedValue({
    pack: { spaceId: "local", citationMode: "optional", multiSign: false, reviewSlaHours: 72 },
    effective: {
      spaceId: "local",
      spaceKind: "team",
      citationMode: "required",
      multiSign: false,
      reviewSlaHours: 72,
      sources: ["kind:team"],
    },
  }),
  getSpaceQuotas: vi.fn().mockResolvedValue({
    spaceId: "local",
    limits: { maxConcurrentRuns: 0, tokenBudgetProxy: 0 },
    usage: { activeConcurrentRuns: 0, tokenBudgetProxyUsed: 0 },
  }),
  putSpacePolicy: vi.fn(),
  createAgentAsset: vi.fn(),
  createMemoryAsset: vi.fn(),
  patchAgentAssetStatus: vi.fn(),
  patchMemoryAssetStatus: vi.fn(),
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
    const createSpace = screen.getByRole("button", { name: "创建空间" });
    expect(createSpace).toBeDisabled();
    expect(createSpace).toHaveAttribute("title", "需要先创建组织");
    const createOrg = screen.getByTestId("space-org-create");
    expect(createOrg).toBeDisabled();
    expect(createOrg).toHaveAttribute("title", "需要填写组织名称");
    const createRole = screen.getByRole("button", { name: "创建角色" });
    expect(createRole).toBeDisabled();
    expect(createRole).toHaveAttribute("title", "需要先创建或选择组织");
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
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(true);
    fireEvent.click(screen.getByTestId("device-mint-submit"));
    await waitFor(() => {
      expect(confirmSpy).toHaveBeenCalled();
      expect(platform.createDeviceAuthSession).toHaveBeenCalled();
      expect(screen.getByTestId("device-mint-result")).toBeInTheDocument();
      expect(screen.getByTestId("device-mint-token")).toHaveValue("device-access");
    });
    confirmSpy.mockRestore();
  });

  it("requires confirm before minting device token", async () => {
    const platform = await import("@/modules/platform/api/platform.api");
    vi.mocked(platform.createDeviceAuthSession).mockClear();
    localStorage.setItem("ash.auth.token", "primary-tok");
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    renderPage(<SpacePage />);
    const mint = await screen.findByTestId("device-mint-submit");
    expect(mint).toHaveAttribute("title", "签发 device token（需确认）");
    fireEvent.click(mint);
    expect(confirmSpy).toHaveBeenCalled();
    expect(platform.createDeviceAuthSession).not.toHaveBeenCalled();
    confirmSpy.mockRestore();
  });

  it("disables device mint when TTL is not a positive integer", async () => {
    localStorage.setItem("ash.auth.token", "primary-tok");
    renderPage(<SpacePage />);
    const mint = await screen.findByTestId("device-mint-submit");
    fireEvent.change(screen.getByTestId("device-mint-ttl"), { target: { value: "abc" } });
    expect(mint).toBeDisabled();
    expect(mint).toHaveAttribute("title", "TTL 须为正整数秒（可空）");
    fireEvent.change(screen.getByTestId("device-mint-ttl"), { target: { value: "-1" } });
    expect(mint).toBeDisabled();
    fireEvent.change(screen.getByTestId("device-mint-ttl"), { target: { value: "0" } });
    expect(mint).toBeDisabled();
    fireEvent.change(screen.getByTestId("device-mint-ttl"), { target: { value: "3600" } });
    expect(mint).not.toBeDisabled();
    expect(mint).toHaveAttribute("title", "签发 device token（需确认）");
  });

  it("requires confirm before creating an organization", async () => {
    const platform = await import("@/modules/platform/api/platform.api");
    vi.mocked(platform.createOrg).mockResolvedValue({ id: "org_spot", name: "Spot Org" } as never);
    vi.mocked(platform.createOrg).mockClear();
    renderPage(<SpacePage />);
    const createOrg = await screen.findByTestId("space-org-create");
    fireEvent.change(screen.getByTestId("space-org-name"), { target: { value: "Spot Org" } });
    expect(createOrg).toHaveAttribute("title", "创建组织（需确认）");
    const form = createOrg.closest("form");
    expect(form).toBeTruthy();
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    fireEvent.submit(form!);
    expect(confirmSpy).toHaveBeenCalled();
    expect(platform.createOrg).not.toHaveBeenCalled();
    confirmSpy.mockReturnValue(true);
    fireEvent.submit(form!);
    await waitFor(() => {
      expect(platform.createOrg).toHaveBeenCalledWith(
        expect.objectContaining({ name: "Spot Org" }),
        expect.anything(),
      );
    });
    confirmSpy.mockRestore();
  });

  it("renders auth sessions panel", async () => {
    renderPage(<SpacePage />);
    await waitFor(() => {
      expect(screen.getByTestId("auth-sessions-panel")).toBeInTheDocument();
      expect(screen.getByTestId("auth-sessions-table")).toBeInTheDocument();
      expect(screen.getByTestId("auth-session-refresh")).toBeInTheDocument();
    });
    expect(screen.getByTestId("auth-session-refresh")).toHaveAttribute("title");
    expect(screen.getByTestId("device-mint-submit")).toHaveAttribute("title");
  });

  it("requires confirm before refreshing auth session token", async () => {
    localStorage.setItem("ash.auth.token", "primary-tok");
    const { refreshAuthSession } = await import("@/modules/platform/api/platform.api");
    vi.mocked(refreshAuthSession).mockClear();
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    renderPage(<SpacePage />);
    const refresh = await screen.findByTestId("auth-session-refresh");
    expect(refresh).toHaveAttribute("title", "刷新当前会话 token（需确认）");
    fireEvent.click(refresh);
    expect(confirmSpy).toHaveBeenCalled();
    expect(refreshAuthSession).not.toHaveBeenCalled();
    confirmSpy.mockRestore();
  });

  it("requires confirm before revoking auth session", async () => {
    localStorage.setItem("ash.auth.token", "primary-tok");
    const { revokeAuthSession } = await import("@/modules/platform/api/platform.api");
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    renderPage(<SpacePage />);
    const revoke = await screen.findByTestId("space-session-revoke-asess_test");
    expect(revoke).toHaveAttribute("title", "吊销会话 asess_test（需确认）");
    fireEvent.click(revoke);
    expect(confirmSpy).toHaveBeenCalled();
    expect(revokeAuthSession).not.toHaveBeenCalled();

    confirmSpy.mockReturnValue(true);
    fireEvent.click(revoke);
    await waitFor(() => {
      expect(revokeAuthSession).toHaveBeenCalledWith("asess_test");
    });
    confirmSpy.mockRestore();
  });

  it("renders org templates panel with provision title", async () => {
    renderPage(<SpacePage />);
    await waitFor(() => {
      expect(screen.getByTestId("org-templates-panel")).toBeInTheDocument();
      expect(screen.getByText("小型团队")).toBeInTheDocument();
    });
    const provision = screen.getByTestId("space-template-provision");
    expect(provision).toHaveAttribute("title", "按所选样板开通 Org / Space / 角色（需确认）");
  });

  it("requires confirm before provisioning org template", async () => {
    const { provisionOrgTemplate } = await import("@/modules/platform/api/platform.api");
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    renderPage(<SpacePage />);
    const provision = await screen.findByTestId("space-template-provision");
    const form = provision.closest("form");
    expect(form).toBeTruthy();
    fireEvent.submit(form!);
    expect(confirmSpy).toHaveBeenCalled();
    expect(provisionOrgTemplate).not.toHaveBeenCalled();

    confirmSpy.mockReturnValue(true);
    fireEvent.submit(form!);
    await waitFor(() => {
      expect(provisionOrgTemplate).toHaveBeenCalled();
    });
    confirmSpy.mockRestore();
  });

  it("renders space rules panel", async () => {
    renderPage(<SpacePage />);
    await waitFor(() => {
      expect(screen.getByTestId("space-rules-panel")).toBeInTheDocument();
      expect(screen.getByTestId("space-rules-editor")).toBeInTheDocument();
    });
  });

  it("disables rules save for empty or invalid JSON (never coerce to {})", async () => {
    const { putSpaceRules } = await import("@/modules/platform/api/platform.api");
    renderPage(<SpacePage />);
    const editor = await screen.findByTestId("space-rules-editor");
    await waitFor(() => {
      expect((editor as HTMLTextAreaElement).value).toMatch(/version/);
    });
    const save = screen.getByTestId("space-rules-save");
    expect(save).not.toBeDisabled();

    fireEvent.change(editor, { target: { value: "" } });
    expect(save).toBeDisabled();
    expect(save).toHaveAttribute("title", "需要填写规则 JSON");

    fireEvent.change(editor, { target: { value: "[]" } });
    expect(save).toBeDisabled();
    expect(save).toHaveAttribute("title", "规则必须是合法 JSON 对象");

    fireEvent.change(editor, { target: { value: "{" } });
    expect(save).toBeDisabled();
    expect(save).toHaveAttribute("title", "规则 JSON 不合法");

    fireEvent.change(editor, { target: { value: "{}" } });
    expect(save).toBeDisabled();
    expect(save).toHaveAttribute("title", "规则 JSON 对象不能为空");

    fireEvent.click(save);
    expect(putSpaceRules).not.toHaveBeenCalled();
  });

  it("requires confirm before saving or importing space rules", async () => {
    const { putSpaceRules, importSpaceRules } = await import("@/modules/platform/api/platform.api");
    vi.mocked(putSpaceRules).mockResolvedValue({ ok: true } as never);
    vi.mocked(importSpaceRules).mockResolvedValue({ ok: true } as never);
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    renderPage(<SpacePage />);
    const save = await screen.findByTestId("space-rules-save");
    await waitFor(() => expect(save).not.toBeDisabled());
    expect(save).toHaveAttribute("title", "将规则保存到 DB（需确认）");
    fireEvent.click(save);
    expect(confirmSpy).toHaveBeenCalled();
    expect(putSpaceRules).not.toHaveBeenCalled();

    const imp = screen.getByTestId("space-rules-import");
    expect(imp).toHaveAttribute("title", "从文件导入规则到 DB（需确认）");
    fireEvent.click(imp);
    expect(importSpaceRules).not.toHaveBeenCalled();

    confirmSpy.mockReturnValue(true);
    fireEvent.click(save);
    await waitFor(() => {
      expect(putSpaceRules).toHaveBeenCalled();
    });
    confirmSpy.mockRestore();
  });

  it("requires confirm before issuing local Dev Token", async () => {
    const { devLogin } = await import("@/modules/platform/api/platform.api");
    vi.mocked(devLogin).mockClear();
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    renderPage(<SpacePage />);
    const btn = await screen.findByTestId("dev-token-btn");
    expect(btn).toHaveAttribute("title", "签发本地 Dev Token（需确认）");
    fireEvent.click(btn);
    expect(confirmSpy).toHaveBeenCalled();
    expect(devLogin).not.toHaveBeenCalled();
    confirmSpy.mockRestore();
  });

  it("disables rules import/export when repoRoot is empty", async () => {
    const platform = await import("@/modules/platform/api/platform.api");
    renderPage(<SpacePage />);
    await screen.findByTestId("space-rules-panel");
    const imp = screen.getByTestId("space-rules-import");
    const exp = screen.getByTestId("space-rules-export");
    expect(imp).not.toBeDisabled();
    expect(exp).not.toBeDisabled();

    fireEvent.change(screen.getByTestId("space-rules-repo-root"), { target: { value: "" } });
    expect(imp).toBeDisabled();
    expect(exp).toBeDisabled();
    expect(imp).toHaveAttribute("title", "需要填写 repoRoot");
    expect(exp).toHaveAttribute("title", "需要填写 repoRoot");
    fireEvent.click(imp);
    fireEvent.click(exp);
    expect(platform.importSpaceRules).not.toHaveBeenCalled();
    expect(platform.exportSpaceRules).not.toHaveBeenCalled();
  });

  it("does not refetch the previous space rules with the new space token", async () => {
    const platform = await import("@/modules/platform/api/platform.api");
    const ruleCalls: string[] = [];
    vi.mocked(platform.listOrgs).mockResolvedValue({
      items: [{ id: "org_1", name: "Org", slug: "org" }],
    });
    vi.mocked(platform.createSpace).mockResolvedValue({
      id: "space_new",
      orgId: "org_1",
      name: "Spot",
      slug: "spot",
    });
    vi.mocked(platform.devLogin).mockResolvedValue({
      token: "new-token",
      user: { id: "dev-user", displayName: "Dev User" },
      space: { id: "space_new", name: "Spot" },
    });
    vi.mocked(platform.getSpaceRules).mockImplementation(async (spaceId: string) => {
      ruleCalls.push(`${spaceId}|${localStorage.getItem("ash.auth.token") || ""}`);
      return {
        spaceId,
        version: 1,
        source: "default",
        builtin: true,
        updatedAt: 0,
        document: { version: 1, route: { hotfix: ["hotfix"] }, defaults: { policyProfile: "default" } },
      };
    });
    renderPage(<SpacePage />);
    const submit = await screen.findByRole("button", { name: "创建空间" });
    expect(submit).toBeDisabled();
    fireEvent.change(submit.closest("form")!.querySelector('input[name="name"]')!, {
      target: { value: "Spot" },
    });
    await waitFor(() => {
      expect(submit).toBeEnabled();
    });
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(true);
    fireEvent.click(submit);
    await waitFor(() => {
      expect(ruleCalls.some((call) => call.startsWith("space_new|new-token"))).toBe(true);
    });
    expect(ruleCalls.some((call) => call === "local|new-token")).toBe(false);
    confirmSpy.mockRestore();
  });

  it("renders Registry assets panel and EffectivePolicy summary", async () => {
    renderPage(<SpacePage />);
    await waitFor(() => {
      expect(screen.getByTestId("registry-assets-panel")).toBeInTheDocument();
      expect(screen.getByText("管控登记 / 策略")).toBeInTheDocument();
      expect(screen.getByTestId("space-quotas-summary")).toHaveTextContent("配额");
      expect(screen.getByTestId("effective-policy-summary")).toHaveTextContent("生效策略");
      expect(screen.getByTestId("effective-policy-summary")).toHaveTextContent("引用=required");
      expect(screen.getByTestId("agent-assets-list")).toBeInTheDocument();
      expect(screen.getByTestId("memory-assets-list")).toBeInTheDocument();
    });
  });

  it("tolerates null builtin role permissions in matrix", async () => {
    const platform = await import("@/modules/platform/api/platform.api");
    localStorage.setItem("ash.auth.token", "tok");
    vi.mocked(platform.getPermissionMatrix).mockImplementation(async () => ({
      spaceId: "local",
      catalog: [],
      builtinRoles: [{ name: "viewer", label: "只读", permissions: null as unknown as string[] }],
      scenarioTools: [],
    }));
    const { container } = renderPage(<SpacePage />);
    await waitFor(() => {
      expect(container.textContent).toContain("viewer");
    });
    expect(container.textContent).toContain("权限矩阵");
  });

  it("disables rules preview when Goal is empty", async () => {
    renderPage(<SpacePage />);
    const goal = await screen.findByTestId("space-rules-preview-goal");
    fireEvent.change(goal, { target: { value: "" } });
    const btn = screen.getByTestId("space-rules-preview");
    expect(btn).toBeDisabled();
    expect(btn).toHaveAttribute("title", "需要填写预览 Goal");
  });

  it("surfaces space rules preview API error message", async () => {
    const platform = await import("@/modules/platform/api/platform.api");
    vi.mocked(platform.previewSpaceRules).mockRejectedValueOnce(new Error("goal is required"));
    renderPage(<SpacePage />);
    const goal = await screen.findByTestId("space-rules-preview-goal");
    fireEvent.change(goal, { target: { value: "fix CVE" } });
    fireEvent.click(screen.getByTestId("space-rules-preview"));
    expect(await screen.findByTestId("space-rules-error")).toHaveTextContent("goal is required");
  });
});
