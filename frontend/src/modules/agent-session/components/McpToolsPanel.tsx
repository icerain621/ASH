import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import {
  executeMCPTool,
  listMCPTools,
  patchMCPTool,
  registerMCPTool,
  type MCPTool,
} from "@/modules/platform/api/platform.api";

type Props = {
  open: boolean;
  onClose: () => void;
  /** Optional agent session for EW04 permissionMode / allow-list gating. */
  sessionId?: string | null;
};

function riskNeedsConfirm(risk: string) {
  const r = (risk || "").toLowerCase().trim();
  return r !== "low" && r !== "safe";
}

/** Agent Chat first-class MCP tools panel (list / register / enable-disable / try-exec). */
export function McpToolsPanel({ open, onClose, sessionId }: Props) {
  const qc = useQueryClient();
  const [name, setName] = useState("");
  const [server, setServer] = useState("");
  const [risk, setRisk] = useState("medium");
  const [error, setError] = useState("");
  const [execResult, setExecResult] = useState("");

  const toolsQuery = useQuery({
    queryKey: ["mcp-tools"],
    queryFn: listMCPTools,
    enabled: open,
  });

  const registerMut = useMutation({
    mutationFn: () =>
      registerMCPTool({
        name: name.trim(),
        server: server.trim(),
        risk: risk.trim() || "medium",
      }),
    onSuccess: () => {
      setError("");
      setName("");
      setServer("");
      void qc.invalidateQueries({ queryKey: ["mcp-tools"] });
      void qc.invalidateQueries({ queryKey: ["agent-commands"] });
    },
    onError: (e: Error) => setError(e.message),
  });

  const patchMut = useMutation({
    mutationFn: ({ id, status }: { id: string; status: string }) => patchMCPTool(id, { status }),
    onSuccess: () => {
      setError("");
      void qc.invalidateQueries({ queryKey: ["mcp-tools"] });
      void qc.invalidateQueries({ queryKey: ["agent-commands"] });
    },
    onError: (e: Error) => setError(e.message),
  });

  const execMut = useMutation({
    mutationFn: ({ tool, approve }: { tool: MCPTool; approve: boolean }) =>
      executeMCPTool(tool.id, {
        arguments: {},
        sessionId: sessionId || undefined,
        approve,
      }),
    onSuccess: (res) => {
      setError("");
      setExecResult(
        res.ok
          ? `OK · ${res.tool} · ${res.durationMs}ms`
          : `失败 · ${res.error || res.failureClass || "unknown"}`,
      );
    },
    onError: (e: Error) => {
      setExecResult("");
      setError(e.message);
    },
  });

  if (!open) return null;

  const items: MCPTool[] = toolsQuery.data?.items ?? [];

  return (
    <div className="agent-mcp-overlay" data-testid="agent-mcp-panel" role="dialog" aria-modal="true">
      <div className="agent-mcp-panel">
        <div className="pane-title">
          <h2>MCP 工具</h2>
          <button type="button" className="btn mini" data-testid="agent-mcp-close" onClick={onClose}>
            关闭
          </button>
        </div>
        <p className="muted-line">
          登记后可通过 Chat slash `/name` 或「试执行」调用（status=disabled 时不出现在命令目录）。medium+
          风险需确认审批，禁止 YOLO。
        </p>
        {error ? (
          <p className="error-text" data-testid="agent-mcp-error">
            {error}
          </p>
        ) : null}
        {execResult ? (
          <p className="muted-line" data-testid="agent-mcp-exec-result">
            {execResult}
          </p>
        ) : null}

        <form
          className="agent-mcp-form"
          data-testid="agent-mcp-register"
          onSubmit={(e) => {
            e.preventDefault();
            if (!name.trim() || !server.trim()) {
              setError("name 与 server 必填");
              return;
            }
            registerMut.mutate();
          }}
        >
          <input
            placeholder="name"
            value={name}
            data-testid="agent-mcp-name"
            onChange={(e) => setName(e.target.value)}
          />
          <input
            placeholder="server"
            value={server}
            data-testid="agent-mcp-server"
            onChange={(e) => setServer(e.target.value)}
          />
          <select
            value={risk}
            data-testid="agent-mcp-risk"
            onChange={(e) => setRisk(e.target.value)}
          >
            <option value="low">low</option>
            <option value="medium">medium</option>
            <option value="high">high</option>
            <option value="danger">danger</option>
          </select>
          <button
            type="submit"
            className="btn mini ok"
            disabled={registerMut.isPending}
            data-testid="agent-mcp-register-btn"
          >
            登记
          </button>
        </form>

        <ul className="agent-mcp-list" data-testid="agent-mcp-list">
          {items.map((tool) => {
            const disabled = (tool.status || "").toLowerCase() === "disabled";
            return (
              <li key={tool.id} className="agent-mcp-row" data-testid={`agent-mcp-row-${tool.id}`}>
                <div>
                  <strong>{tool.name}</strong>
                  <span className="muted-line">
                    {tool.server} · {tool.risk} · {tool.status}
                  </span>
                </div>
                <div className="agent-mcp-row-actions">
                  <button
                    type="button"
                    className="btn mini"
                    data-testid={`agent-mcp-try-${tool.id}`}
                    disabled={disabled || execMut.isPending}
                    onClick={() => {
                      const needs = riskNeedsConfirm(tool.risk);
                      if (needs) {
                        const ok = window.confirm(
                          `工具「${tool.name}」风险为 ${tool.risk || "medium"}，确认试执行？（非 YOLO，仅本次批准）`,
                        );
                        if (!ok) return;
                      }
                      execMut.mutate({ tool, approve: needs });
                    }}
                  >
                    试执行
                  </button>
                  <button
                    type="button"
                    className="btn mini"
                    data-testid={`agent-mcp-toggle-${tool.id}`}
                    disabled={patchMut.isPending}
                    onClick={() =>
                      patchMut.mutate({
                        id: tool.id,
                        status: disabled ? "registered" : "disabled",
                      })
                    }
                  >
                    {disabled ? "启用" : "停用"}
                  </button>
                </div>
              </li>
            );
          })}
          {!toolsQuery.isLoading && items.length === 0 ? (
            <li className="muted-line">暂无 MCP 工具</li>
          ) : null}
        </ul>
      </div>
    </div>
  );
}
