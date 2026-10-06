import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { flushSync } from "react-dom";
import { Building2, KeyRound, Plus, ShieldCheck, UsersRound } from "lucide-react";
import { useEffect, useMemo, useState, type FormEvent } from "react";
import {
  createOrg,
  createRole,
  createSpace,
  createSpaceMember,
  devLogin,
  getAuthMe,
  getSpaceRules,
  putSpaceRules,
  importSpaceRules,
  exportSpaceRules,
  previewSpaceRules,
  listAuthSessions,
  listOrgTemplates,
  listOrgs,
  listRoles,
  getPermissionMatrix,
  listSpaceMembers,
  listSpaceResourceScopes,
  listSpaces,
  provisionOrgTemplate,
  createDeviceAuthSession,
  refreshAuthSession,
  revokeAuthSession,
  updateSpaceResourceScope,
} from "@/modules/platform/api/platform.api";
import { getReadyz } from "@/modules/health/api/health.api";
import { isConsoleAuthRequiredFlag } from "@/modules/platform/auth/consoleGate";
import { RegistryAssetsPanel } from "@/modules/registry/components/RegistryAssetsPanel";
import { getAuthToken, getCurrentSpaceId, getRefreshToken, setAuthSession } from "@/services/http/client";
import type { AuthSessionResponse } from "@/modules/platform/api/platform.api";

/** Space rules editor must be a non-empty JSON object (never coerce "" → {}). */
export function validateSpaceRulesDraft(raw: string): string | null {
  const trimmed = raw.trim();
  if (!trimmed) return "需要填写规则 JSON";
  try {
    const doc = JSON.parse(trimmed) as unknown;
    if (doc === null || typeof doc !== "object" || Array.isArray(doc)) {
      return "规则必须是合法 JSON 对象";
    }
    if (Object.keys(doc as Record<string, unknown>).length === 0) {
      return "规则 JSON 对象不能为空";
    }
    return null;
  } catch {
    return "规则 JSON 不合法";
  }
}

