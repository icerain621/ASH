import { Link, Outlet, useRouterState } from "@tanstack/react-router";
import {
	Activity,
	BarChart3,
	Brain,
	Building2,
	ClipboardCheck,
	ClipboardList,
	Gauge,
	HeartPulse,
	KanbanSquare,
	BookOpen,
	MessageSquarePlus,
	GitBranch,
	RadioTower,
	ShieldCheck,
	ShieldAlert,
	Workflow,
} from "lucide-react";
import { getCurrentSpaceId } from "@/services/http/client";

const tabs = [
	{ to: "/runs", label: "运行", icon: Activity },
	{ to: "/memory", label: "记忆", icon: Brain },
	{ to: "/reviews", label: "评审", icon: ClipboardList },
	{ to: "/quest", label: "Quest", icon: KanbanSquare },
	{ to: "/knowledge", label: "知识", icon: BookOpen },
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
  const activeSpaceId = getCurrentSpaceId();
  const pathname = useRouterState({ select: (s) => s.location.pathname });
  const mobileShell = pathname.startsWith("/m/");

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
