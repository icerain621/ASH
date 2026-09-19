import { Link, Outlet, useRouterState } from "@tanstack/react-router";
import { RadioTower, Settings } from "lucide-react";
import { getCurrentSpaceId } from "@/services/http/client";
import { persistWorkMode, workModeFromPath } from "./workMode";

const workspaceLinks = [{ to: "/space", label: "空间设置", testId: "nav-settings-space" }] as const;

const opsLinks = [
  { to: "/quest", label: "任务板", testId: "nav-settings-quest" },
  { to: "/runs", label: "运行" },
  { to: "/automation", label: "自动化" },
  { to: "/feedback", label: "反馈" },
  { to: "/ci", label: "CI" },
  { to: "/releases", label: "发布" },
  { to: "/compliance", label: "合规" },
  { to: "/scale", label: "规模化" },
  { to: "/doctor", label: "诊断" },
  { to: "/metrics", label: "指标" },
  { to: "/observability", label: "可观测" },
] as const;

export function AppLayout() {
  const activeSpaceId = getCurrentSpaceId();
  const pathname = useRouterState({ select: (s) => s.location.pathname });
  const mobileShell = pathname.startsWith("/m/");
  const workMode = workModeFromPath(pathname);

  if (mobileShell) {
    return (
      <div className="app-shell mobile-shell" data-testid="mobile-shell">
        <main>
          <Outlet />
        </main>
      </div>
    );
  }

  return (
    <div className="app-shell">
      <header className="header">
        <div className="brand">
          <span className="brand-mark">A</span>
          <span>
            ASH <span className="muted">控制台</span>
          </span>
        </div>
        <div
          className="work-mode"
          role="tablist"
          aria-label="三大主题切换"
          data-testid="work-mode-switch"
        >
          <Link
            to="/quest"
            role="tab"
            aria-selected={workMode === "agent"}
            className={workMode === "agent" ? "work-mode-btn active" : "work-mode-btn"}
            data-testid="work-mode-agent"
            onClick={() => persistWorkMode("agent")}
          >
            Agent
          </Link>
          <Link
            to="/memory"
            role="tab"
            aria-selected={workMode === "memory"}
            className={workMode === "memory" ? "work-mode-btn active" : "work-mode-btn"}
            data-testid="work-mode-memory"
            onClick={() => persistWorkMode("memory")}
          >
            记忆
          </Link>
          <Link
            to="/reviews"
            role="tab"
            aria-selected={workMode === "review"}
            className={workMode === "review" ? "work-mode-btn active" : "work-mode-btn"}
            data-testid="work-mode-review"
            onClick={() => persistWorkMode("review")}
          >
            评审管控
          </Link>
        </div>
        <div className="header-actions">
          <details className="nav-dropdown" data-testid="nav-settings">
            <summary
              className="nav-dropdown-trigger nav-dropdown-trigger-icon"
              title="设置"
              aria-label="设置"
            >
              <Settings size={16} strokeWidth={1.8} aria-hidden="true" />
            </summary>
            <div className="nav-dropdown-menu" role="menu">
              <div className="nav-dropdown-meta" data-testid="nav-settings-workspace-hdr">
                账号与工作区
              </div>
              {workspaceLinks.map((item) => (
                <Link
                  key={item.to}
                  to={item.to}
                  className="nav-dropdown-item"
                  role="menuitem"
                  data-testid={item.testId}
                >
                  {item.label}
                </Link>
              ))}
              <div className="nav-dropdown-meta" data-testid="nav-settings-ops-hdr">
                运维与合规
              </div>
              {opsLinks.map((item) => (
                <Link
                  key={item.to}
                  to={item.to}
                  className="nav-dropdown-item"
                  role="menuitem"
                  data-testid={"testId" in item ? item.testId : undefined}
                >
                  {item.label}
                </Link>
              ))}
            </div>
          </details>
          <details className="nav-dropdown" data-testid="nav-account">
            <summary className="nav-dropdown-trigger">账号</summary>
            <div className="nav-dropdown-menu" role="menu">
              <div className="nav-dropdown-meta" data-testid="nav-account-space">
                当前 Space · {activeSpaceId}
              </div>
              <Link to="/login" className="nav-dropdown-item" role="menuitem" data-testid="nav-login">
                登录
              </Link>
              <Link to="/space" className="nav-dropdown-item" role="menuitem" data-testid="nav-space">
                Space 设置
              </Link>
            </div>
          </details>
          <div className="status">
            <RadioTower size={15} strokeWidth={1.8} />
            /api/v1
          </div>
        </div>
      </header>
      <main>
        <Outlet />
      </main>
    </div>
  );
}
