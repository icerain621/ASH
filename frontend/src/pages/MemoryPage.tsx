import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { FormEvent, useState } from "react";
import {
  createCandidate,
  listCandidates,
  reviewCandidate,
} from "@/modules/memory/api/memory.api";
import { shortId } from "@/shared/utils/format";

export function MemoryPage() {
  const qc = useQueryClient();
  const [runId, setRunId] = useState("");

  const candidatesQuery = useQuery({
    queryKey: ["memory", "candidates"],
    queryFn: () => listCandidates(50),
  });

  const createMut = useMutation({
    mutationFn: createCandidate,
    onSuccess: () => qc.invalidateQueries({ queryKey: ["memory", "candidates"] }),
  });

  const reviewMut = useMutation({
    mutationFn: ({ id, decision }: { id: string; decision: "approve" | "reject" }) =>
      reviewCandidate(id, {
        decision,
        reason: "reviewed from UI",
        policyProfile: "default",
        reviewerId: "ui",
        ...(runId ? { runId } : {}),
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["memory", "candidates"] }),
  });

  const onSubmit = (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const fd = new FormData(e.currentTarget);
    const layer = String(fd.get("layer") || "L1");
    const body: Record<string, unknown> = {
      layer,
      title: fd.get("title"),
      body: fd.get("body"),
      scopeRepo: "ash",
    };
    const formRunId = String(fd.get("runId") || "");
    if (formRunId) body.runId = formRunId;
    const ref = String(fd.get("evidenceRef") || "");
    if (layer !== "L0" && ref) {
      body.evidence = [{ kind: "file", ref }];
    }
    createMut.mutate(body, {
      onSuccess: () => e.currentTarget.reset(),
    });
  };

  const items = candidatesQuery.data?.items ?? [];

  return (
    <section className="panel active">
      <div className="toolbar">
        <button className="btn" onClick={() => candidatesQuery.refetch()}>
          刷新候选
        </button>
      </div>
      <div className="split">
        <div className="pane">
          <h2>候选列表</h2>
          <table className="table">
            <thead>
              <tr>
                <th>ID</th>
                <th>Layer</th>
                <th>Title</th>
                <th>Status</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {items.map((m) => (
                <tr key={m.id}>
                  <td title={m.id}>{shortId(m.id)}</td>
                  <td>{m.layer}</td>
                  <td>{m.title}</td>
                  <td>{m.status}</td>
                  <td>
                    {m.status === "candidate" && (
                      <>
                        <button
                          className="btn small ok"
                          onClick={() => reviewMut.mutate({ id: m.id, decision: "approve" })}
                        >
                          ✓
                        </button>
                        <button
                          className="btn small err"
                          onClick={() => reviewMut.mutate({ id: m.id, decision: "reject" })}
                        >
                          ✗
                        </button>
                      </>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        <div className="pane">
          <h2>新建候选</h2>
          <form className="form" onSubmit={onSubmit}>
            <label>
              Run ID <span className="muted">(可选，挂 SSE)</span>
              <input name="runId" placeholder="run_..." value={runId} onChange={(e) => setRunId(e.target.value)} />
            </label>
            <label>
              Layer
              <select name="layer" defaultValue="L1">
                <option>L0</option>
                <option>L1</option>
                <option>L2</option>
              </select>
            </label>
            <label>
              Title
              <input name="title" required defaultValue="M0 rule" />
            </label>
            <label>
              Body
              <textarea name="body" rows={3} required defaultValue="Always run tests before merge." />
            </label>
            <label>
              Evidence ref (L1+)
              <input name="evidenceRef" defaultValue="doc/README.md" />
            </label>
            <button type="submit" className="btn primary" disabled={createMut.isPending}>
              提交候选
            </button>
          </form>
        </div>
      </div>
    </section>
  );
}
