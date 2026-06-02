import { NavLink, Outlet } from "react-router-dom";

const tabs = [
  { to: "/runs", label: "Runs" },
  { to: "/memory", label: "Memory" },
  { to: "/doctor", label: "Doctor" },
];

export function AppLayout() {
  return (
    <>
      <header className="header">
        <div className="brand">
          ASH <span className="muted">Console</span>
        </div>
        <nav className="tabs">
          {tabs.map((tab) => (
            <NavLink
              key={tab.to}
              to={tab.to}
              className={({ isActive }) => "tab" + (isActive ? " active" : "")}
            >
              {tab.label}
            </NavLink>
          ))}
        </nav>
        <div className="status">API: /api/v1</div>
      </header>
      <main>
        <Outlet />
      </main>
    </>
  );
}
