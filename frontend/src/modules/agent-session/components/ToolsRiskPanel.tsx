import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useMemo, useState } from "react";
import { listToolRiskCatalog } from "@/modules/platform/api/platform.api";
import { patchAgentSession, type AgentSessionView } from "@/modules/agent-session/api/session.api";

type Props = {
  open: boolean;
  onClose: () => void;
  session?: AgentSessionView | null;
};

type RiskFilter = "all" | "danger" | "medium" | "safe";

function disabledSet(session?: AgentSessionView | null): Set<string> {
  const fromField = session?.disabledTools ?? [];
  const fromMeta = Array.isArray(session?.meta?.disabledTools)
    ? (session?.meta?.disabledTools as unknown[]).filter((x): x is string => typeof x === "string")
    : [];
  return new Set([...fromField, ...fromMeta].map((s) => s.trim()).filter(Boolean));
}

/** Agent Chat Tools panel: risk catalog + session-scoped enable/disable. */
export function ToolsRiskPanel({ open, onClose, session = null }: Props) {
  const qc = useQueryClient();
  const [q, setQ] = useState("");
  const [risk, setRisk] = useState<RiskFilter>("all");
  const [error, setError] = useState("");

  const catalogQuery = useQuery({
    queryKey: ["tool-risk-catalog"],
    queryFn: listToolRiskCatalog,
    enabled: open,
  });

  const disabled = disabledSet(session);

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

  const toggleMut = useMutation({
    mutationFn: async (toolName: string) => {
      if (!session?.id) throw new Error("先选择会话再切换工具");
      const next = new Set(disabled);
      if (next.has(toolName)) next.delete(toolName);
      else next.add(toolName);
      return patchAgentSession(session.id, { disabledTools: Array.from(next).sort() });
    },
    onSuccess: (updated) => {
      setError("");
      void qc.invalidateQueries({ queryKey: ["agent-session", updated.id] });
      void qc.invalidateQueries({ queryKey: ["agent-sessions"] });
    },
    onError: (e: Error) => setError(e.message),
  });

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
          danger 默认需人工批准或场景 allow_dangerous。会话级停用会在绑定 run 执行时拒绝该工具。MCP 用「设置 →
          MCP」。
          {catalogQuery.data?.docRef ? ` · ${catalogQuery.data.docRef}` : ""}
        </p>
        {!session?.id ? (
          <p className="muted-line" data-testid="agent-tools-need-session">
            选择会话后可启停工具。
          </p>
        ) : null}
        {error ? (
          <p className="error-text" data-testid="agent-tools-error">
            {error}
          </p>
        ) : null}

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
          {items.map((tool) => {
            const off = disabled.has(tool.name);
            return (
              <li
                key={tool.name}
                className={`agent-mcp-row${tool.defaultDeny || off ? " deny" : ""}`}
                data-testid={`agent-tools-row-${tool.name}`}
                data-risk={tool.risk}
                data-disabled={off ? "1" : "0"}
              >
                <div>
                  <strong title={tool.label}>{tool.name}</strong>
                  <span className="muted-line">
                    {tool.risk}
                    {tool.defaultDeny ? " · default deny" : ""}
                    {off ? " · 已停用" : ""}
                  </span>
                </div>
                <div className="agent-tools-row-actions">
                  <span className={`agent-tools-badge risk-${tool.risk}`}>{tool.risk}</span>
                  <button
                    type="button"
                    className="btn mini"
                    data-testid={`agent-tools-toggle-${tool.name}`}
                    disabled={!session?.id || toggleMut.isPending}
                    onClick={() => toggleMut.mutate(tool.name)}
                  >
                    {off ? "启用" : "停用"}
                  </button>
                </div>
              </li>
            );
          })}
          {!catalogQuery.isLoading && items.length === 0 ? (
            <li className="muted-line">无匹配工具</li>
          ) : null}
        </ul>
      </div>
    </div>
  );
}
