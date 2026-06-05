import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { Building2, KeyRound, Plus, ShieldCheck, UsersRound } from "lucide-react";
import { useState, type FormEvent } from "react";
import {
  createOrg,
  createRole,
  createSpace,
  createSpaceMember,
  devLogin,
  getAuthMe,
  listOrgs,
  listRoles,
  getPermissionMatrix,
  listSpaceMembers,
  listSpaceResourceScopes,
  listSpaces,
  updateSpaceResourceScope,
} from "@/modules/platform/api/platform.api";
import { getAuthToken, getCurrentSpaceId, setAuthSession } from "@/services/http/client";

export function SpacePage() {
  const qc = useQueryClient();
  const [activeSpaceId, setActiveSpaceId] = useState(getCurrentSpaceId());
  const orgsQuery = useQuery({
    queryKey: ["orgs"],
    queryFn: listOrgs,
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

  const err =
    orgsQuery.error?.message ||
    spacesQuery.error?.message ||
    rolesQuery.error?.message ||
    membersQuery.error?.message ||
    createOrgMut.error?.message ||
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
