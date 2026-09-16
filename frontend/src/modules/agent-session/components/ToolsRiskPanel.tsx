import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { listToolRiskCatalog } from "@/modules/platform/api/platform.api";

type Props = {
  open: boolean;
  onClose: () => void;
};

type RiskFilter = "all" | "danger" | "medium" | "safe";

/** Agent Chat Tools panel: built-in tool risk catalog with filter (read-only). */
export function ToolsRiskPanel({ open, onClose }: Props) {
  const [q, setQ] = useState("");
  const [risk, setRisk] = useState<RiskFilter>("all");

  const catalogQuery = useQuery({
    queryKey: ["tool-risk-catalog"],
    queryFn: listToolRiskCatalog,
    enabled: open,
  });

  const items = useMemo(() => {
    const all = catalogQuery.data?.items ?? [];
    const needle = q.trim().toLowerCase();
    return all.filter((tool) => {
      if (risk !== "all" && tool.risk !== risk) return false;
      if (!needle) return true;
      return (
        tool.name.toLowerCase().includes(needle) ||
        (tool.label || "").toLowerCase().includes(needle)
      );
    });
  }, [catalogQuery.data?.items, q, risk]);

  if (!open) return null;

  return (
    <div className="agent-mcp-overlay" data-testid="agent-tools-panel" role="dialog" aria-modal="true">
      <div className="agent-mcp-panel">
        <div className="pane-title">
          <h2>内置工具风险</h2>
          <button type="button" className="btn mini" data-testid="agent-tools-close" onClick={onClose}>
            关闭
          </button>
        </div>
        <p className="muted-line">
          只读目录：danger 默认需人工批准或场景 allow_dangerous。运行时启停需 space/scenario
          策略（尚未开放）。MCP 用「设置 → MCP」。
          {catalogQuery.data?.docRef ? ` · ${catalogQuery.data.docRef}` : ""}
        </p>

        <div className="agent-tools-filters" data-testid="agent-tools-filters">
          <input
            value={q}
            placeholder="搜索工具名"
            data-testid="agent-tools-search"
            onChange={(e) => setQ(e.target.value)}
          />
          <select
            value={risk}
            data-testid="agent-tools-risk-filter"
            onChange={(e) => setRisk(e.target.value as RiskFilter)}
          >
            <option value="all">全部风险</option>
            <option value="danger">danger</option>
            <option value="medium">medium</option>
            <option value="safe">safe</option>
          </select>
        </div>

        <ul className="agent-mcp-list" data-testid="agent-tools-list">
          {items.map((tool) => (
            <li
              key={tool.name}
              className={`agent-mcp-row${tool.defaultDeny ? " deny" : ""}`}
              data-testid={`agent-tools-row-${tool.name}`}
              data-risk={tool.risk}
            >
              <div>
                <strong title={tool.label}>{tool.name}</strong>
                <span className="muted-line">
                  {tool.risk}
                  {tool.defaultDeny ? " · default deny" : ""}
                </span>
              </div>
              <span className={`agent-tools-badge risk-${tool.risk}`}>{tool.risk}</span>
            </li>
          ))}
          {!catalogQuery.isLoading && items.length === 0 ? (
            <li className="muted-line">无匹配工具</li>
          ) : null}
        </ul>
      </div>
    </div>
  );
}
