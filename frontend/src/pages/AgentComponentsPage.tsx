import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { Puzzle } from "lucide-react";
import { useState } from "react";
import {
  listAgentComponents,
  registerAgentComponent,
} from "@/modules/agent-catalog/api/agentCatalog.api";

const KINDS = ["memory", "tool", "loop_policy", "compaction", "reviewer"] as const;

export function AgentComponentsPage() {
  const qc = useQueryClient();
  const [id, setId] = useState("tool.custom.");
  const [kind, setKind] = useState<(typeof KINDS)[number]>("tool");
  const [message, setMessage] = useState("");

  const listQuery = useQuery({
    queryKey: ["agent-components"],
    queryFn: listAgentComponents,
  });

  const registerMut = useMutation({
    mutationFn: () => registerAgentComponent({ id, kind, status: "candidate" }),
    onSuccess: (comp) => {
      setMessage(`已登记候选组件 ${comp.id}（${comp.kind}）`);
      void qc.invalidateQueries({ queryKey: ["agent-components"] });
    },
    onError: (err) => setMessage((err as Error).message),
  });

  const items = listQuery.data?.items ?? [];

  return (
    <section className="panel active" data-testid="agent-components-page">
      <div className="page-kicker">
        <Puzzle size={17} strokeWidth={1.8} />
        Harness / 插件组件
      </div>
      <div className="page-heading">
        <div>
          <h1>组件清单</h1>
          <p className="muted-line">
            Tool / LoopPolicy / Compaction / Reviewer / Memory 与模板分开管理。批准走{" "}
            <Link to="/reviews" className="inline-link">
              评审管控
            </Link>
            ；运行诊断见{" "}
            <Link to="/doctor" className="inline-link">
              诊断
            </Link>
            。
          </p>
        </div>
      </div>

      {message ? <p className="muted-line" data-testid="components-message">{message}</p> : null}

      <div className="pane" data-testid="components-list">
        <div className="pane-title">
          <h2>已登记</h2>
          <span>{items.length} 项</span>
        </div>
        {listQuery.isError ? <p className="error-text">{(listQuery.error as Error).message}</p> : null}
        <table className="data-table">
          <thead>
            <tr>
              <th>ID</th>
              <th>Kind</th>
              <th>状态</th>
            </tr>
          </thead>
          <tbody>
            {items.map((item) => (
              <tr key={item.id} data-testid={`component-row-${item.id}`}>
                <td>
                  <code>{item.id}</code>
                </td>
                <td>{item.kind}</td>
                <td data-testid={`component-status-${item.id}`}>{item.status}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="pane" data-testid="components-register">
        <div className="pane-title">
          <h2>登记候选</h2>
        </div>
        <div className="toolbar">
          <label>
            ID
            <input value={id} data-testid="component-id-input" onChange={(e) => setId(e.target.value)} />
          </label>
          <label>
            Kind
            <select
              value={kind}
              data-testid="component-kind-input"
              onChange={(e) => setKind(e.target.value as (typeof KINDS)[number])}
            >
              {KINDS.map((k) => (
                <option key={k} value={k}>
                  {k}
                </option>
              ))}
            </select>
          </label>
          <button
            type="button"
            className="btn primary"
            data-testid="component-register"
            disabled={registerMut.isPending}
            onClick={() => registerMut.mutate()}
          >
            登记候选
          </button>
        </div>
      </div>
    </section>
  );
}
