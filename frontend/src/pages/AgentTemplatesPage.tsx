import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { Layers3 } from "lucide-react";
import { useState } from "react";
import {
  isTemplateBindable,
  listAgentTemplates,
  submitAgentTemplateCandidate,
  type AgentTemplateManifest,
} from "@/modules/agent-catalog/api/agentCatalog.api";

const emptyForm: AgentTemplateManifest = {
  id: "tpl.custom.",
  version: "1.0.0",
  loop: "react",
  tools: ["read"],
  compaction: "threshold",
  sandbox: "workspace-write",
  maxTurns: 4,
  memory: ["mem.retrieve", "mem.inject"],
};

export function AgentTemplatesPage() {
  const qc = useQueryClient();
  const [form, setForm] = useState<AgentTemplateManifest>(emptyForm);
  const [message, setMessage] = useState("");
  const [bindableOnly, setBindableOnly] = useState(false);

  const listQuery = useQuery({
    queryKey: ["agent-templates"],
    queryFn: listAgentTemplates,
  });

  const submitMut = useMutation({
    mutationFn: () =>
      submitAgentTemplateCandidate({
        ...form,
        tools: form.tools ?? ["read"],
        memory: form.memory ?? ["mem.retrieve", "mem.inject"],
      }),
    onSuccess: (entry) => {
      setMessage(`已提交候选 ${entry.manifest.id}（待评审，不可绑定运行）`);
      void qc.invalidateQueries({ queryKey: ["agent-templates"] });
      void qc.invalidateQueries({ queryKey: ["agent-reviews"] });
    },
    onError: (err) => setMessage((err as Error).message),
  });

  const items = (listQuery.data?.items ?? []).filter((e) => !bindableOnly || isTemplateBindable(e));

  return (
    <section className="panel active" data-testid="agent-templates-page">
      <div className="page-kicker">
        <Layers3 size={17} strokeWidth={1.8} />
        Agent 模板
      </div>
      <div className="page-heading">
        <div>
          <h1>Agent 模板</h1>
          <p className="muted-line">
            内置与自定义模板分开展示批准状态。未批准模板不可绑定运行。评审见{" "}
            <Link to="/reviews" className="inline-link">
              评审管控
            </Link>
            。
          </p>
        </div>
        <label className="toolbar">
          <input
            type="checkbox"
            checked={bindableOnly}
            data-testid="templates-bindable-only"
            onChange={(e) => setBindableOnly(e.target.checked)}
          />
          仅可绑定运行
        </label>
      </div>

      {message ? <p className="muted-line" data-testid="templates-message">{message}</p> : null}

      <div className="pane" data-testid="templates-list">
        <div className="pane-title">
          <h2>目录</h2>
          <span>{items.length} 项</span>
        </div>
        {listQuery.isError ? <p className="error-text">{(listQuery.error as Error).message}</p> : null}
        <table className="data-table">
          <thead>
            <tr>
              <th>ID</th>
              <th>版本</th>
              <th>状态</th>
              <th>可绑定</th>
              <th>maxTurns</th>
            </tr>
          </thead>
          <tbody>
            {items.map((entry) => {
              const bindable = isTemplateBindable(entry);
              return (
                <tr key={entry.manifest.id} data-testid={`template-row-${entry.manifest.id}`}>
                  <td>
                    <code>{entry.manifest.id}</code>
                  </td>
                  <td>{entry.manifest.version}</td>
                  <td data-testid={`template-status-${entry.manifest.id}`}>{entry.status}</td>
                  <td data-testid={`template-bindable-${entry.manifest.id}`}>{bindable ? "是" : "否"}</td>
                  <td>{entry.manifest.maxTurns ?? "—"}</td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>

      <div className="pane" data-testid="templates-submit">
        <div className="pane-title">
          <h2>提交自定义候选</h2>
        </div>
        <div className="toolbar">
          <label>
            ID
            <input
              value={form.id}
              data-testid="template-id-input"
              onChange={(e) => setForm({ ...form, id: e.target.value })}
            />
          </label>
          <label>
            maxTurns
            <input
              type="number"
              min={1}
              value={form.maxTurns ?? 4}
              data-testid="template-maxturns-input"
              onChange={(e) => setForm({ ...form, maxTurns: Number(e.target.value) || 1 })}
            />
          </label>
          <button
            type="button"
            className="btn primary"
            data-testid="template-submit"
            disabled={submitMut.isPending}
            onClick={() => submitMut.mutate()}
          >
            提交候选
          </button>
        </div>
      </div>
    </section>
  );
}
