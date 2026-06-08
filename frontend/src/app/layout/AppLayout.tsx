import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link, Navigate, Outlet, useRouterState } from "@tanstack/react-router";
import {
	Activity,
	BarChart3,
	Brain,
	Building2,
	ClipboardCheck,
	Gauge,
	HeartPulse,
	MessageSquarePlus,
	GitBranch,
	LogOut,
	RadioTower,
	Save,
	ShieldCheck,
	ShieldAlert,
	Workflow,
} from "lucide-react";
import { useEffect, useState, type FormEvent } from "react";
import { changePassword, getAuthMe, listSpaces } from "@/modules/platform/api/platform.api";
import {
	AUTH_SESSION_CHANGED,
	clearAuthSession,
	getAuthSession,
	setCurrentSpaceId,
} from "@/services/http/client";

const tabs = [
  { to: "/runs", label: "运行", icon: Activity },
	{ to: "/memory", label: "记忆", icon: Brain },
	{ to: "/automation", label: "自动化", icon: Workflow },
	{ to: "/feedback", label: "反馈", icon: MessageSquarePlus },
	{ to: "/ci", label: "CI", icon: GitBranch },
	{ to: "/metrics", label: "指标", icon: BarChart3 },
	{ to: "/observability", label: "观测", icon: ShieldAlert },
	{ to: "/releases", label: "发布", icon: ClipboardCheck },
	{ to: "/space", label: "空间", icon: Building2 },
  { to: "/compliance", label: "合规", icon: ShieldCheck },
  { to: "/scale", label: "规模化", icon: Gauge },
  { to: "/doctor", label: "诊断", icon: HeartPulse },
];

export function AppLayout() {
  const qc = useQueryClient();
  const pathname = useRouterState({ select: (state) => state.location.pathname });
  const [session, setSession] = useState(getAuthSession);
  const [showPassword, setShowPassword] = useState(false);
  const isLoginRoute = pathname === "/login";

  useEffect(() => {
    const handler = () => setSession(getAuthSession());
    window.addEventListener(AUTH_SESSION_CHANGED, handler);
    return () => window.removeEventListener(AUTH_SESSION_CHANGED, handler);
  }, []);

  const meQuery = useQuery({
    queryKey: ["auth-me", session.spaceId],
    queryFn: getAuthMe,
    enabled: Boolean(session.token) && !isLoginRoute,
  });
  const spacesQuery = useQuery({
    queryKey: ["spaces", "layout"],
    queryFn: listSpaces,
    enabled: Boolean(session.token) && !isLoginRoute,
  });
  const passwordMut = useMutation({
    mutationFn: changePassword,
    onSuccess: async () => {
      setShowPassword(false);
      await qc.invalidateQueries({ queryKey: ["auth-me"] });
    },
  });

  if (isLoginRoute) {
    return <Outlet />;
  }
  if (!session.token) {
    return <Navigate to="/login" replace />;
  }

  async function switchSpace(spaceId: string) {
    setCurrentSpaceId(spaceId);
    setSession(getAuthSession());
    await qc.invalidateQueries();
  }

  async function logout() {
    clearAuthSession();
    setSession(getAuthSession());
    await qc.clear();
  }

  function submitPassword(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const formData = new FormData(event.currentTarget);
    passwordMut.mutate({
      currentPassword: String(formData.get("currentPassword") || ""),
      newPassword: String(formData.get("newPassword") || ""),
    });
  }

  const activeSpaceId = session.spaceId;
  const displayName = meQuery.data?.user.displayName || meQuery.data?.user.email || meQuery.data?.user.id || "User";
  const role = meQuery.data?.role || "viewer";

  return (
    <div className="app-shell">
      <header className="header">
        <div className="brand">
          <span className="brand-mark">A</span>
          <span>
            ASH <span className="muted">控制台</span>
          </span>
        </div>
        <nav className="tabs">
          {tabs.map((tab) => {
            const Icon = tab.icon;
            return (
              <Link
                key={tab.to}
                to={tab.to}
                className="tab"
                activeProps={{ className: "tab active" }}
              >
                <Icon size={16} strokeWidth={1.8} />
                {tab.label}
              </Link>
            );
          })}
        </nav>
        <div className="status">
          <RadioTower size={15} strokeWidth={1.8} />
          /api/v1
          <select
            className="header-select"
            value={activeSpaceId}
            onChange={(event) => void switchSpace(event.target.value)}
          >
            <option value="local">local</option>
            {(spacesQuery.data?.items ?? []).map((space) => (
              <option key={space.id} value={space.id}>
                {space.name || space.id}
              </option>
            ))}
          </select>
          <span className="identity">{displayName} · {role}</span>
          <button className="icon-only" type="button" onClick={() => setShowPassword((value) => !value)} title="修改密码">
            <Save size={15} strokeWidth={1.8} />
          </button>
          <button className="icon-only" type="button" onClick={() => void logout()} title="退出登录">
            <LogOut size={15} strokeWidth={1.8} />
          </button>
        </div>
      </header>
      {showPassword && (
        <div className="header-popover">
          <form className="stack-form compact" onSubmit={submitPassword}>
            <label>
              当前密码
              <input name="currentPassword" type="password" autoComplete="current-password" required />
            </label>
            <label>
              新密码
              <input name="newPassword" type="password" minLength={8} autoComplete="new-password" required />
            </label>
            <button className="btn primary" type="submit" disabled={passwordMut.isPending}>
              <Save size={15} strokeWidth={1.8} />
              保存
            </button>
            {passwordMut.error && <p className="error-text">{passwordMut.error.message}</p>}
          </form>
        </div>
      )}
      <main>
        <Outlet />
      </main>
    </div>
  );
}
