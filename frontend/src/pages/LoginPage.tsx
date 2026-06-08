import { useMutation } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { KeyRound, LogIn, RadioTower } from "lucide-react";
import { type FormEvent, useState } from "react";
import { devLogin, login } from "@/modules/platform/api/platform.api";
import { setAuthSession } from "@/services/http/client";

export function LoginPage() {
  const navigate = useNavigate();
  const [mode, setMode] = useState<"password" | "dev">("password");

  const passwordMut = useMutation({
    mutationFn: login,
    onSuccess: async (data) => {
      setAuthSession(data.token, data.space.id);
      await navigate({ to: "/runs", replace: true });
    },
  });

  const devMut = useMutation({
    mutationFn: (spaceId?: string) => devLogin(spaceId),
    onSuccess: async (data) => {
      setAuthSession(data.token, data.space.id);
      await navigate({ to: "/runs", replace: true });
    },
  });

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const formData = new FormData(event.currentTarget);
    passwordMut.mutate({
      email: String(formData.get("email") || ""),
      password: String(formData.get("password") || ""),
      spaceId: String(formData.get("spaceId") || "") || undefined,
    });
  }

  const error = passwordMut.error?.message || devMut.error?.message;

  return (
    <section className="login-page">
      <div className="login-panel">
        <div className="brand login-brand">
          <span className="brand-mark">A</span>
          <span>
            ASH <span className="muted">控制台</span>
          </span>
        </div>
        <div className="login-copy">
          <h1>登录工作空间</h1>
          <p>进入运行、记忆、审批、CI 和发布治理控制台。</p>
        </div>

        <div className="segmented">
          <button className={mode === "password" ? "active" : ""} type="button" onClick={() => setMode("password")}>
            <LogIn size={15} strokeWidth={1.8} />
            密码
          </button>
          <button className={mode === "dev" ? "active" : ""} type="button" onClick={() => setMode("dev")}>
            <RadioTower size={15} strokeWidth={1.8} />
            Dev
          </button>
        </div>

        {mode === "password" ? (
          <form className="stack-form" onSubmit={submit}>
            <label>
              邮箱或用户 ID
              <input name="email" type="text" autoComplete="username" required />
            </label>
            <label>
              密码
              <input name="password" type="password" autoComplete="current-password" required />
            </label>
            <label>
              空间 ID
              <input name="spaceId" placeholder="local" />
            </label>
            <button className="btn primary" type="submit" disabled={passwordMut.isPending}>
              <LogIn size={16} strokeWidth={1.8} />
              登录
            </button>
          </form>
        ) : (
          <form
            className="stack-form"
            onSubmit={(event) => {
              event.preventDefault();
              const formData = new FormData(event.currentTarget);
              devMut.mutate(String(formData.get("spaceId") || "") || undefined);
            }}
          >
            <label>
              Dev 空间 ID
              <input name="spaceId" placeholder="local" />
            </label>
            <button className="btn primary" type="submit" disabled={devMut.isPending}>
              <KeyRound size={16} strokeWidth={1.8} />
              获取 Dev Token
            </button>
          </form>
        )}

        {error && <p className="error-text">{error}</p>}
      </div>
    </section>
  );
}
