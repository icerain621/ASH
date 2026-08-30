import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { Building2, KeyRound, Plus, ShieldCheck, UsersRound } from "lucide-react";
import { useEffect, useState, type FormEvent } from "react";
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
  listOrgTemplates,
  listOrgs,
  listRoles,
  getPermissionMatrix,
  listSpaceMembers,
  listSpaceResourceScopes,
  listSpaces,
  provisionOrgTemplate,
  updateSpaceResourceScope,
} from "@/modules/platform/api/platform.api";
import { getAuthToken, getCurrentSpaceId, setAuthSession } from "@/services/http/client";

export function SpacePage() {
  const qc = useQueryClient();
  const [activeSpaceId, setActiveSpaceId] = useState(getCurrentSpaceId());
  const [templateId, setTemplateId] = useState("small_team");
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
  const firstOrgId = orgsQuery.data?.items[0]?.id ?? "";
  const activeSpace = spacesQuery.data?.items.find((space) => space.id === activeSpaceId);
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
      const document = JSON.parse(rulesDraft || "{}");
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
  useEffect(() => {
    if (rulesQuery.data?.document) {
      setRulesDraft(JSON.stringify(rulesQuery.data.document, null, 2));
    }
  }, [rulesQuery.data]);
  const loginMut = useMutation({
    mutationFn: (spaceId?: string) => devLogin(spaceId),
    onSuccess: async (data) => {
      setAuthSession(data.token, data.space.id);
      setActiveSpaceId(data.space.id);
      await qc.invalidateQueries();
    },
  });
  const createOrgMut = useMutation({
    mutationFn: createOrg,
    onSuccess: async () => {
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
        loginMut.mutate(firstSpace);
      }
    },
  });
  const createSpaceMut = useMutation({
    mutationFn: createSpace,
    onSuccess: async (space) => {
      await qc.invalidateQueries({ queryKey: ["spaces"] });
      loginMut.mutate(space.id);
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
    const formData = new FormData(event.currentTarget);
    createOrgMut.mutate({
      name: String(formData.get("name") || ""),
      slug: String(formData.get("slug") || "") || undefined,
    });
  }

  function submitTemplate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const formData = new FormData(event.currentTarget);
    const selected = String(formData.get("templateId") || templateId);
    provisionTemplateMut.mutate({
      templateId: selected,
      name: String(formData.get("name") || "") || undefined,
      slug: String(formData.get("slug") || "") || undefined,
    });
  }

  function submitSpace(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const formData = new FormData(event.currentTarget);
    createSpaceMut.mutate({
      orgId: String(formData.get("orgId") || ""),
      name: String(formData.get("name") || ""),
      slug: String(formData.get("slug") || "") || undefined,
    });
  }

  function submitRole(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const formData = new FormData(event.currentTarget);
    const orgId = String(formData.get("orgId") || activeOrgId);
    const rawPermissions = String(formData.get("permissions") || "");
    createRoleMut.mutate({
      orgId,
      name: String(formData.get("name") || ""),
      permissions: rawPermissions
        .split(/[\n,]/)
        .map((item) => item.trim())
        .filter(Boolean),
    });
  }

  function submitMember(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const formData = new FormData(event.currentTarget);
    createMemberMut.mutate({
      spaceId: activeSpaceId,
      userId: String(formData.get("userId") || "") || undefined,
      email: String(formData.get("email") || "") || undefined,
      displayName: String(formData.get("displayName") || "") || undefined,
      password: String(formData.get("password") || "") || undefined,
      roleId: String(formData.get("roleId") || ""),
    });
  }

  const selectedTemplate = templatesQuery.data?.items.find((item) => item.id === templateId);
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
          <button className="btn icon-btn" onClick={() => loginMut.mutate(activeSpaceId)} disabled={loginMut.isPending}>
            <KeyRound size={16} strokeWidth={1.8} />
            Dev Token
          </button>
        </div>
      </div>
      {err && <p className="error-text">{err}</p>}
      <div className="pane" style={{ marginBottom: "1rem" }} data-testid="org-templates-panel">
        <div className="pane-title">
          <h2>组织样板（PRD §3）</h2>
          <span>{templatesQuery.data?.items.length ?? 0} 套</span>
        </div>
        <p className="muted-line">一键开通 Org / Space / 角色；标明谁付费、谁决策、谁审批。</p>
        <form className="stack-form" onSubmit={submitTemplate}>
          <label className="scenario-picker">
            样板
            <select
              name="templateId"
              value={templateId}
              onChange={(e) => setTemplateId(e.target.value)}
              disabled={!templatesQuery.data?.items.length || provisionTemplateMut.isPending}
            >
              {(templatesQuery.data?.items ?? []).map((item) => (
                <option key={item.id} value={item.id}>
                  {item.label}
                </option>
              ))}
              {!templatesQuery.data?.items.length && <option value="small_team">小团队</option>}
            </select>
          </label>
          {selectedTemplate && (
            <p className="muted-line">
              付费：{selectedTemplate.payer} · 决策：{selectedTemplate.decisionMaker} · 审批：{selectedTemplate.approver}
            </p>
          )}
          <input name="name" placeholder="组织名称（可空=样板默认）" />
          <input name="slug" placeholder="slug（可空）" />
          <button className="btn primary icon-btn" type="submit" disabled={provisionTemplateMut.isPending}>
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
          <button className="btn primary" type="button" onClick={() => rulesMut.mutate()} disabled={rulesMut.isPending}>
            保存到 DB
          </button>
          <button className="btn" type="button" onClick={() => importRulesMut.mutate()} disabled={importRulesMut.isPending}>
            Import 文件→DB
          </button>
          <button className="btn" type="button" onClick={() => exportRulesMut.mutate()} disabled={exportRulesMut.isPending}>
            Export DB→文件
          </button>
        </div>
        <label className="scenario-picker" style={{ marginTop: "0.75rem" }}>
          预览 Goal
          <input value={previewGoal} onChange={(e) => setPreviewGoal(e.target.value)} data-testid="space-rules-preview-goal" />
        </label>
        <button className="btn" type="button" onClick={() => previewRulesMut.mutate()} disabled={previewRulesMut.isPending}>
          预览路由
        </button>
        {previewRulesMut.data && (
          <p className="muted-line" data-testid="space-rules-preview-result">
            → {previewRulesMut.data.scenarioName}（{previewRulesMut.data.routeReason}）· policy={previewRulesMut.data.policyProfile}
          </p>
        )}
        {(rulesMut.isError || importRulesMut.isError || exportRulesMut.isError || previewRulesMut.isError) && (
          <p className="error-text">Rules 操作失败，请检查 JSON / 路径权限。</p>
        )}
      </div>
      <div className="split">
        <div className="pane">
          <div className="pane-title">
            <h2>Organizations</h2>
            <span>{orgsQuery.data?.items.length ?? 0} 个</span>
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
              {!orgsQuery.data?.items.length && (
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
              <input name="name" required placeholder="Product Team" />
            </label>
            <label>
              Slug
              <input name="slug" placeholder="product" />
            </label>
            <button className="btn primary icon-btn" type="submit" disabled={createOrgMut.isPending}>
              <Plus size={16} strokeWidth={1.8} />
              创建组织
            </button>
          </form>
        </div>
        <div className="pane">
          <div className="pane-title">
            <h2>Spaces</h2>
            <span>{spacesQuery.data?.items.length ?? 0} 个</span>
          </div>
          <table className="table">
            <thead>
              <tr>
                <th>ID</th>
                <th>Name</th>
                <th>Slug</th>
                <th>Scope</th>
              </tr>
            </thead>
            <tbody>
              {(spacesQuery.data?.items ?? []).map((space) => (
                <tr key={space.id}>
                  <td>{space.id}</td>
                  <td>{space.name}</td>
                  <td>{space.slug || "-"}</td>
                  <td>
                    {activeSpaceId === space.id ? (
                      <span className="status-pill ok">
                        <span className="status-dot" />
                        当前
                      </span>
                    ) : (
                      <button className="btn icon-btn" type="button" onClick={() => loginMut.mutate(space.id)} disabled={loginMut.isPending}>
                        <KeyRound size={14} strokeWidth={1.8} />
                        激活
                      </button>
                    )}
                  </td>
                </tr>
              ))}
              {!spacesQuery.data?.items.length && (
                <tr className="empty-row">
                  <td colSpan={4}>暂无空间。</td>
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
              <input name="name" required placeholder="Delivery Space" />
            </label>
            <label>
              Slug
              <input name="slug" placeholder="delivery" />
            </label>
            <button className="btn primary icon-btn" type="submit" disabled={createSpaceMut.isPending || !firstOrgId}>
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
            <span>{rolesQuery.data?.items.length ?? 0} 个</span>
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
              {!rolesQuery.data?.items.length && (
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
              <input name="name" required placeholder="delivery-runner" />
            </label>
            <label>
              Permissions
              <textarea name="permissions" rows={3} placeholder="run:create, artifact:read" />
            </label>
            <button className="btn primary icon-btn" type="submit" disabled={createRoleMut.isPending || !activeOrgId}>
              <ShieldCheck size={16} strokeWidth={1.8} />
              创建角色
            </button>
          </form>
        </div>
        <div className="pane">
          <div className="pane-title">
            <h2>Members</h2>
            <span>{membersQuery.data?.items.length ?? 0} 个</span>
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
              {!membersQuery.data?.items.length && (
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
              disabled={createMemberMut.isPending || !canManageActiveSpace || !rolesQuery.data?.items.length}
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
          <span>{matrixQuery.data?.builtinRoles.length ?? 0} 内置角色</span>
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
                      <code className="evidence-snippet">{role.permissions.join(", ")}</code>
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
            {!matrixQuery.data?.scenarioTools.length && (
              <p className="muted-line">暂无场景策略（创建空间后会自动种子三场景）。</p>
            )}
          </div>
        </div>
        <div className="pane-title subhead">
          <h3>场景策略编辑</h3>
          <span>{scopesQuery.data?.items.filter((s) => s.resourceType === "scenario").length ?? 0} 条</span>
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
                  disabled={updateScopeMut.isPending || draft === scope.policyJson}
                  onClick={() => updateScopeMut.mutate({ scopeId: scope.id, policyJson: draft })}
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
    </section>
  );
}
