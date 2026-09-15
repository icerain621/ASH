import { useQuery } from "@tanstack/react-query";
import { listToolRiskCatalog } from "@/modules/platform/api/platform.api";

type Props = {
  open: boolean;
  onClose: () => void;
};

/** Agent Chat Tools panel: built-in tool risk catalog (danger defaults). */
export function ToolsRiskPanel({ open, onClose }: Props) {
  const catalogQuery = useQuery({
    queryKey: ["tool-risk-catalog"],
    queryFn: listToolRiskCatalog,
    enabled: open,
  });

  if (!open) return null;

  const items = catalogQuery.data?.items ?? [];

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
          danger 默认需人工批准或场景 allow_dangerous。MCP 工具请用「设置 → MCP」。
          {catalogQuery.data?.docRef ? ` · ${catalogQuery.data.docRef}` : ""}
        </p>
        <ul className="agent-mcp-list" data-testid="agent-tools-list">
          {items.map((tool) => (
            <li
              key={tool.name}
              className={`agent-mcp-row${tool.defaultDeny ? " deny" : ""}`}
              data-testid={`agent-tools-row-${tool.name}`}
            >
              <div>
                <strong title={tool.label}>{tool.name}</strong>
                <span className="muted-line">
                  {tool.risk}
                  {tool.defaultDeny ? " · default deny" : ""}
                </span>
              </div>
            </li>
          ))}
          {!catalogQuery.isLoading && items.length === 0 ? (
            <li className="muted-line">暂无风险目录</li>
          ) : null}
        </ul>
      </div>
    </div>
  );
}