export function SpacePage() {
  const qc = useQueryClient();
  const [activeSpaceId, setActiveSpaceId] = useState(getCurrentSpaceId());
  const [templateId, setTemplateId] = useState("small_team");
  const [deviceId, setDeviceId] = useState("");
  const [deviceScope, setDeviceScope] = useState("");
  const [deviceTtl, setDeviceTtl] = useState("");
  const deviceTtlIssue = (() => {
    const raw = deviceTtl.trim();
    if (!raw) return "";
    const n = Number(raw);
    if (!Number.isInteger(n) || n <= 0) return "TTL 须为正整数秒（可空）";
    return "";
  })();
  const [orgName, setOrgName] = useState("");
  const [spaceName, setSpaceName] = useState("");
  const [roleName, setRoleName] = useState("");
  const [mintResult, setMintResult] = useState<AuthSessionResponse | null>(null);
  const gateQuery = useQuery({
    queryKey: ["console-auth-gate"],
    queryFn: getReadyz,
    staleTime: 60_000,
  });
  const consoleAuthRequired = isConsoleAuthRequiredFlag(gateQuery.data ?? {});
  const orgsQuery = useQuery({
    queryKey: ["orgs"],
    queryFn: listOrgs,
  });
  const templatesQuery = useQuery({
    queryKey: ["org-templates"],
    queryFn: listOrgTemplates,
  });
  const spacesQuery = useQuery({
    queryKey: ["spaces"],
    queryFn: listSpaces,
  });
  const firstOrgId = orgsQuery.data?.items?.[0]?.id ?? "";
  const activeSpace = (spacesQuery.data?.items ?? []).find((space) => space.id === activeSpaceId);
  const activeOrgId = activeSpace?.orgId || firstOrgId;
  const canManageActiveSpace = Boolean(activeSpaceId && activeSpaceId !== "local" && activeSpace);
  const rolesQuery = useQuery({
    queryKey: ["roles", activeOrgId],
    queryFn: () => listRoles(activeOrgId),
    enabled: Boolean(activeOrgId),
  });
  const membersQuery = useQuery({
    queryKey: ["members", activeSpaceId],
    queryFn: () => listSpaceMembers(activeSpaceId),
    enabled: canManageActiveSpace,
  });
  const meQuery = useQuery({
    queryKey: ["auth-me", activeSpaceId],
    queryFn: getAuthMe,
    enabled: Boolean(getAuthToken()),
  });
  const sessionsQuery = useQuery({
    queryKey: ["auth-sessions"],
    queryFn: listAuthSessions,
    enabled: Boolean(getAuthToken()),
  });
  const revokeSessionMut = useMutation({
    mutationFn: (sid: string) => revokeAuthSession(sid),
    onSuccess: async () => {
      await qc.invalidateQueries({ queryKey: ["auth-sessions"] });
      await qc.invalidateQueries({ queryKey: ["auth-me"] });
    },
  });
  const refreshSessionMut = useMutation({
    mutationFn: () => refreshAuthSession(),
    onSuccess: async (data) => {
      setAuthSession(data.token, data.space.id, data.refreshToken);
      setActiveSpaceId(data.space.id);
      await qc.invalidateQueries({ queryKey: ["auth-sessions"] });
      await qc.invalidateQueries({ queryKey: ["auth-me"] });
    },
  });
  const mintDeviceMut = useMutation({
    mutationFn: () => {
      const scope = deviceScope
        .split(",")
        .map((s) => s.trim())
        .filter(Boolean);
      const ttl = Number(deviceTtl);
      return createDeviceAuthSession({
        deviceId: deviceId.trim() || undefined,
        scope: scope.length ? scope : undefined,
        ttlSeconds: Number.isFinite(ttl) && ttl > 0 ? ttl : undefined,
      });
    },
    onSuccess: async (data) => {
      setMintResult(data);
      await qc.invalidateQueries({ queryKey: ["auth-sessions"] });
    },
  });
  const matrixQuery = useQuery({
    queryKey: ["permissions-matrix", activeSpaceId],
    queryFn: () => getPermissionMatrix(activeSpaceId !== "local" ? activeSpaceId : undefined),
    enabled: Boolean(getAuthToken()),
  });
  const scopesQuery = useQuery({
    queryKey: ["resource-scopes", activeSpaceId],
    queryFn: () => listSpaceResourceScopes(activeSpaceId),
    enabled: canManageActiveSpace,
  });
  const [policyDrafts, setPolicyDrafts] = useState<Record<string, string>>({});
  const [rulesYamlHint, setRulesYamlHint] = useState(".");
  const [previewGoal, setPreviewGoal] = useState("fix CVE in auth");
  const [rulesDraft, setRulesDraft] = useState("");
  const rulesQuery = useQuery({
    queryKey: ["space-rules", activeSpaceId],
    queryFn: () => getSpaceRules(activeSpaceId || "local"),
  });
  const rulesMut = useMutation({
    mutationFn: () => {
      const issue = validateSpaceRulesDraft(rulesDraft);
      if (issue) throw new Error(issue);
      const document = JSON.parse(rulesDraft.trim()) as Record<string, unknown>;
      return putSpaceRules(activeSpaceId || "local", document);
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ["space-rules"] }),
  });
  const importRulesMut = useMutation({
    mutationFn: () => importSpaceRules(activeSpaceId || "local", rulesYamlHint),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["space-rules"] }),
  });
  const exportRulesMut = useMutation({
    mutationFn: () => exportSpaceRules(activeSpaceId || "local", rulesYamlHint),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["space-rules"] }),
  });
  const previewRulesMut = useMutation({
    mutationFn: () => previewSpaceRules(activeSpaceId || "local", { goal: previewGoal, repoRoot: rulesYamlHint }),
  });
  const rulesDraftIssue = useMemo(() => validateSpaceRulesDraft(rulesDraft), [rulesDraft]);
  const rulesRepoReady = Boolean(rulesYamlHint.trim());
  useEffect(() => {
    if (rulesQuery.data?.document) {
      setRulesDraft(JSON.stringify(rulesQuery.data.document, null, 2));
    }
  }, [rulesQuery.data]);
  function commitActiveSpace(token: string, spaceId: string, refreshToken?: string) {
    setAuthSession(token, spaceId, refreshToken);
    flushSync(() => {
      setActiveSpaceId(spaceId);
    });
  }
  const loginMut = useMutation({
    mutationFn: (spaceId?: string) => devLogin(spaceId),
    onSuccess: async (data) => {
      commitActiveSpace(data.token, data.space.id);
      await qc.invalidateQueries();
    },
  });
  function activateSpace(spaceId: string) {
    if (consoleAuthRequired && getAuthToken()) {
      commitActiveSpace(getAuthToken(), spaceId, getRefreshToken() || undefined);
      void qc.invalidateQueries();
      return;
    }
    loginMut.mutate(spaceId);
  }
  const createOrgMut = useMutation({
    mutationFn: createOrg,
    onSuccess: async () => {
      setOrgName("");
      await qc.invalidateQueries({ queryKey: ["orgs"] });
    },
  });
  const provisionTemplateMut = useMutation({
    mutationFn: (body: { templateId: string; name?: string; slug?: string }) =>
      provisionOrgTemplate(body.templateId, { name: body.name, slug: body.slug }),
    onSuccess: async (result) => {
      await qc.invalidateQueries({ queryKey: ["orgs"] });
      await qc.invalidateQueries({ queryKey: ["spaces"] });
      await qc.invalidateQueries({ queryKey: ["roles"] });
      const firstSpace = result.spaces[0]?.id;
      if (firstSpace) {
        activateSpace(firstSpace);
      }
    },
  });
  const createSpaceMut = useMutation({
    mutationFn: createSpace,
    onSuccess: async (space) => {
      setSpaceName("");
      await qc.invalidateQueries({ queryKey: ["spaces"] });
      activateSpace(space.id);
    },
  });
  const updateScopeMut = useMutation({
    mutationFn: (body: { scopeId: string; policyJson: string }) =>
      updateSpaceResourceScope(activeSpaceId, body.scopeId, body.policyJson),
    onSuccess: async () => {
      await qc.invalidateQueries({ queryKey: ["resource-scopes", activeSpaceId] });
      await qc.invalidateQueries({ queryKey: ["permissions-matrix", activeSpaceId] });
    },
  });
  const createRoleMut = useMutation({
    mutationFn: (body: { orgId: string; name: string; permissions: string[] }) =>
      createRole(body.orgId, { name: body.name, permissions: body.permissions }),
    onSuccess: async () => {
      setRoleName("");
      await qc.invalidateQueries({ queryKey: ["roles", activeOrgId] });
    },
  });
  const createMemberMut = useMutation({
    mutationFn: (body: { spaceId: string; userId?: string; email?: string; displayName?: string; password?: string; roleId: string }) =>
      createSpaceMember(body.spaceId, {
        userId: body.userId,
        email: body.email,
        displayName: body.displayName,
        password: body.password,
        roleId: body.roleId,
      }),
    onSuccess: async () => {
      await qc.invalidateQueries({ queryKey: ["members", activeSpaceId] });
    },
  });

  function submitOrg(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const name = orgName.trim();
    if (!name) return;
    const formData = new FormData(event.currentTarget);
    const ok = window.confirm(`确认创建组织「${name}」？`);
    if (!ok) return;
    createOrgMut.mutate({
      name,
      slug: String(formData.get("slug") || "").trim() || undefined,
    });
  }

  function submitTemplate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const formData = new FormData(event.currentTarget);
    const selected = String(formData.get("templateId") || templateId);
    const label =
      (templatesQuery.data?.items ?? []).find((item) => item.id === selected)?.label || selected;
    const ok = window.confirm(`确认按样板「${label}」一键开通 Org / Space / 角色？`);
    if (!ok) return;
    provisionTemplateMut.mutate({
      templateId: selected,
      name: String(formData.get("name") || "") || undefined,
      slug: String(formData.get("slug") || "") || undefined,
    });
  }

  function submitSpace(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!spaceName.trim() || !firstOrgId) return;
    const formData = new FormData(event.currentTarget);
    const ok = window.confirm(`确认创建空间「${spaceName.trim()}」？`);
    if (!ok) return;
    createSpaceMut.mutate({
      orgId: String(formData.get("orgId") || firstOrgId),
      name: spaceName.trim(),
      slug: String(formData.get("slug") || "") || undefined,
    });
  }

  function submitRole(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!roleName.trim() || !activeOrgId) return;
    const formData = new FormData(event.currentTarget);
    const orgId = String(formData.get("orgId") || activeOrgId);
    const rawPermissions = String(formData.get("permissions") || "");
    const ok = window.confirm(`确认创建角色「${roleName.trim()}」？`);
    if (!ok) return;
    createRoleMut.mutate({
      orgId,
      name: roleName.trim(),
      permissions: rawPermissions
        .split(/[\n,]/)
        .map((item) => item.trim())
        .filter(Boolean),
    });
  }

  function submitMember(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const formData = new FormData(event.currentTarget);
    const roleId = String(formData.get("roleId") || "").trim();
    if (!roleId) return;
    const label =
      String(formData.get("email") || formData.get("displayName") || formData.get("userId") || "").trim() ||
      "新成员";
    const ok = window.confirm(`确认添加成员「${label}」到当前 Space？`);
    if (!ok) return;
    createMemberMut.mutate({
      spaceId: activeSpaceId,
      userId: String(formData.get("userId") || "") || undefined,
      email: String(formData.get("email") || "") || undefined,
      displayName: String(formData.get("displayName") || "") || undefined,
      password: String(formData.get("password") || "") || undefined,
      roleId,
    });
  }

  const selectedTemplate = (templatesQuery.data?.items ?? []).find((item) => item.id === templateId);
  const err =
    orgsQuery.error?.message ||
    templatesQuery.error?.message ||
    spacesQuery.error?.message ||
    rolesQuery.error?.message ||
    membersQuery.error?.message ||
    createOrgMut.error?.message ||
    provisionTemplateMut.error?.message ||
    createSpaceMut.error?.message ||
    createRoleMut.error?.message ||
    createMemberMut.error?.message;

  return (
    <section className="panel active">
      <div className="page-kicker">
        <Building2 size={17} strokeWidth={1.8} />
        Space Scope
      </div>
      <div className="page-heading">
        <div>
          <h1>空间</h1>
          <p>
            查看当前控制台可见的空间范围和开发身份。TR2 合规检查见{" "}
            <Link to="/compliance" className="inline-link">
              合规控制台
            </Link>
            。
          </p>
        </div>
        <div className="toolbar">
          <Link to="/login" className="btn icon-btn">
            登录页
          </Link>
          {!consoleAuthRequired && (
            <button
              className="btn icon-btn"
              data-testid="dev-token-btn"
              onClick={() => {
                const ok = window.confirm(
                  `确认签发 Space「${activeSpaceId}」的本地 Dev Token？`,
                );
                if (!ok) return;
                loginMut.mutate(activeSpaceId);
              }}
              disabled={loginMut.isPending}
              title={loginMut.isPending ? "签发中…" : "签发本地 Dev Token（需确认）"}
            >
              <KeyRound size={16} strokeWidth={1.8} />
              Dev Token
            </button>
          )}
        </div>
      </div>
      {err && <p className="error-text">{err}</p>}
      <div className="pane" style={{ marginBottom: "1rem" }} data-testid="auth-sessions-panel">
        <div className="pane-title">
          <h2>Auth Sessions</h2>
          <span>{sessionsQuery.data?.items?.length ?? 0}</span>
        </div>
        <p className="muted-line">
          多端会话（DX57–58）：当前{" "}
          {meQuery.data?.session?.sid ? (
            <code>{meQuery.data.session.sid}</code>
          ) : (
            "—"
          )}
          {meQuery.data?.session?.typ ? ` · ${meQuery.data.session.typ}` : ""}
        </p>
        <div className="toolbar" style={{ marginBottom: "0.5rem", gap: "0.5rem" }}>
          <button
            className="btn"
            type="button"
            data-testid="auth-session-refresh"
            onClick={() => {
              const ok = window.confirm("确认刷新当前会话 token？旧 token 可能立即失效。");
              if (!ok) return;
              refreshSessionMut.mutate();
            }}
            disabled={!getAuthToken() || refreshSessionMut.isPending}
            title={
              !getAuthToken()
                ? "需要先登录获取 token"
                : refreshSessionMut.isPending
                  ? "刷新中…"
                  : "刷新当前会话 token（需确认）"
            }
          >
            {refreshSessionMut.isPending ? "刷新中…" : "Refresh 当前 token"}
          </button>
        </div>
        <form
          className="stack-form"
          data-testid="device-mint-form"
          style={{ marginBottom: "0.75rem" }}
          onSubmit={(e) => {
            e.preventDefault();
            if (deviceTtlIssue) return;
            const ok = window.confirm(
              "确认签发 device token？令牌仅展示一次，请妥善保存。",
            );
            if (!ok) return;
            mintDeviceMut.mutate();
          }}
        >
          <p className="muted-line">Mint device token（展示一次；不切换控制台会话）</p>
          <label>
            Device ID
            <input
              data-testid="device-mint-id"
              value={deviceId}
              onChange={(e) => setDeviceId(e.target.value)}
              placeholder="laptop-1（可空）"
            />
          </label>
          <label>
            Scope（逗号分隔，可空）
            <input
              data-testid="device-mint-scope"
              value={deviceScope}
              onChange={(e) => setDeviceScope(e.target.value)}
              placeholder="run:create,artifact:read"
            />
          </label>
          <label>
            TTL 秒（可空）
            <input
              data-testid="device-mint-ttl"
              value={deviceTtl}
              onChange={(e) => setDeviceTtl(e.target.value)}
              placeholder="3600"
              inputMode="numeric"
            />
          </label>
          <button
            className="btn primary"
            type="submit"
            data-testid="device-mint-submit"
            disabled={!getAuthToken() || mintDeviceMut.isPending || Boolean(deviceTtlIssue)}
            title={
              !getAuthToken()
                ? "需要先登录获取 token"
                : deviceTtlIssue
                  ? deviceTtlIssue
                  : mintDeviceMut.isPending
                    ? "签发中…"
                    : "签发 device token（需确认）"
            }
          >
            {mintDeviceMut.isPending ? "Minting…" : "Mint device"}
          </button>
        </form>
        {mintDeviceMut.error && (
          <p className="error-text" data-testid="device-mint-error">
            {(mintDeviceMut.error as Error).message}
          </p>
        )}
        {mintResult && (
          <div className="pane" data-testid="device-mint-result" style={{ marginBottom: "0.75rem" }}>
            <p className="muted-line">
              已签发 device（请立即复制；控制台仍使用原 primary）
              {mintResult.session?.sid ? (
                <>
                  {" "}
                  · sid <code>{mintResult.session.sid}</code>
                </>
              ) : null}
              {mintResult.session?.did ? (
                <>
                  {" "}
                  · did <code>{mintResult.session.did}</code>
                </>
              ) : null}
            </p>
            <label>
              access token
              <textarea data-testid="device-mint-token" readOnly rows={2} value={mintResult.token} />
            </label>
            {mintResult.refreshToken ? (
              <label>
                refresh token
                <textarea data-testid="device-mint-refresh" readOnly rows={2} value={mintResult.refreshToken} />
              </label>
            ) : null}
          </div>
        )}
        {(refreshSessionMut.error || revokeSessionMut.error || sessionsQuery.error) && (
          <p className="error-text">
            {(refreshSessionMut.error as Error | undefined)?.message ||
              (revokeSessionMut.error as Error | undefined)?.message ||
              (sessionsQuery.error as Error | undefined)?.message}
          </p>
        )}
        <div className="table-wrap">
          <table data-testid="auth-sessions-table">
            <thead>
              <tr>
                <th>sid</th>
                <th>typ</th>
                <th>did</th>
                <th>status</th>
                <th>exp</th>
                <th />
              </tr>
            </thead>
            <tbody>
              {(sessionsQuery.data?.items ?? []).map((row) => (
                <tr key={row.sid}>
                  <td>
                    <code>{row.sid}</code>
                  </td>
                  <td>{row.typ}</td>
                  <td>{row.did || "—"}</td>
                  <td>{row.status || "—"}</td>
                  <td>{row.exp ? new Date(row.exp * 1000).toISOString() : "—"}</td>
                  <td>
                    <button
                      className="btn icon-btn"
                      type="button"
                      disabled={row.status === "revoked" || revokeSessionMut.isPending}
                      onClick={() => {
                        if (row.status === "revoked") return;
                        const ok = window.confirm(`确认吊销会话 ${row.sid}？吊销后对应 token 将失效。`);
                        if (!ok) return;
                        revokeSessionMut.mutate(row.sid);
                      }}
                      title={
                        row.status === "revoked"
                          ? "会话已吊销"
                          : revokeSessionMut.isPending
                            ? "吊销中…"
                            : `吊销会话 ${row.sid}（需确认）`
                      }
                      data-testid={`space-session-revoke-${row.sid}`}
                    >
                      Revoke
                    </button>
                  </td>
                </tr>
              ))}
              {!sessionsQuery.data?.items?.length && (
                <tr>
                  <td colSpan={6} className="muted-line">
                    {getAuthToken() ? "暂无会话登记（或未登录 JWT）" : consoleAuthRequired ? "请先登录" : "请先登录或 Dev Token"}
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>
      <div className="pane" style={{ marginBottom: "1rem" }} data-testid="org-templates-panel">
        <div className="pane-title">
          <h2>组织样板（PRD §3）</h2>
          <span>{templatesQuery.data?.items?.length ?? 0} 套</span>
        </div>
        <p className="muted-line">一键开通 Org / Space / 角色；标明谁付费、谁决策、谁审批。</p>
        <form className="stack-form" onSubmit={submitTemplate}>
          <label className="scenario-picker">
            样板
            <select
              name="templateId"
              value={templateId}
              onChange={(e) => setTemplateId(e.target.value)}
              disabled={!templatesQuery.data?.items?.length || provisionTemplateMut.isPending}
            >
              {(templatesQuery.data?.items ?? []).map((item) => (
                <option key={item.id} value={item.id}>
                  {item.label}
                </option>
              ))}
              {!templatesQuery.data?.items?.length && <option value="small_team">小团队</option>}
            </select>
          </label>
          {selectedTemplate && (
            <p className="muted-line">
              付费：{selectedTemplate.payer} · 决策：{selectedTemplate.decisionMaker} · 审批：{selectedTemplate.approver}
            </p>
          )}
          <input name="name" placeholder="组织名称（可空=样板默认）" />
          <input name="slug" placeholder="slug（可空）" />
          <button
            className="btn primary icon-btn"
            type="submit"
            disabled={!templatesQuery.data?.items?.length || provisionTemplateMut.isPending}
            title={
              !templatesQuery.data?.items?.length
                ? "样板列表加载中或不可用"
                : provisionTemplateMut.isPending
                  ? "开通中…"
                  : "按所选样板开通 Org / Space / 角色（需确认）"
            }
            data-testid="space-template-provision"
          >
            <Plus size={16} strokeWidth={1.8} />
            {provisionTemplateMut.isPending ? "开通中…" : "一键开通样板"}
          </button>
        </form>
        {provisionTemplateMut.isSuccess && (
          <p className="muted-line">
            已开通 {provisionTemplateMut.data.org.name}（{provisionTemplateMut.data.spaces.length} 个 Space）
          </p>
        )}
      </div>
      <div className="pane" style={{ marginBottom: "1rem" }} data-testid="space-rules-panel">
        <div className="pane-title">
          <h2>Space Rules</h2>
          <span>{rulesQuery.data?.builtin ? "builtin" : rulesQuery.data?.source || "—"}</span>
        </div>
        <p className="muted-line">
          Goal 路由关键词与默认 policy；DB 持久化，可与仓库 <code>.ash/rules.yaml</code> 双向同步。
        </p>
        <label className="scenario-picker">
          repoRoot（同步）
          <input value={rulesYamlHint} onChange={(e) => setRulesYamlHint(e.target.value)} data-testid="space-rules-repo-root" />
        </label>
        <textarea
          data-testid="space-rules-editor"
          value={rulesDraft}
          onChange={(e) => setRulesDraft(e.target.value)}
          rows={12}
          style={{ width: "100%", fontFamily: "ui-monospace, monospace", fontSize: "0.85rem" }}
        />
        <div className="toolbar" style={{ gap: "0.5rem", flexWrap: "wrap" }}>
          <button
            className="btn primary"
            type="button"
            data-testid="space-rules-save"
            onClick={() => {
              if (rulesDraftIssue) return;
              const ok = window.confirm("确认将当前规则 JSON 保存到 DB？将覆盖该 Space 现有规则。");
              if (!ok) return;
              rulesMut.mutate();
            }}
            disabled={Boolean(rulesDraftIssue) || rulesMut.isPending}
            title={
              rulesDraftIssue
                ? rulesDraftIssue
                : rulesMut.isPending
                  ? "保存中…"
                  : "将规则保存到 DB（需确认）"
            }
          >
            保存到 DB
          </button>
          <button
            className="btn"
            type="button"
            data-testid="space-rules-import"
            onClick={() => {
              if (!rulesRepoReady) return;
              const ok = window.confirm(
                `确认从「${rulesYamlHint.trim()}」导入规则到 DB？将覆盖该 Space 现有规则。`,
              );
              if (!ok) return;
              importRulesMut.mutate();
            }}
            disabled={!rulesRepoReady || importRulesMut.isPending}
            title={
              !rulesRepoReady
                ? "需要填写 repoRoot"
                : importRulesMut.isPending
                  ? "导入中…"
                  : "从文件导入规则到 DB（需确认）"
            }
          >
            Import 文件→DB
          </button>
          <button
            className="btn"
            type="button"
            data-testid="space-rules-export"
            onClick={() => {
              if (!rulesRepoReady) return;
              const ok = window.confirm(
                `确认将 DB 规则导出到「${rulesYamlHint.trim()}」？可能覆盖仓库内规则文件。`,
              );
              if (!ok) return;
              exportRulesMut.mutate();
            }}
            disabled={!rulesRepoReady || exportRulesMut.isPending}
            title={
              !rulesRepoReady
                ? "需要填写 repoRoot"
                : exportRulesMut.isPending
                  ? "导出中…"
                  : "从 DB 导出规则到文件（需确认）"
            }
          >
            Export DB→文件
          </button>
        </div>
        <label className="scenario-picker" style={{ marginTop: "0.75rem" }}>
          预览 Goal
          <input value={previewGoal} onChange={(e) => setPreviewGoal(e.target.value)} data-testid="space-rules-preview-goal" />
        </label>
        <button
          className="btn"
          type="button"
          data-testid="space-rules-preview"
          onClick={() => {
            if (!rulesRepoReady || !previewGoal.trim()) return;
            previewRulesMut.mutate();
          }}
          disabled={previewRulesMut.isPending || !previewGoal.trim() || !rulesRepoReady}
          title={
            !rulesRepoReady
              ? "需要填写 repoRoot"
              : !previewGoal.trim()
                ? "需要填写预览 Goal"
                : previewRulesMut.isPending
                  ? "预览中…"
                  : "预览路由"
          }
        >
          预览路由
        </button>
        {previewRulesMut.data && (
          <p className="muted-line" data-testid="space-rules-preview-result">
            → {previewRulesMut.data.scenarioName}（{previewRulesMut.data.routeReason}）· policy={previewRulesMut.data.policyProfile}
          </p>
        )}
        {(rulesMut.isError || importRulesMut.isError || exportRulesMut.isError || previewRulesMut.isError) && (
          <p className="error-text" data-testid="space-rules-error">
            {(() => {
              const err =
                (rulesMut.error as Error | null) ||
                (importRulesMut.error as Error | null) ||
                (exportRulesMut.error as Error | null) ||
                (previewRulesMut.error as Error | null);
              return err?.message?.trim() || "Rules 操作失败，请检查 JSON / 路径权限。";
            })()}
          </p>
        )}
      </div>
      <div className="split">
        <div className="pane">
          <div className="pane-title">
            <h2>Organizations</h2>
            <span>{orgsQuery.data?.items?.length ?? 0} 个</span>
          </div>
          <table className="table">
            <thead>
              <tr>
                <th>ID</th>
                <th>Name</th>
                <th>Slug</th>
              </tr>
            </thead>
            <tbody>
              {(orgsQuery.data?.items ?? []).map((org) => (
                <tr key={org.id}>
                  <td>{org.id}</td>
                  <td>{org.name}</td>
                  <td>{org.slug || "-"}</td>
                </tr>
              ))}
              {!orgsQuery.data?.items?.length && (
                <tr className="empty-row">
                  <td colSpan={3}>暂无组织。</td>
                </tr>
              )}
            </tbody>
          </table>
          <form
            className="form"
            onSubmit={submitOrg}
          >
            <label>
              Name
              <input
                name="name"
                required
                placeholder="Product Team"
                value={orgName}
                onChange={(e) => setOrgName(e.target.value)}
                data-testid="space-org-name"
              />
            </label>
            <label>
              Slug
              <input name="slug" placeholder="product" />
            </label>
            <button
              className="btn primary icon-btn"
              type="submit"
              disabled={createOrgMut.isPending || !orgName.trim()}
              title={
                !orgName.trim()
                  ? "需要填写组织名称"
                  : createOrgMut.isPending
                    ? "创建中…"
                    : "创建组织（需确认）"
              }
              data-testid="space-org-create"
            >
              <Plus size={16} strokeWidth={1.8} />
              创建组织
            </button>
          </form>
        </div>
        <div className="pane">
          <div className="pane-title">
            <h2>Spaces</h2>
            <span>{spacesQuery.data?.items?.length ?? 0} 个</span>
          </div>
          <table className="table">
            <thead>
              <tr>
                <th>ID</th>
                <th>Name</th>
                <th>Slug</th>
                <th>Kind</th>
                <th>Scope</th>
              </tr>
            </thead>
            <tbody>
              {(spacesQuery.data?.items ?? []).map((space) => (
                <tr key={space.id}>
                  <td>{space.id}</td>
                  <td>{space.name}</td>
                  <td>{space.slug || "-"}</td>
                  <td data-testid={`space-kind-${space.id}`}>{space.kind || "team"}</td>
                  <td>
                    {activeSpaceId === space.id ? (
                      <span className="status-pill ok">
                        <span className="status-dot" />
                        当前
                      </span>
                    ) : (
                      <button
                        className="btn icon-btn"
                        type="button"
                        onClick={() => activateSpace(space.id)}
                        disabled={loginMut.isPending}
                        title={loginMut.isPending ? "激活中…" : `切换到空间 ${space.name || space.id}`}
                      >
                        <KeyRound size={14} strokeWidth={1.8} />
                        激活
                      </button>
                    )}
                  </td>
                </tr>
              ))}
              {!spacesQuery.data?.items?.length && (
                <tr className="empty-row">
                  <td colSpan={5}>暂无空间。</td>
                </tr>
              )}
            </tbody>
          </table>
          <form
            className="form"
            onSubmit={submitSpace}
          >
            <label>
              Org
              <select key={firstOrgId} name="orgId" required defaultValue={firstOrgId}>
                <option value="" disabled>
                  选择组织
                </option>
                {(orgsQuery.data?.items ?? []).map((org) => (
                  <option key={org.id} value={org.id}>
                    {org.name}
                  </option>
                ))}
              </select>
            </label>
            <label>
              Name
              <input
                name="name"
                required
                placeholder="Delivery Space"
                value={spaceName}
                onChange={(e) => setSpaceName(e.target.value)}
              />
            </label>
            <label>
              Slug
              <input name="slug" placeholder="delivery" />
            </label>
            <button
              className="btn primary icon-btn"
              type="submit"
              disabled={createSpaceMut.isPending || !firstOrgId || !spaceName.trim()}
              data-testid="space-create"
              title={
                !firstOrgId
                  ? "需要先创建组织"
                  : !spaceName.trim()
                    ? "需要填写空间名称"
                    : createSpaceMut.isPending
                      ? "创建中…"
                      : "创建空间（需确认）"
              }
            >
              <Plus size={16} strokeWidth={1.8} />
              创建空间
            </button>
          </form>
          <div className="pane-title subhead">
            <h3>当前身份</h3>
            <span>{meQuery.data?.role || activeSpaceId}</span>
          </div>
          <pre className="code-block">
            {loginMut.data
              ? JSON.stringify(loginMut.data, null, 2)
              : JSON.stringify(meQuery.data ?? { space: activeSpaceId }, null, 2)}
          </pre>
        </div>
      </div>
      <div className="split">
        <div className="pane">
          <div className="pane-title">
            <h2>Roles</h2>
            <span>{rolesQuery.data?.items?.length ?? 0} 个</span>
          </div>
          <table className="table">
            <thead>
              <tr>
                <th>ID</th>
                <th>Name</th>
                <th>Permissions</th>
              </tr>
            </thead>
            <tbody>
              {(rolesQuery.data?.items ?? []).map((role) => (
                <tr key={role.id}>
                  <td>{role.id}</td>
                  <td>{role.name}</td>
                  <td>{role.permissions || "[]"}</td>
                </tr>
              ))}
              {!rolesQuery.data?.items?.length && (
                <tr className="empty-row">
                  <td colSpan={3}>暂无角色。</td>
                </tr>
              )}
            </tbody>
          </table>
          <form className="form" onSubmit={submitRole}>
            <label>
              Org
              <select key={activeOrgId} name="orgId" required defaultValue={activeOrgId}>
                <option value="" disabled>
                  选择组织
                </option>
                {(orgsQuery.data?.items ?? []).map((org) => (
                  <option key={org.id} value={org.id}>
                    {org.name}
                  </option>
                ))}
              </select>
            </label>
            <label>
              Name
              <input
                name="name"
                required
                placeholder="delivery-runner"
                value={roleName}
                onChange={(e) => setRoleName(e.target.value)}
              />
            </label>
            <label>
              Permissions
              <textarea name="permissions" rows={3} placeholder="run:create, artifact:read" />
            </label>
            <button
              className="btn primary icon-btn"
              type="submit"
              disabled={createRoleMut.isPending || !activeOrgId || !roleName.trim()}
              data-testid="space-role-create"
              title={
                !activeOrgId
                  ? "需要先创建或选择组织"
                  : !roleName.trim()
                    ? "需要填写角色名称"
                    : createRoleMut.isPending
                      ? "创建中…"
                      : "创建角色（需确认）"
              }
            >
              <ShieldCheck size={16} strokeWidth={1.8} />
              创建角色
            </button>
          </form>
        </div>
        <div className="pane">
          <div className="pane-title">
            <h2>Members</h2>
            <span>{membersQuery.data?.items?.length ?? 0} 个</span>
          </div>
          <table className="table">
            <thead>
              <tr>
                <th>ID</th>
                <th>User</th>
                <th>Role</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              {(membersQuery.data?.items ?? []).map((member) => (
                <tr key={member.id}>
                  <td>{member.id}</td>
                  <td>{member.userId}</td>
                  <td>{member.roleId}</td>
                  <td>{member.status}</td>
                </tr>
              ))}
              {!membersQuery.data?.items?.length && (
                <tr className="empty-row">
                  <td colSpan={4}>暂无成员。</td>
                </tr>
              )}
            </tbody>
          </table>
          <form className="form" onSubmit={submitMember}>
            <label>
              User ID
              <input name="userId" placeholder="user_delivery_member" />
            </label>
            <label>
              Email
              <input name="email" type="email" autoComplete="username" placeholder="member@example.com" />
            </label>
            <label>
              Display Name
              <input name="displayName" placeholder="Delivery Member" />
            </label>
            <label>
              Password
              <input name="password" type="password" minLength={8} autoComplete="new-password" placeholder="temporary password" />
            </label>
            <label>
              Role
              <select name="roleId" required defaultValue="">
                <option value="" disabled>
                  选择角色
                </option>
                {(rolesQuery.data?.items ?? []).map((role) => (
                  <option key={role.id} value={role.id}>
                    {role.name}
                  </option>
                ))}
              </select>
            </label>
            <button
              className="btn primary icon-btn"
              type="submit"
              disabled={createMemberMut.isPending || !canManageActiveSpace || !rolesQuery.data?.items?.length}
              title={
                !canManageActiveSpace
                  ? "本地 space 不可添加成员，请切换到托管 space"
                  : !rolesQuery.data?.items?.length
                    ? "需要先创建角色"
                    : createMemberMut.isPending
                      ? "添加中…"
                      : "添加成员（需确认）"
              }
            >
              <UsersRound size={16} strokeWidth={1.8} />
              添加成员
            </button>
          </form>
        </div>
      </div>

      <div className="pane">
        <div className="pane-title">
          <h2>权限矩阵 (M2)</h2>
           <span>{matrixQuery.data?.builtinRoles?.length ?? 0} 内置角色</span>
        </div>
        <p className="muted-line">
          内置 RBAC 与场景 × 角色工具策略。运行创建时会记录 <code>actorRole</code>，工具链执行前按场景矩阵校验。
        </p>
        {matrixQuery.isError && (
          <p className="error-text">{(matrixQuery.error as Error).message}</p>
        )}
        <div className="split tr2-grid">
          <div>
            <div className="pane-title subhead">
              <h3>内置角色</h3>
            </div>
            <table className="table compact">
              <thead>
                <tr>
                  <th>角色</th>
                  <th>权限</th>
                </tr>
              </thead>
              <tbody>
                {(matrixQuery.data?.builtinRoles ?? []).map((role) => (
                  <tr key={role.name}>
                    <td title={role.name}>
                      {role.label} <code>{role.name}</code>
                    </td>
                    <td>
                      <code className="evidence-snippet">{(role.permissions ?? []).join(", ")}</code>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <div>
            <div className="pane-title subhead">
              <h3>场景工具策略</h3>
            </div>
            {(matrixQuery.data?.scenarioTools ?? []).map((row) => (
              <details key={row.scenarioKey} className="raw-report">
                <summary>
                  {row.scenario}@{row.version}
                </summary>
                <pre className="code-block compact">{JSON.stringify(row.toolMatrix, null, 2)}</pre>
              </details>
            ))}
            {!matrixQuery.data?.scenarioTools?.length && (
              <p className="muted-line">暂无场景策略（创建空间后会自动种子三场景）。</p>
            )}
          </div>
        </div>
        <div className="pane-title subhead">
          <h3>场景策略编辑</h3>
          <span>{(scopesQuery.data?.items ?? []).filter((s) => s.resourceType === "scenario").length} 条</span>
        </div>
        {(scopesQuery.data?.items ?? [])
          .filter((scope) => scope.resourceType === "scenario")
          .map((scope) => {
            const draft = policyDrafts[scope.id] ?? scope.policyJson;
            return (
              <details key={scope.id} className="raw-report">
                <summary>{scope.resourceId}</summary>
                <textarea
                  className="code-block editable"
                  rows={8}
                  value={draft}
                  onChange={(event) =>
                    setPolicyDrafts((prev) => ({ ...prev, [scope.id]: event.target.value }))
                  }
                />
                <button
                  className="btn icon-btn"
                  type="button"
                  data-testid={`space-scope-save-${scope.id}`}
                  disabled={updateScopeMut.isPending || draft === scope.policyJson}
                  title={
                    draft === scope.policyJson
                      ? "策略未修改"
                      : updateScopeMut.isPending
                        ? "保存中…"
                        : "保存场景工具策略（需确认）"
                  }
                  onClick={() => {
                    const ok = window.confirm(
                      `确认保存场景「${scope.resourceId}」的工具策略？`,
                    );
                    if (!ok) return;
                    updateScopeMut.mutate({ scopeId: scope.id, policyJson: draft });
                  }}
                >
                  保存策略
                </button>
              </details>
            );
          })}
        {!canManageActiveSpace && (
          <p className="muted-line">选择非 local 空间后可编辑场景工具策略。</p>
        )}
        <Link to="/compliance" className="inline-link">
          在合规控制台查看资源作用域 →
        </Link>
      </div>

      <RegistryAssetsPanel spaceId={activeSpaceId} />
    </section>
  );
}
