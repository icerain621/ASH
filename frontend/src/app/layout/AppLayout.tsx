import { Link, Outlet } from "@tanstack/react-router";
import { Activity, Brain, Building2, HeartPulse, MessageSquarePlus, RadioTower, Workflow } from "lucide-react";
import { getCurrentSpaceId } from "@/services/http/client";

const tabs = [
  { to: "/runs", label: "运行", icon: Activity },
  { to: "/memory", label: "记忆", icon: Brain },
  { to: "/automation", label: "自动化", icon: Workflow },
  { to: "/feedback", label: "反馈", icon: MessageSquarePlus },
  { to: "/space", label: "空间", icon: Building2 },
  { to: "/doctor", label: "诊断", icon: HeartPulse },
];

export function AppLayout() {
  const activeSpaceId = getCurrentSpaceId();
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
          /api/v1 · {activeSpaceId}
        </div>
      </header>
      <main>
        <Outlet />
      </main>
    </div>
  );
}
