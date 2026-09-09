import { useMutation, useQuery } from "@tanstack/react-query";
import { Link, useNavigate } from "@tanstack/react-router";
import { KeyRound, LogIn } from "lucide-react";
import { useEffect, useState, type FormEvent } from "react";
import { getReadyz } from "@/modules/health/api/health.api";
import { passwordLogin } from "@/modules/platform/api/platform.api";
import { isConsoleAuthRequiredFlag } from "@/modules/platform/auth/consoleGate";
import { oidcLoginHref, parseLoginHash } from "@/modules/platform/auth/loginHash";
import { setAuthSession } from "@/services/http/client";

export function LoginPage() {
  const navigate = useNavigate();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [spaceId, setSpaceId] = useState("");
  const [hashNote, setHashNote] = useState("");
  const gateQuery = useQuery({
    queryKey: ["console-auth-gate"],
    queryFn: getReadyz,
    staleTime: 60_000,
  });
  const consoleAuthRequired = isConsoleAuthRequiredFlag(gateQuery.data ?? {});

  useEffect(() => {
    const parsed = parseLoginHash(window.location.hash);
    if (!parsed) return;
    setAuthSession(parsed.token, parsed.spaceId, parsed.refreshToken);
    setHashNote("已从 OIDC 回调写入会话");
    window.history.replaceState(null, "", window.location.pathname + window.location.search);
    void navigate({ to: "/space" });
  }, [navigate]);

  const loginMut = useMutation({
    mutationFn: () =>
      passwordLogin({
        email: email.trim(),
        password,
        spaceId: spaceId.trim() || undefined,
      }),
    onSuccess: (data) => {
      setAuthSession(data.token, data.space.id, data.refreshToken);
      void navigate({ to: "/space" });
    },
  });

  function onSubmit(e: FormEvent) {
    e.preventDefault();
    loginMut.mutate();
  }

  return (
    <section className="panel active" data-testid="login-page">
      <div className="page-kicker">
        <LogIn size={17} strokeWidth={1.8} />
        Auth
      </div>
      <div className="page-heading">
        <div>
          <h1>登录</h1>
          <p>
            密码登录或 OIDC（需 <code>ASH_OIDC_ENABLED=1</code>）。
            {!consoleAuthRequired && (
              <>
                {" "}
                也可在{" "}
                <Link to="/space" className="inline-link">
                  空间
                </Link>{" "}
                使用 Dev Token。
              </>
            )}
          </p>
        </div>
      </div>
      {hashNote && <p className="muted-line">{hashNote}</p>}
      {loginMut.error && <p className="error-text">{(loginMut.error as Error).message}</p>}
      <form className="stack-form" onSubmit={onSubmit} data-testid="password-login-form">
        <label>
          邮箱
          <input
            data-testid="login-email"
            type="email"
            autoComplete="username"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
          />
        </label>
        <label>
          密码
          <input
            data-testid="login-password"
            type="password"
            autoComplete="current-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
          />
        </label>
        <label>
          Space ID（可选）
          <input
            data-testid="login-space"
            value={spaceId}
            onChange={(e) => setSpaceId(e.target.value)}
            placeholder="默认取成员首个 Space"
          />
        </label>
        <button className="btn primary icon-btn" type="submit" disabled={loginMut.isPending}>
          <KeyRound size={16} strokeWidth={1.8} />
          {loginMut.isPending ? "登录中…" : "密码登录"}
        </button>
      </form>
      <div className="toolbar" style={{ marginTop: "1rem" }}>
        <a className="btn icon-btn" href={oidcLoginHref()} data-testid="oidc-login-link">
          <LogIn size={16} strokeWidth={1.8} />
          OIDC 登录
        </a>
      </div>
    </section>
  );
}
